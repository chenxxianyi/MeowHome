package tests

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/meowhome/backend/internal/app"
	"github.com/meowhome/backend/internal/domain/model"
	"github.com/meowhome/backend/internal/domain/repository"
	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
	"github.com/meowhome/backend/internal/infrastructure/scheduler"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func agentTestMessage(familyID, userID string, at time.Time) *model.AgentMessage {
	return &model.AgentMessage{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: at, UpdatedAt: at}, FamilyID: familyID, Role: "assistant", Visibility: "family", Type: "patrol_abnormal", Severity: "warning", Title: "测试提示", Body: "仅根据测试记录提示。", GeneratedAt: at, Model: "rule-engine-v1"}
}

// A synchronized start exercises the real InnoDB locks, not a repository substitute.
func agentConcurrent(n int, fn func(int) error) []error {
	start := make(chan struct{})
	var group sync.WaitGroup
	errors := make([]error, n)
	for i := range errors {
		group.Add(1)
		go func(i int) {
			defer group.Done()
			<-start
			errors[i] = fn(i)
		}(i)
	}
	close(start)
	group.Wait()
	return errors
}

func TestAgentMySQLConcurrentPatrolDedupAndQuota(t *testing.T) {
	db, service, repo, familyID, userID, _ := agentDBFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Millisecond)
	from, to := now.Add(-time.Hour), now.Add(time.Hour)
	duplicateKey := familyID + ":same-rule"
	errors := agentConcurrent(12, func(int) error {
		message := agentTestMessage(familyID, userID, now)
		message.DedupKey = &duplicateKey
		return repo.CreatePatrolMessage(ctx, message, from, to, 3)
	})
	created := 0
	for _, err := range errors {
		if err == nil {
			created++
		} else {
			require.ErrorIs(t, err, repository.ErrDuplicateKey)
		}
	}
	require.Equal(t, 1, created)
	message, err := repo.FindMessageByDedup(ctx, familyID, duplicateKey)
	require.NoError(t, err)
	require.NoError(t, service.DismissMessage(ctx, familyID, userID, message.ID))
	errors = agentConcurrent(12, func(i int) error {
		message := agentTestMessage(familyID, userID, now)
		key := fmt.Sprintf("%s:rule-%d", familyID, i)
		message.DedupKey = &key
		return repo.CreatePatrolMessage(ctx, message, from, to, 3)
	})
	created = 0
	for _, err := range errors {
		if err == nil {
			created++
		} else {
			require.ErrorIs(t, err, repository.ErrDailyLimit)
		}
	}
	require.Equal(t, 2, created, "dismissed messages still count toward the daily limit")
	count, err := repo.CountNonDangerMessages(ctx, familyID, from, to)
	require.NoError(t, err)
	require.EqualValues(t, 3, count)
	// Danger is never suppressed by the ordinary-message quota.
	danger := agentTestMessage(familyID, userID, now)
	dangerKey := familyID + ":danger"
	danger.DedupKey, danger.Severity = &dangerKey, "danger"
	require.NoError(t, repo.CreatePatrolMessage(ctx, danger, from, to, 3))
	stored, err := repo.FindMessageByDedup(ctx, familyID, duplicateKey)
	require.NoError(t, err)
	require.Equal(t, "dismissed", stored.DisplayStatus)
	var total int64
	require.NoError(t, db.Model(&model.AgentMessage{}).Count(&total).Error)
	require.EqualValues(t, 4, total)
}

func TestAgentMySQLConcurrentConfirmAndWriteRollback(t *testing.T) {
	db, service, repo, familyID, userID, catID := agentDBFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	now := time.Now().UTC().Truncate(time.Millisecond)
	target := agentTestMessage(familyID, userID, now)
	require.NoError(t, repo.CreateMessage(ctx, target))
	scheduled := now.Add(2 * time.Hour)
	draft, err := service.EditDraft(ctx, familyID, userID, target.ID, app.AgentReminderEditRequest{Reminder: &app.AgentReminderInput{CatID: catID, Type: "custom", Title: "剪指甲", ScheduledAt: &scheduled, Timezone: "Asia/Shanghai"}})
	require.NoError(t, err)
	for _, tc := range []struct{ name, table, event string }{
		{"reminder", "reminders", "INSERT"},
		{"message", "ai_agent_messages", "UPDATE"},
	} {
		t.Run(tc.name+" failure rolls back", func(t *testing.T) {
			trigger := "agent_test_" + tc.name + "_failure"
			require.NoError(t, db.Exec("CREATE TRIGGER "+trigger+" BEFORE "+tc.event+" ON "+tc.table+" FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='agent test write failure'").Error)
			_, err := service.ConfirmDraft(ctx, familyID, userID, target.ID, draft.Version)
			require.Error(t, err)
			require.NoError(t, db.Exec("DROP TRIGGER "+trigger).Error)
			var count int64
			require.NoError(t, db.Model(&model.Reminder{}).Count(&count).Error)
			require.Zero(t, count)
			require.NoError(t, db.Model(&model.AuditLog{}).Count(&count).Error)
			require.Zero(t, count)
			stored, err := repo.FindMessage(ctx, familyID, target.ID)
			require.NoError(t, err)
			require.Equal(t, "pending", stored.ActionStatus)
			require.Empty(t, stored.ConfirmedReminderID)
		})
	}
	results := make([]*app.AgentConfirmResponse, 12)
	errors := agentConcurrent(len(results), func(i int) error {
		var err error
		results[i], err = service.ConfirmDraft(ctx, familyID, userID, target.ID, draft.Version)
		return err
	})
	for i, err := range errors {
		require.NoError(t, err)
		require.Equal(t, results[0].Reminder.ID, results[i].Reminder.ID)
	}
	var count int64
	require.NoError(t, db.Model(&model.Reminder{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, db.Model(&model.AuditLog{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, service.DismissMessage(ctx, familyID, userID, target.ID))
	require.NoError(t, service.DismissMessage(ctx, familyID, userID, target.ID))
	stored, err := repo.FindMessage(ctx, familyID, target.ID)
	require.NoError(t, err)
	require.Equal(t, "confirmed", stored.ActionStatus)
	require.Equal(t, "dismissed", stored.DisplayStatus)
	reminder, err := mysql.NewReminderRepo(db).FindByID(ctx, results[0].Reminder.ID)
	require.NoError(t, err)
	require.Equal(t, "todo", reminder.State)
}

func TestAgentMySQLStablePaginationAndSoftDeletion(t *testing.T) {
	_, service, repo, familyID, userID, _ := agentDBFixture(t)
	ctx := context.Background()
	at := time.Now().UTC().Truncate(time.Millisecond)
	want := []string{}
	for i := 0; i < 7; i++ {
		message := agentTestMessage(familyID, userID, at)
		if i == 0 {
			message.FamilyID = ulid.Make().String()
		} else if i == 1 {
			message.DeletedAt = &at
		} else if i == 2 {
			message.Visibility, message.SessionID, message.UserID = "private", ulid.Make().String(), userID
		} else {
			want = append(want, message.ID)
		}
		require.NoError(t, repo.CreateMessage(ctx, message))
		if i < 2 {
			_, err := repo.FindMessage(ctx, familyID, message.ID)
			require.ErrorIs(t, err, repository.ErrNotFound)
			_, err = service.GetMessage(ctx, familyID, userID, message.ID)
			require.Error(t, err)
			copy := *message
			copy.Body = "forbidden update"
			require.ErrorIs(t, repo.UpdateMessageIfVersion(ctx, familyID, &copy, 0, ""), repository.ErrConflict)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(want)))
	got, before := []string{}, ""
	for page := 0; page < 4; page++ {
		result, err := service.ListMessages(ctx, familyID, userID, app.AgentMessageListQuery{Before: before, Limit: 2})
		require.NoError(t, err)
		for _, message := range result.Messages {
			got = append(got, message.ID)
		}
		before = result.NextCursor
		if before == "" {
			break
		}
	}
	require.Equal(t, want, got)
	want = nil
	for i := 0; i < 7; i++ {
		session := &model.AgentSession{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: at, UpdatedAt: at}, FamilyID: familyID, UserID: userID, Status: "active"}
		if i == 0 {
			session.FamilyID = ulid.Make().String()
		} else if i == 1 {
			session.DeletedAt = &at
		} else if i == 2 {
			session.UserID = ulid.Make().String()
		} else {
			want = append(want, session.ID)
		}
		require.NoError(t, repo.CreateSession(ctx, session))
		if i < 3 {
			_, err := repo.FindSession(ctx, familyID, userID, session.ID)
			require.ErrorIs(t, err, repository.ErrNotFound)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(want)))
	got, before = nil, ""
	for page := 0; page < 4; page++ {
		result, err := service.ListSessions(ctx, familyID, userID, before, 2)
		require.NoError(t, err)
		for _, session := range result.Sessions {
			got = append(got, session.ID)
		}
		before = result.NextCursor
		if before == "" {
			break
		}
	}
	require.Equal(t, want, got)
}

func agentMigrationFile(t *testing.T, db *gorm.DB, filename string, down bool) {
	t.Helper()
	// The db is only obtained from SetupTestDB. Never run rollback against a configured business name.
	var name string
	require.NoError(t, db.Raw("SELECT DATABASE()").Scan(&name).Error)
	require.Regexp(t, `^meowhome_agent_test_[0-9a-hjkmnp-tv-z]{26}$`, name)
	body, err := loadMigration(filepath.Join("..", "migrations", filename), down)
	require.NoError(t, err)
	require.NoError(t, db.Exec(body).Error, filename)
}

func TestAgentMySQLUpgradeRollbackAndSchema(t *testing.T) {
	db := SetupTestDB(t)
	downFiles := []string{"014_agent_patrol_enhancement", "013_agent_chat_runs", "012_agent_task_progress", "011_agent_message_display_status", "010_agent_reminder_schedule_index", "009_agent_message_linkage", "008_agent_idempotency", "007_agent"}
	for _, migration := range downFiles {
		agentMigrationFile(t, db, migration+".down.sql", true)
	}
	// 007 intentionally preserves the reminder compatibility columns. Remove them
	// only in this owned empty test database to recreate the pre-Agent schema.
	require.NoError(t, db.Exec("ALTER TABLE reminders DROP COLUMN scheduled_at, DROP COLUMN timezone, DROP COLUMN completed_at").Error)
	now := time.Now().UTC().Truncate(time.Millisecond)
	userID, familyID, reminderID, messageID := ulid.Make().String(), ulid.Make().String(), ulid.Make().String(), ulid.Make().String()
	require.NoError(t, db.Create(&model.Family{Base: model.Base{ID: familyID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, Name: "Legacy family", Timezone: "Asia/Shanghai", Currency: "CNY"}).Error)
	require.NoError(t, db.Omit("ScheduledAt", "Timezone", "CompletedAt").Create(&model.Reminder{Base: model.Base{ID: reminderID, CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatID: "both", Type: "custom", Title: "Legacy reminder", TimeLabel: "明天", State: "todo"}).Error)
	agentMigrationFile(t, db, "007_agent.up.sql", false)
	agentMigrationFile(t, db, "008_agent_idempotency.up.sql", false)
	agentMigrationFile(t, db, "009_agent_message_linkage.up.sql", false)
	// Reproduce the old incorrectly shared conversation draft, including an empty
	// client key. The upgrade must repair it without losing the draft or evidence.
	require.NoError(t, db.Table("ai_agent_messages").Create(map[string]any{"id": messageID, "created_by": userID, "family_id": familyID, "session_id": ulid.Make().String(), "user_id": userID, "role": "assistant", "visibility": "family", "type": "reminder_draft", "title": "Legacy draft", "body": "Legacy body", "evidence": "[]", "model": "test", "generated_at": now, "created_at": now, "updated_at": now, "turn_id": ulid.Make().String(), "client_message_id": ""}).Error)
	require.NoError(t, applyMigrations(db, t))
	var reminder model.Reminder
	require.NoError(t, db.First(&reminder, "id = ?", reminderID).Error)
	require.Nil(t, reminder.ScheduledAt)
	require.Equal(t, "明天", reminder.TimeLabel)
	var message model.AgentMessage
	require.NoError(t, db.First(&message, "id = ?", messageID).Error)
	require.Equal(t, "private", message.Visibility)
	require.Nil(t, message.ClientMessageID)
	require.Equal(t, "Legacy body", message.Body)
	require.Equal(t, "[]", message.Evidence)
	type column struct {
		ColumnName string
		Nullable   string
		Length     int
	}
	var columns []column
	require.NoError(t, db.Raw("SELECT COLUMN_NAME AS column_name, IS_NULLABLE AS nullable, COALESCE(CHARACTER_MAXIMUM_LENGTH,0) AS length FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=?", (model.AgentMessage{}).TableName()).Scan(&columns).Error)
	metadata := map[string]column{}
	for _, col := range columns {
		metadata[col.ColumnName] = col
	}
	for _, tc := range []struct {
		name, nullable string
		length         int
	}{{"id", "NO", 26}, {"family_id", "NO", 26}, {"session_id", "YES", 26}, {"dedup_key", "YES", 255}, {"client_message_id", "YES", 128}, {"tool_call_id", "YES", 128}, {"run_status", "NO", 16}, {"run_token", "NO", 26}, {"rule_version", "YES", 32}} {
		require.Contains(t, metadata, tc.name)
		require.Equal(t, tc.nullable, metadata[tc.name].Nullable, tc.name)
		require.Equal(t, tc.length, metadata[tc.name].Length, tc.name)
	}
	require.True(t, db.Migrator().HasTable((model.AgentSession{}).TableName()))
	for _, migration := range downFiles {
		agentMigrationFile(t, db, migration+".down.sql", true)
	}
	require.False(t, db.Migrator().HasTable((model.AgentMessage{}).TableName()))
	require.False(t, db.Migrator().HasTable((model.AgentSession{}).TableName()))
	reminder = model.Reminder{}
	require.NoError(t, db.First(&reminder, "id = ?", reminderID).Error)
	require.Equal(t, "Legacy reminder", reminder.Title)
	var family model.Family
	require.NoError(t, db.First(&family, "id = ?", familyID).Error)
	require.Equal(t, "Legacy family", family.Name)
	require.NoError(t, applyMigrations(db, t))
}

type agentEventFailure struct {
	*app.AgentService
	fail bool
}

func (runner *agentEventFailure) PatrolDangerRecordSystem(ctx context.Context, familyID, recordID string) error {
	if runner.fail {
		runner.fail = false
		return fmt.Errorf("injected event failure")
	}
	return runner.AgentService.PatrolDangerRecordSystem(ctx, familyID, recordID)
}

func TestAgentMySQLSchedulerRestartAndBackdatedEvent(t *testing.T) {
	db, service, _, familyID, userID, catID := agentDBFixture(t)
	ctx := context.Background()
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC)
	progress := mysql.NewAgentTaskProgressRepo(db)
	runner := &agentEventFailure{AgentService: service}
	newScheduler := func() *scheduler.AgentPatrolScheduler {
		s, err := scheduler.NewAgentPatrolScheduler(runner, mysql.NewFamilyRepo(db), mysql.NewDailyRecordRepo(db), progress, true, "08:00,20:00")
		require.NoError(t, err)
		return s
	}
	// A family created after the scheduled window produces no message but still
	// records successful progress for both slots.
	require.NoError(t, newScheduler().RunOnce(ctx, now))
	for _, key := range []string{"patrol:2026-09-27:08:00", "patrol:2026-09-27:20:00", "danger-events"} {
		p, err := progress.Get(ctx, familyID, key)
		require.NoError(t, err)
		require.Equal(t, "success", p.Status)
		require.True(t, p.LastSuccessAt.Equal(now))
	}
	var count int64
	require.NoError(t, db.Model(&model.AgentMessage{}).Count(&count).Error)
	require.Zero(t, count)
	record := &model.DailyRecord{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: now, UpdatedAt: now}, FamilyID: familyID, CatIDs: []string{catID}, RecordType: "vomit", Severity: "danger", OccurredAt: now.AddDate(0, 0, -45), Title: "补录测试", Payload: "{}"}
	require.NoError(t, db.Create(record).Error)
	runner.fail = true
	require.NoError(t, newScheduler().RunOnce(ctx, now.Add(time.Minute)))
	p, err := progress.Get(ctx, familyID, "danger-events")
	require.NoError(t, err)
	require.Equal(t, "failed", p.Status)
	require.Empty(t, p.CursorID, "failed events must remain eligible for replay")
	require.NotNil(t, p.LastFailureAt)
	require.NoError(t, newScheduler().RunOnce(ctx, now.Add(2*time.Minute)))
	p, err = progress.Get(ctx, familyID, "danger-events")
	require.NoError(t, err)
	require.Equal(t, "success", p.Status)
	require.Equal(t, record.ID, p.CursorID)
	require.True(t, p.CursorCreatedAt.Equal(record.CreatedAt))
	var message model.AgentMessage
	require.NoError(t, db.Where("severity = ?", "danger").First(&message).Error)
	require.Equal(t, catID, message.CatID)
	require.Contains(t, message.Evidence, record.ID)
	// Recreate the scheduler, then simulate a crash after publishing an event but
	// before persisting its cursor. The published message remains unique.
	require.NoError(t, newScheduler().RunOnce(ctx, now.Add(3*time.Minute)))
	p.CursorCreatedAt, p.CursorID, p.Status = nil, "", "failed"
	require.NoError(t, progress.Save(ctx, p))
	require.NoError(t, newScheduler().RunOnce(ctx, now.Add(4*time.Minute)))
	require.NoError(t, db.Model(&model.AgentMessage{}).Where("severity = ?", "danger").Count(&count).Error)
	require.EqualValues(t, 1, count)
	_, err = progress.Get(ctx, ulid.Make().String(), "danger-events")
	require.ErrorIs(t, err, repository.ErrNotFound)
}

func TestAgentMySQLScheduledReminderBoundaries(t *testing.T) {
	db, _, _, familyID, userID, catID := agentDBFixture(t)
	ctx := context.Background()
	from := time.Date(2026, 9, 27, 16, 0, 0, 0, time.UTC) // Shanghai local midnight
	to := from.AddDate(0, 0, 7)
	want := []string{}
	for i, tc := range []struct {
		at     *time.Time
		state  string
		delete bool
		other  bool
	}{
		{&from, "todo", false, false},
		{&from, "todo", false, false},
		{&to, "todo", false, false},
		{nil, "todo", false, false},
		{&from, "done", false, false},
		{&from, "todo", true, false},
		{&from, "todo", false, true},
	} {
		reminder := &model.Reminder{Base: model.Base{ID: ulid.Make().String(), CreatedBy: userID, CreatedAt: from, UpdatedAt: from}, FamilyID: familyID, CatID: catID, Type: "vaccine", Title: "测试计划", State: tc.state, ScheduledAt: tc.at, Timezone: "Asia/Shanghai"}
		if tc.delete {
			reminder.DeletedAt = &from
		}
		if tc.other {
			reminder.FamilyID = ulid.Make().String()
		}
		if i < 2 {
			want = append(want, reminder.ID)
		}
		require.NoError(t, db.Create(reminder).Error)
	}
	sort.Strings(want)
	repo := mysql.NewReminderRepo(db)
	q := repository.ReminderScheduleQuery{FamilyID: familyID, State: "todo", Types: []string{"vaccine"}, From: &from, ToExclusive: &to, Limit: 1}
	got := []string{}
	for page := 0; page < 3; page++ {
		rows, err := repo.ListScheduled(ctx, q)
		require.NoError(t, err)
		if len(rows) == 0 {
			break
		}
		got = append(got, rows[0].ID)
		q.CursorAt, q.CursorID = rows[0].ScheduledAt, rows[0].ID
	}
	require.Equal(t, want, got)
}
