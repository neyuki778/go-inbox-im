package service_test

import (
	"context"
	"testing"
	"time"

	"go-im/internal/model"
	"go-im/internal/repository"
	"go-im/internal/service"

	"gorm.io/gorm"
)

func newPullService(t *testing.T) (*service.PullService, *repository.PullRepository) {
	t.Helper()
	db, err := repository.NewDB()
	if err != nil {
		t.Fatalf("init db: %v", err)
	}
	if err := db.AutoMigrate(&model.TimelineMessage{}); err != nil {
		t.Fatalf("migrate timeline_message: %v", err)
	}
	if err := db.AutoMigrate(&model.UserConversationState{}); err != nil {
		t.Fatalf("migrate user_conversation_state: %v", err)
	}
	repo := repository.NewPullRepository(db)
	return service.NewPullService(repo), repo
}

func seedMessages(t *testing.T, db *gorm.DB, conv string, seqs []int64) {
	t.Helper()
	for _, s := range seqs {
		msg := model.TimelineMessage{
			MsgID:          uniqueID(conv + "-msg"),
			ConversationID: conv,
			Seq:            uint64(s),
			SenderID:       "u1",
			Content:        "c",
			MsgType:        1,
			Status:         1,
			SendTime:       time.Now().UnixMilli(),
		}
		if err := db.Create(&msg).Error; err != nil {
			t.Fatalf("seed message seq=%d: %v", s, err)
		}
	}
}

func TestPullMessagesNoMore(t *testing.T) {
	svc, repo := newPullService(t)
	ctx := context.Background()
	convID := uniqueID("conv1")
	seedMessages(t, repo.DB(), convID, []int64{1, 2, 3})

	res, err := svc.PullMessages(ctx, convID, 0, 50)
	if err != nil {
		t.Fatalf("PullMessages error: %v", err)
	}
	if len(res.Messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(res.Messages))
	}
	if res.HasMore {
		t.Fatalf("expected HasMore=false")
	}
	if res.NextCursorSeq != 3 {
		t.Fatalf("expected NextCursorSeq=3, got %d", res.NextCursorSeq)
	}
}

func TestPullMessagesWithMore(t *testing.T) {
	svc, repo := newPullService(t)
	ctx := context.Background()
	seqs := []int64{5, 6, 7, 8, 9}
	convID := uniqueID("conv2")
	seedMessages(t, repo.DB(), convID, seqs)

	res, err := svc.PullMessages(ctx, convID, 5, 2)
	if err != nil {
		t.Fatalf("PullMessages error: %v", err)
	}
	if len(res.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(res.Messages))
	}
	if !res.HasMore {
		t.Fatalf("expected HasMore=true")
	}
	if res.NextCursorSeq != 7 {
		t.Fatalf("expected NextCursorSeq=7, got %d", res.NextCursorSeq)
	}
}

func TestAckConversationInsertAndUpdate(t *testing.T) {
	svc, repo := newPullService(t)
	ctx := context.Background()
	convID := uniqueID("conv-ack")
	userID := uniqueID("u")

	// 首次插入
	if err := svc.AckConversation(ctx, userID, convID, 10); err != nil {
		t.Fatalf("first ack error: %v", err)
	}
	var ack model.UserConversationState
	if err := repo.DB().WithContext(ctx).First(&ack, "user_id=? AND conversation_id=?", userID, convID).Error; err != nil {
		t.Fatalf("query ack: %v", err)
	}
	if ack.LastAckSeq != 10 {
		t.Fatalf("expected LastAckSeq=10, got %d", ack.LastAckSeq)
	}

	// 回退的 ack 不应降低值
	if err := svc.AckConversation(ctx, userID, convID, 5); err != nil {
		t.Fatalf("rollback ack error: %v", err)
	}
	if err := repo.DB().WithContext(ctx).First(&ack, "user_id=? AND conversation_id=?", userID, convID).Error; err != nil {
		t.Fatalf("query ack: %v", err)
	}
	if ack.LastAckSeq != 10 {
		t.Fatalf("expected LastAckSeq still 10 after rollback, got %d", ack.LastAckSeq)
	}

	// 更大的 ack 更新
	if err := svc.AckConversation(ctx, userID, convID, 15); err != nil {
		t.Fatalf("forward ack error: %v", err)
	}
	if err := repo.DB().WithContext(ctx).First(&ack, "user_id=? AND conversation_id=?", userID, convID).Error; err != nil {
		t.Fatalf("query ack: %v", err)
	}
	if ack.LastAckSeq != 15 {
		t.Fatalf("expected LastAckSeq=15, got %d", ack.LastAckSeq)
	}
}

// helper 生成唯一 ID，避免测试间冲突
func uniqueID(prefix string) string {
	return prefix + "-" + time.Now().Format("20060102-150405.000000000")
}
