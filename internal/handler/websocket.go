package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"go-im/internal/model"
	"go-im/internal/service"
)

const (
	readDeadline = 90 * time.Second // 允许心跳丢 2-3 次（30s/跳）
	writeTimeout = 10 * time.Second // 写超时防止阻塞
	readLimit    = int64(4 << 10)   // 单条消息最大 4KB
)

// WebSocketHandler 负责握手、注册连接以及消息读循环。
type WebSocketHandler struct {
	connManager *service.ConnectionManager
	upgrader    websocket.Upgrader
}

// NewWebSocketHandler 创建 Handler，允许注入连接管理器。
func NewWebSocketHandler(connManager *service.ConnectionManager) *WebSocketHandler {
	return &WebSocketHandler{
		connManager: connManager,
		upgrader: websocket.Upgrader{
			// 生产环境需校验 Origin
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// HandleWebSocket 提供给 Gin 的路由函数。
func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id 不能为空"})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("用户 %s 升级 WebSocket 失败: %v", userID, err)
		return
	}

	h.connManager.Add(userID, conn)
	log.Printf("用户 %s 已连接，当前在线: %v", userID, h.connManager.ListIDs())

	// 独立 goroutine 读消息，避免阻塞握手返回
	go h.readLoop(userID, conn)
}

// readLoop 读取客户端消息，先支持心跳，后续扩展业务指令。
func (h *WebSocketHandler) readLoop(userID string, conn *websocket.Conn) {
	defer func() {
		h.connManager.Remove(userID)
		log.Printf("用户 %s 连接关闭", userID)
	}()

	conn.SetReadLimit(readLimit)
	_ = conn.SetReadDeadline(time.Now().Add(readDeadline))
	conn.SetPongHandler(func(string) error {
		// 客户端 Pong 刷新超时
		return conn.SetReadDeadline(time.Now().Add(readDeadline))
	})

	for {
		var packet model.InputPacket
		if err := conn.ReadJSON(&packet); err != nil {
			log.Printf("读取用户 %s 消息失败: %v", userID, err)
			return
		}

		switch packet.Cmd {
		case model.CmdHeartbeat:
			if err := h.writeJSON(conn, model.OutputPacket{Cmd: model.CmdHeartbeat, Code: 0}); err != nil {
				log.Printf("心跳回复失败 user=%s: %v", userID, err)
				return
			}
		default:
			// 预留：登录、聊天、拉取等指令后续接入 service 层
			log.Printf("收到用户 %s 的指令 cmd=%d msg_id=%s", userID, packet.Cmd, packet.MsgId)
		}
	}
}

// writeJSON 统一设置写超时，防止写阻塞。
func (h *WebSocketHandler) writeJSON(conn *websocket.Conn, payload interface{}) error {
	_ = conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return conn.WriteJSON(payload)
}
