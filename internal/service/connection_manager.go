package service

import (
	"sync"

	"github.com/gorilla/websocket"
)

// ConnectionManager 负责管理所有在线的 WebSocket 连接，使用读写锁保证并发安全。
type ConnectionManager struct {
	mu    sync.RWMutex
	conns map[string]*websocket.Conn
}

// NewConnectionManager 创建一个连接管理器实例。
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		conns: make(map[string]*websocket.Conn),
	}
}

// Add 注册一个新的连接；如果同一用户已存在旧连接，则先关闭旧连接再覆盖。
func (m *ConnectionManager) Add(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if old, ok := m.conns[userID]; ok {
		_ = old.Close()
	}
	m.conns[userID] = conn
}

// Remove 移除并关闭指定用户的连接。
func (m *ConnectionManager) Remove(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if conn, ok := m.conns[userID]; ok {
		_ = conn.Close()
		delete(m.conns, userID)
	}
}

// Get 返回指定用户的连接实例；若不存在则返回 nil。
func (m *ConnectionManager) Get(userID string) *websocket.Conn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conns[userID]
}

// ListIDs 返回当前在线的用户 ID 列表。
func (m *ConnectionManager) ListIDs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ids := make([]string, 0, len(m.conns))
	for id := range m.conns {
		ids = append(ids, id)
	}
	return ids
}
