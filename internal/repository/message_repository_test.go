package repository_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"go-im/internal/model"
	"go-im/internal/repository"
)

func newTestRepo(t *testing.T) *repository.MessageRepository {
	t.Helper()
	// 复用正式的数据库配置解析逻辑，默认走 internal/repository/db.go 内的 DSN。
	db, err := repository.NewDB()
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	if err := db.AutoMigrate(&model.TimelineMessage{}); err != nil {
		t.Fatalf("failed to migrate timeline_message: %v", err)
	}
	return repository.NewMessageRepository(db)
}

func uniqueID(t *testing.T, prefix string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

func TestSaveMessageIncrementsSeqPerConversation(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	convID := uniqueID(t, "conv-seq")
	m1 := &model.TimelineMessage{MsgID: uniqueID(t, "msg"), ConversationID: convID, SenderID: "u1", Content: "hello", MsgType: 1, SendTime: time.Now().UnixMilli()}
	if err := repo.SaveMessage(ctx, m1); err != nil {
		t.Fatalf("SaveMessage 1 failed: %v", err)
	}
	if m1.Seq != 1 {
		t.Fatalf("expected seq=1, got %d", m1.Seq)
	}

	m2 := &model.TimelineMessage{MsgID: uniqueID(t, "msg"), ConversationID: convID, SenderID: "u1", Content: "world", MsgType: 1, SendTime: time.Now().UnixMilli()}
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

	convA := uniqueID(t, "convA")
	convB := uniqueID(t, "convB")
	now := time.Now().UnixMilli()
	a1 := &model.TimelineMessage{MsgID: uniqueID(t, "msgA"), ConversationID: convA, SenderID: "u1", Content: "A1", MsgType: 1, SendTime: now}
	b1 := &model.TimelineMessage{MsgID: uniqueID(t, "msgB"), ConversationID: convB, SenderID: "u2", Content: "B1", MsgType: 1, SendTime: now}
	a2 := &model.TimelineMessage{MsgID: uniqueID(t, "msgA"), ConversationID: convA, SenderID: "u1", Content: "A2", MsgType: 1, SendTime: now}

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

	conv := uniqueID(t, "conv-dup")
	msgID := uniqueID(t, "dup-msg")
	first := &model.TimelineMessage{MsgID: msgID, ConversationID: conv, SenderID: "u1", Content: "once", MsgType: 1, SendTime: time.Now().UnixMilli()}
	if err := repo.SaveMessage(ctx, first); err != nil {
		t.Fatalf("SaveMessage first failed: %v", err)
	}

	second := &model.TimelineMessage{MsgID: msgID, ConversationID: conv, SenderID: "u1", Content: "twice", MsgType: 1, SendTime: time.Now().UnixMilli()}
	err := repo.SaveMessage(ctx, second)
	if !errors.Is(err, repository.ErrDuplicateMsgID) {
		t.Fatalf("expected ErrDuplicateMsgID, got %v", err)
	}

	var count int64
	if err := repo.DB().WithContext(ctx).Model(&model.TimelineMessage{}).Where("msg_id = ?", msgID).Count(&count).Error; err != nil {
		t.Fatalf("count failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row for duplicate msg_id, got %d", count)
	}
}

func TestFindByMsgID(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	msgID := uniqueID(t, "lookup")
	conv := uniqueID(t, "conv-lookup")
	msg := &model.TimelineMessage{MsgID: msgID, ConversationID: conv, SenderID: "u1", Content: "hello", MsgType: 1, SendTime: time.Now().UnixMilli()}
	if err := repo.SaveMessage(ctx, msg); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	found, err := repo.FindByMsgID(ctx, msgID)
	if err != nil {
		t.Fatalf("FindByMsgID failed: %v", err)
	}
	if found.Seq != msg.Seq || found.ConversationID != msg.ConversationID || found.MsgID != msg.MsgID {
		t.Fatalf("unexpected message returned: %+v", found)
	}
}
