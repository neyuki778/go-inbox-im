package model

import "time"

// TimelineMessage 对应 timeline_message 表。
type TimelineMessage struct {
	ID             uint64    `gorm:"primaryKey;autoIncrement"`
	MsgID          string    `gorm:"column:msg_id;size:64;not null;uniqueIndex:uk_msg_id"`
	ConversationID string    `gorm:"column:conversation_id;size:64;not null;uniqueIndex:uk_conv_seq;index:idx_conv_seq"`
	Seq            uint64    `gorm:"column:seq;not null;uniqueIndex:uk_conv_seq;index:idx_conv_seq"`
	SenderID       string    `gorm:"column:sender_id;size:64;not null"`
	Content        string    `gorm:"column:content;size:4096"`
	MsgType        int8      `gorm:"column:msg_type;default:1"`
	Status         int8      `gorm:"column:status;default:0"`
	SendTime       int64     `gorm:"column:send_time;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

// TableName 自定义表名以符合设计文档。
func (TimelineMessage) TableName() string {
	return "timeline_message"
}
