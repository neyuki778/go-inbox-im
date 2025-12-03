package service_test

import (
	"context"
	"errors"
	"testing"

	"go-im/internal/model"
	"go-im/internal/service"
)

type stubConn struct {
	writes []interface{}
	err    error
}

func (s *stubConn) WriteJSON(v interface{}) error {
	s.writes = append(s.writes, v)
	return s.err
}

type stubConnLookup struct {
	conns map[string]*stubConn
}

func (l *stubConnLookup) Get(userID string) service.ConnWriter {
	if l.conns == nil {
		return nil
	}
	return l.conns[userID]
}

func TestBroadcastSuccessToAllTargets(t *testing.T) {
	lookup := &stubConnLookup{
		conns: map[string]*stubConn{
			"u1": {},
			"u2": {},
		},
	}
	push := service.NewPushService(lookup)
	packet := model.OutputPacket{Cmd: model.CmdChat, Code: 0, MsgId: "m1", Seq: 1}

	if err := push.Broadcast(context.Background(), packet, []string{"u1", "u2"}); err != nil {
		t.Fatalf("Broadcast returned error: %v", err)
	}
	for _, id := range []string{"u1", "u2"} {
		conn := lookup.conns[id]
		if len(conn.writes) != 1 {
			t.Fatalf("expected one write for %s, got %d", id, len(conn.writes))
		}
		if conn.writes[0] != packet {
			t.Fatalf("unexpected payload for %s: %#v", id, conn.writes[0])
		}
	}
}

func TestBroadcastContinuesOnWriteError(t *testing.T) {
	errWrite := errors.New("write failed")
	lookup := &stubConnLookup{
		conns: map[string]*stubConn{
			"ok":   {},
			"fail": {err: errWrite},
		},
	}
	push := service.NewPushService(lookup)
	packet := model.OutputPacket{Cmd: model.CmdChat, Code: 0, MsgId: "m2", Seq: 2}

	err := push.Broadcast(context.Background(), packet, []string{"ok", "fail"})
	if err == nil {
		t.Fatalf("expected error when one connection fails")
	}
	// 成功的连接仍应收到推送
	if len(lookup.conns["ok"].writes) != 1 {
		t.Fatalf("expected ok connection to receive payload")
	}
	// 失败的连接也尝试写入
	if len(lookup.conns["fail"].writes) != 1 {
		t.Fatalf("expected fail connection to be attempted")
	}
}

func TestBroadcastSkipsMissingConnection(t *testing.T) {
	lookup := &stubConnLookup{
		conns: map[string]*stubConn{
			"only": {},
		},
	}
	push := service.NewPushService(lookup)
	packet := model.OutputPacket{Cmd: model.CmdChat, Code: 0, MsgId: "m3", Seq: 3}

	err := push.Broadcast(context.Background(), packet, []string{"only", "absent"})
	if err != nil {
		t.Fatalf("Broadcast should not fail when skipping missing connections, got %v", err)
	}
	if len(lookup.conns["only"].writes) != 1 {
		t.Fatalf("expected present connection to receive payload")
	}
}
