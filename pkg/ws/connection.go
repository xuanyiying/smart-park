package ws

import (
	"net/http"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
	sendBufSize    = 256
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type UpgradeOptions struct {
	UserID   string
	TenantID string
}

func UpgradeHTTP(w http.ResponseWriter, r *http.Request, hub *Hub, opts UpgradeOptions, logger log.Logger) error {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	client := &Client{
		ID:       "",
		UserID:   opts.UserID,
		TenantID: opts.TenantID,
		Conn:     conn,
		Send:     make(chan []byte, sendBufSize),
		Hub:      hub,
	}

	hub.Register(client)

	logHelper := log.NewHelper(logger)
	go readPump(client, hub, logHelper)
	go writePump(client, logHelper)

	return nil
}

func readPump(client *Client, hub *Hub, logger *log.Helper) {
	defer func() {
		hub.Unregister(client)
		client.Conn.Close()
	}()

	client.Conn.SetReadLimit(maxMessageSize)
	client.Conn.SetReadDeadline(time.Now().Add(pongWait))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				logger.Errorf("read error for client %s: %v", client.ID, err)
			}
			break
		}

		msg, err := DecodeMessage(message)
		if err != nil {
			logger.Warnf("invalid message from client %s: %v", client.ID, err)
			continue
		}

		logger.Infof("received message from client %s: type=%s", client.ID, msg.Type)
	}
}

func writePump(client *Client, logger *log.Helper) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Errorf("write error for client %s: %v", client.ID, err)
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Errorf("ping error for client %s: %v", client.ID, err)
				return
			}
		}
	}
}
