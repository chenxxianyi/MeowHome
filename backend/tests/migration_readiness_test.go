package tests

import (
	"testing"

	"github.com/meowhome/backend/internal/infrastructure/persistence/mysql"
	"github.com/stretchr/testify/require"
)

func TestSchemaReadinessRequiresCurrentMigrationVersion(t *testing.T) {
	db := SetupTestDB(t)

	err := mysql.CheckSchemaVersion(db)
	require.ErrorContains(t, err, "migration history unavailable")

	require.NoError(t, db.Exec(`CREATE TABLE schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`).Error)
	require.NoError(t, db.Exec("INSERT INTO schema_migrations(version) VALUES (?)", "014_agent_patrol_enhancement.up.sql").Error)
	require.ErrorContains(t, mysql.CheckSchemaVersion(db), "behind required version 015")

	require.NoError(t, db.Exec("INSERT INTO schema_migrations(version) VALUES (?)", "015_vaccine_reminder_linkage.up.sql").Error)
	require.NoError(t, mysql.CheckSchemaVersion(db))
}
