package ws

import (
	"encoding/json"
	"time"
)

const (
	MsgTypePaymentSuccess = "payment_success"
	MsgTypeVehicleEntry   = "vehicle_entry"
	MsgTypeVehicleExit    = "vehicle_exit"
	MsgTypeDeviceStatus   = "device_status"
	MsgTypeSystemAlert    = "system_alert"
	MsgTypeNotification   = "notification"
)

type Message struct {
	Type           string          `json:"type"`
	Data           json.RawMessage `json:"data"`
	TargetUserID   string          `json:"target_user_id,omitempty"`
	TargetTenantID string          `json:"target_tenant_id,omitempty"`
	Timestamp      int64           `json:"timestamp"`
}

func NewMessage(msgType string, data interface{}) (*Message, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:      msgType,
		Data:      raw,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

func NewMessageToUser(msgType string, data interface{}, userID string) (*Message, error) {
	msg, err := NewMessage(msgType, data)
	if err != nil {
		return nil, err
	}
	msg.TargetUserID = userID
	return msg, nil
}

func NewMessageToTenant(msgType string, data interface{}, tenantID string) (*Message, error) {
	msg, err := NewMessage(msgType, data)
	if err != nil {
		return nil, err
	}
	msg.TargetTenantID = tenantID
	return msg, nil
}

func (m *Message) Encode() ([]byte, error) {
	return json.Marshal(m)
}

func DecodeMessage(raw []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *Message) DecodeData(v interface{}) error {
	return json.Unmarshal(m.Data, v)
}
