package ws

import (
	"sync"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	UserID   string
	TenantID string
	Conn     *websocket.Conn
	Send     chan []byte
	Hub      *Hub
}

type hubAction struct {
	action  string
	client  *Client
	message *Message
}

const (
	actionRegister   = "register"
	actionUnregister = "unregister"
	actionBroadcast  = "broadcast"
	actionShutdown   = "shutdown"
)

type Hub struct {
	clients    map[string]*Client
	userIndex  map[string]map[string]struct{}
	tenantIdx  map[string]map[string]struct{}
	register   chan *hubAction
	unregister chan *hubAction
	broadcast  chan *hubAction
	shutdown   chan struct{}
	log        *log.Helper
	mu         sync.RWMutex
	running    bool
}

func NewHub(logger log.Logger) *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		userIndex:  make(map[string]map[string]struct{}),
		tenantIdx:  make(map[string]map[string]struct{}),
		register:   make(chan *hubAction, 256),
		unregister: make(chan *hubAction, 256),
		broadcast:  make(chan *hubAction, 256),
		shutdown:   make(chan struct{}),
		log:        log.NewHelper(logger),
	}
}

func (h *Hub) Run() {
	h.mu.Lock()
	h.running = true
	h.mu.Unlock()

	h.log.Info("ws hub started")

	for {
		select {
		case action := <-h.register:
			h.handleRegister(action.client)
		case action := <-h.unregister:
			h.handleUnregister(action.client)
		case action := <-h.broadcast:
			h.handleBroadcast(action.message)
		case <-h.shutdown:
			h.handleShutdown()
			h.log.Info("ws hub stopped")
			return
		}
	}
}

func (h *Hub) Register(client *Client) {
	if client.ID == "" {
		client.ID = uuid.New().String()
	}
	h.register <- &hubAction{action: actionRegister, client: client}
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- &hubAction{action: actionUnregister, client: client}
}

func (h *Hub) Broadcast(msg *Message) {
	h.broadcast <- &hubAction{action: actionBroadcast, message: msg}
}

func (h *Hub) SendToUser(userID string, msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientIDs, ok := h.userIndex[userID]
	if !ok {
		return
	}

	data, err := msg.Encode()
	if err != nil {
		h.log.Errorf("failed to encode message for user %s: %v", userID, err)
		return
	}

	for cid := range clientIDs {
		if client, exists := h.clients[cid]; exists {
			select {
			case client.Send <- data:
			default:
				go h.evictClient(client)
			}
		}
	}
}

func (h *Hub) SendToTenant(tenantID string, msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientIDs, ok := h.tenantIdx[tenantID]
	if !ok {
		return
	}

	data, err := msg.Encode()
	if err != nil {
		h.log.Errorf("failed to encode message for tenant %s: %v", tenantID, err)
		return
	}

	for cid := range clientIDs {
		if client, exists := h.clients[cid]; exists {
			select {
			case client.Send <- data:
			default:
				go h.evictClient(client)
			}
		}
	}
}

func (h *Hub) Shutdown() {
	h.shutdown <- struct{}{}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) IsRunning() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.running
}

func (h *Hub) handleRegister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client.ID] = client

	if client.UserID != "" {
		if h.userIndex[client.UserID] == nil {
			h.userIndex[client.UserID] = make(map[string]struct{})
		}
		h.userIndex[client.UserID][client.ID] = struct{}{}
	}

	if client.TenantID != "" {
		if h.tenantIdx[client.TenantID] == nil {
			h.tenantIdx[client.TenantID] = make(map[string]struct{})
		}
		h.tenantIdx[client.TenantID][client.ID] = struct{}{}
	}

	h.log.Infof("client registered: %s (user=%s, tenant=%s), total: %d",
		client.ID, client.UserID, client.TenantID, len(h.clients))
}

func (h *Hub) handleUnregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client.ID]; !ok {
		return
	}

	delete(h.clients, client.ID)

	if client.UserID != "" {
		if ids, ok := h.userIndex[client.UserID]; ok {
			delete(ids, client.ID)
			if len(ids) == 0 {
				delete(h.userIndex, client.UserID)
			}
		}
	}

	if client.TenantID != "" {
		if ids, ok := h.tenantIdx[client.TenantID]; ok {
			delete(ids, client.ID)
			if len(ids) == 0 {
				delete(h.tenantIdx, client.TenantID)
			}
		}
	}

	close(client.Send)

	h.log.Infof("client unregistered: %s (user=%s, tenant=%s), total: %d",
		client.ID, client.UserID, client.TenantID, len(h.clients))
}

func (h *Hub) handleBroadcast(msg *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := msg.Encode()
	if err != nil {
		h.log.Errorf("failed to encode broadcast message: %v", err)
		return
	}

	for _, client := range h.clients {
		select {
		case client.Send <- data:
		default:
			go h.evictClient(client)
		}
	}
}

func (h *Hub) handleShutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.running = false

	for id, client := range h.clients {
		close(client.Send)
		if client.Conn != nil {
			client.Conn.Close()
		}
		delete(h.clients, id)
	}

	h.userIndex = make(map[string]map[string]struct{})
	h.tenantIdx = make(map[string]map[string]struct{})
}

func (h *Hub) evictClient(client *Client) {
	h.Unregister(client)
	if client.Conn != nil {
		client.Conn.Close()
	}
}
