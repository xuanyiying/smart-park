package ws

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func newTestHub() *Hub {
	return NewHub(log.DefaultLogger)
}

func TestHub_RegisterAndUnregister(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		ID:       "client-1",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_RegisterMultipleClients(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 5; i++ {
		client := &Client{
			ID:       string(rune('a' + i)),
			UserID:   "user-1",
			TenantID: "tenant-1",
			Conn:     nil,
			Send:     make(chan []byte, 256),
			Hub:      hub,
		}
		hub.Register(client)
	}

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 5, hub.ClientCount())
}

func TestHub_Broadcast(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	clients := make([]*Client, 3)
	for i := 0; i < 3; i++ {
		clients[i] = &Client{
			ID:       string(rune('a' + i)),
			UserID:   "user-1",
			TenantID: "tenant-1",
			Conn:     nil,
			Send:     make(chan []byte, 256),
			Hub:      hub,
		}
		hub.Register(clients[i])
	}

	time.Sleep(50 * time.Millisecond)

	msg, err := NewMessage(MsgTypeNotification, map[string]string{"text": "hello"})
	assert.NoError(t, err)

	hub.Broadcast(msg)
	time.Sleep(50 * time.Millisecond)

	for _, client := range clients {
		select {
		case data := <-client.Send:
			received, err := DecodeMessage(data)
			assert.NoError(t, err)
			assert.Equal(t, MsgTypeNotification, received.Type)
		default:
			t.Fatal("expected message on client send channel")
		}
	}
}

func TestHub_SendToUser(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	targetClient := &Client{
		ID:       "client-target",
		UserID:   "user-target",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}
	otherClient := &Client{
		ID:       "client-other",
		UserID:   "user-other",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(targetClient)
	hub.Register(otherClient)
	time.Sleep(50 * time.Millisecond)

	msg, err := NewMessageToUser(MsgTypePaymentSuccess, map[string]string{"order": "123"}, "user-target")
	assert.NoError(t, err)

	hub.SendToUser("user-target", msg)
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-targetClient.Send:
		received, err := DecodeMessage(data)
		assert.NoError(t, err)
		assert.Equal(t, MsgTypePaymentSuccess, received.Type)
		assert.Equal(t, "user-target", received.TargetUserID)
	default:
		t.Fatal("expected message on target client")
	}

	select {
	case <-otherClient.Send:
		t.Fatal("should not receive message on other client")
	default:
	}
}

func TestHub_SendToTenant(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	tenantClient1 := &Client{
		ID:       "c1",
		UserID:   "u1",
		TenantID: "tenant-A",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}
	tenantClient2 := &Client{
		ID:       "c2",
		UserID:   "u2",
		TenantID: "tenant-A",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}
	otherTenantClient := &Client{
		ID:       "c3",
		UserID:   "u3",
		TenantID: "tenant-B",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(tenantClient1)
	hub.Register(tenantClient2)
	hub.Register(otherTenantClient)
	time.Sleep(50 * time.Millisecond)

	msg, err := NewMessageToTenant(MsgTypeSystemAlert, map[string]string{"alert": "maintenance"}, "tenant-A")
	assert.NoError(t, err)

	hub.SendToTenant("tenant-A", msg)
	time.Sleep(50 * time.Millisecond)

	for _, c := range []*Client{tenantClient1, tenantClient2} {
		select {
		case data := <-c.Send:
			received, err := DecodeMessage(data)
			assert.NoError(t, err)
			assert.Equal(t, MsgTypeSystemAlert, received.Type)
			assert.Equal(t, "tenant-A", received.TargetTenantID)
		default:
			t.Fatal("expected message on tenant client")
		}
	}

	select {
	case <-otherTenantClient.Send:
		t.Fatal("should not receive message on other tenant client")
	default:
	}
}

func TestHub_Shutdown(t *testing.T) {
	hub := newTestHub()
	go hub.Run()

	time.Sleep(10 * time.Millisecond)

	for i := 0; i < 3; i++ {
		client := &Client{
			ID:       string(rune('a' + i)),
			UserID:   "user-1",
			TenantID: "tenant-1",
			Conn:     nil,
			Send:     make(chan []byte, 256),
			Hub:      hub,
		}
		hub.Register(client)
	}

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, hub.ClientCount())

	hub.Shutdown()
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
	assert.False(t, hub.IsRunning())
}

func TestHub_UnregisterNonExistent(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		ID:       "ghost",
		UserID:   "ghost-user",
		TenantID: "ghost-tenant",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Unregister(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_SendToNonExistentUser(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	msg, err := NewMessage(MsgTypeNotification, "test")
	assert.NoError(t, err)

	hub.SendToUser("nonexistent", msg)
}

func TestHub_SendToNonExistentTenant(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	msg, err := NewMessage(MsgTypeNotification, "test")
	assert.NoError(t, err)

	hub.SendToTenant("nonexistent", msg)
}

func TestHub_RegisterAutoID(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		ID:       "",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	assert.NotEmpty(t, client.ID)
	assert.Equal(t, 1, hub.ClientCount())
}

func TestHub_BroadcastSlowClient(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	slowClient := &Client{
		ID:       "slow",
		UserID:   "user-1",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 1),
		Hub:      hub,
	}
	_ = slowClient

	fastClient := &Client{
		ID:       "fast",
		UserID:   "user-2",
		TenantID: "tenant-1",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(slowClient)
	hub.Register(fastClient)
	time.Sleep(50 * time.Millisecond)

	slowClient.Send <- []byte("fill")

	msg, _ := NewMessage(MsgTypeNotification, "overflow-test")
	hub.Broadcast(msg)
	time.Sleep(100 * time.Millisecond)

	select {
	case <-fastClient.Send:
	default:
		t.Fatal("fast client should receive message")
	}
}

func TestHub_ClientWithNoUserOrTenant(t *testing.T) {
	hub := newTestHub()
	go hub.Run()
	defer hub.Shutdown()

	time.Sleep(10 * time.Millisecond)

	client := &Client{
		ID:       "anon",
		UserID:   "",
		TenantID: "",
		Conn:     nil,
		Send:     make(chan []byte, 256),
		Hub:      hub,
	}

	hub.Register(client)
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 1, hub.ClientCount())

	msg, _ := NewMessage(MsgTypeNotification, "broadcast-to-anon")
	hub.Broadcast(msg)
	time.Sleep(50 * time.Millisecond)

	select {
	case data := <-client.Send:
		received, err := DecodeMessage(data)
		assert.NoError(t, err)
		assert.Equal(t, MsgTypeNotification, received.Type)
	default:
		t.Fatal("anonymous client should receive broadcast")
	}
}

func init() {
	_ = websocket.CloseGoingAway
}
