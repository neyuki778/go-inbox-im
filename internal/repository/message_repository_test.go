package repository_test

import (
	"context"
	"errors"
	"testing"

	"go-im/internal/model"
	"go-im/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestRepo(t *testing.T) *repository.MessageRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.TimelineMessage{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return repository.NewMessageRepository(db)
}

func TestSaveMessageIncrementsSeqPerConversation(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	m1 := &model.TimelineMessage{MsgID: "m1", ConversationID: "c1", SenderID: "u1", Content: "hello", MsgType: 1, SendTime: 1}
	if err := repo.SaveMessage(ctx, m1); err != nil {
		t.Fatalf("SaveMessage 1 failed: %v", err)
	}
	if m1.Seq != 1 {
		t.Fatalf("expected seq=1, got %d", m1.Seq)
	}

	m2 := &model.TimelineMessage{MsgID: "m2", ConversationID: "c1", SenderID: "u1", Content: "world", MsgType: 1, SendTime: 2}
	if err := repo.SaveMessage(ctx, m2); err != nil {
		t.Fatalf("SaveMessage 2 failed: %v", err)
	}
	if m2.Seq != 2 {
		t.Fatalf("expected seq=2, got %d", m2.Seq)
	}
}

func TestSaveMessageIsolatedSeqAcrossConversations(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	a1 := &model.TimelineMessage{MsgID: "a1", ConversationID: "convA", SenderID: "u1", Content: "A1", MsgType: 1, SendTime: 1}
	b1 := &model.TimelineMessage{MsgID: "b1", ConversationID: "convB", SenderID: "u2", Content: "B1", MsgType: 1, SendTime: 1}
	a2 := &model.TimelineMessage{MsgID: "a2", ConversationID: "convA", SenderID: "u1", Content: "A2", MsgType: 1, SendTime: 2}

	if err := repo.SaveMessage(ctx, a1); err != nil {
		t.Fatalf("SaveMessage a1 failed: %v", err)
	}
	if a1.Seq != 1 {
		t.Fatalf("expected convA seq=1, got %d", a1.Seq)
	}
	if err := repo.SaveMessage(ctx, b1); err != nil {
		t.Fatalf("SaveMessage b1 failed: %v", err)
	}
	if b1.Seq != 1 {
		t.Fatalf("expected convB seq=1, got %d", b1.Seq)
	}
	if err := repo.SaveMessage(ctx, a2); err != nil {
		t.Fatalf("SaveMessage a2 failed: %v", err)
	}
	if a2.Seq != 2 {
		t.Fatalf("expected convA seq=2, got %d", a2.Seq)
	}
}

func TestSaveMessageDuplicateMsgID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	first := &model.TimelineMessage{MsgID: "dup-msg", ConversationID: "c-dup", SenderID: "u1", Content: "once", MsgType: 1, SendTime: 1}
	if err := repo.SaveMessage(ctx, first); err != nil {
		t.Fatalf("SaveMessage first failed: %v", err)
	}

	second := &model.TimelineMessage{MsgID: "dup-msg", ConversationID: "c-dup", SenderID: "u1", Content: "twice", MsgType: 1, SendTime: 2}
	err := repo.SaveMessage(ctx, second)
	if !errors.Is(err, repository.ErrDuplicateMsgID) {
		t.Fatalf("expected ErrDuplicateMsgID, got %v", err)
	}

	var count int64
	if err := repo.DB().WithContext(ctx).Model(&model.TimelineMessage{}).Where("msg_id = ?", "dup-msg").Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row for duplicate msg_id, got %d", count)
	}
}

func TestFindByMsgID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	msg := &model.TimelineMessage{MsgID: "lookup-1", ConversationID: "c-lookup", SenderID: "u1", Content: "hello", MsgType: 1, SendTime: 1}
	if err := repo.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	found, err := repo.FindByMsgID(ctx, "lookup-1")
	if err != nil {
		t.Fatalf("FindByMsgID failed: %v", err)
	}
	if found.Seq != msg.Seq || found.ConversationID != msg.ConversationID || found.MsgID != msg.MsgID {
		t.Fatalf("unexpected message returned: %+v", found)
	}
}
