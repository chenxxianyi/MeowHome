package app

import (
	"context"
	"github.com/meowhome/backend/internal/domain/model"
	"testing"
)

func TestAgentRemovedMemberCannotRecoverSavedChat(t *testing.T) {
	s, _ := newAgentServiceForTest()
	response, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "查询", ClientMessageID: "key"})
	if err != nil {
		t.Fatal(err)
	}
	s.members = memberMemory{familyID: "fam", userID: "someone-else", role: "member"}
	if _, err := s.GetMessage(context.Background(), "fam", "user", response.Message.ID); err == nil {
		t.Fatal("removed member read saved message")
	}
	if _, err := s.ListSessionMessages(context.Background(), "fam", "user", response.SessionID, "", 20); err == nil {
		t.Fatal("removed member read conversation")
	}
	if _, err := s.Chat(context.Background(), "fam", "user", AgentChatRequest{Message: "查询", ClientMessageID: "key"}); err == nil {
		t.Fatal("removed member recovered answer")
	}
}

func TestAgentDeletedCatAndRecordNeverBecomeNewToolEvidence(t *testing.T) {
	s, _ := newAgentServiceForTest()
	deleted := s.clock()
	s.cats = catMemory{cat: &model.Cat{Base: model.Base{ID: "cat", DeletedAt: &deleted}, FamilyID: "fam", Name: "已删除"}}
	if _, err := s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, "listRecords", `{"cat_id":"cat"}`); err == nil {
		t.Fatal("deleted cat was queried")
	}
	s.cats = catMemory{cat: &model.Cat{Base: model.Base{ID: "cat"}, FamilyID: "fam", Name: "小白"}}
	record := toolRecord("deleted-record", "feeding", `{"amount":10}`, s.clock())
	record.DeletedAt = &deleted
	s.records = &toolRecordRepo{rows: []*model.DailyRecord{record}}
	if result, err := s.ExecuteTool(context.Background(), AgentToolScope{FamilyID: "fam", UserID: "user"}, "listRecords", `{"cat_id":"cat"}`); err == nil || result != nil {
		t.Fatal("deleted record leaked into tool result")
	}
}
