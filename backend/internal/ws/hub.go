package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"redblue-server/internal/protocol"
)

type Client struct {
	conn    *websocket.Conn
	send    chan []byte
	matchID string

	closeOnce sync.Once
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		_ = c.conn.Close()
		close(c.send)
	})
}

type Hub struct {
	mu sync.Mutex
	// matchID => set of clients
	rooms map[string]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[string]map[*Client]struct{}),
	}
}

func (h *Hub) AddClient(matchID string, conn *websocket.Conn) *Client {
	c := &Client{
		conn: conn,
		send: make(chan []byte, 64),
		matchID: matchID,
	}

	h.mu.Lock()
	if _, ok := h.rooms[matchID]; !ok {
		h.rooms[matchID] = make(map[*Client]struct{})
	}
	h.rooms[matchID][c] = struct{}{}
	h.mu.Unlock()

	go c.writeLoop()
	go c.readLoop(h)
	return c
}

func (h *Hub) RemoveClient(matchID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.rooms[matchID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.rooms, matchID)
		}
	}
}

func (h *Hub) Broadcast(matchID string, msg protocol.WSMessage) {
	h.mu.Lock()
	clients := make([]*Client, 0, len(h.rooms[matchID]))
	for c := range h.rooms[matchID] {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	b, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for _, c := range clients {
		select {
		case c.send <- b:
		default:
			// 缓冲区满，说明客户端处理不及时，直接丢弃旧消息避免阻塞。
		}
	}
}

func (c *Client) readLoop(h *Hub) {
	// 后端无需处理客户端业务消息，但仍要读取以触发连接断开。
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	for {
		// 简单读取，忽略内容
		if _, _, err := c.conn.ReadMessage(); err != nil {
			h.RemoveClient(c.matchID, c)
			c.Close()
			return
		}
	}
}

func (c *Client) writeLoop() {
	// 适当的写入超时，防止某些客户端网络卡顿导致 goroutine 挂死。
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	for msgBytes := range c.send {
		_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := c.conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
			c.Close()
			return
		}
	}
}

