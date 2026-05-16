package ws

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessage_NewMessage(t *testing.T) {
	data := map[string]string{"key": "value"}
	msg, err := NewMessage(MsgTypeNotification, data)
	assert.NoError(t, err)
	assert.Equal(t, MsgTypeNotification, msg.Type)
	assert.NotZero(t, msg.Timestamp)
	assert.Empty(t, msg.TargetUserID)
	assert.Empty(t, msg.TargetTenantID)

	var decoded map[string]string
	err = msg.DecodeData(&decoded)
	assert.NoError(t, err)
	assert.Equal(t, "value", decoded["key"])
}

func TestMessage_NewMessageToUser(t *testing.T) {
	data := map[string]string{"order": "123"}
	msg, err := NewMessageToUser(MsgTypePaymentSuccess, data, "user-1")
	assert.NoError(t, err)
	assert.Equal(t, MsgTypePaymentSuccess, msg.Type)
	assert.Equal(t, "user-1", msg.TargetUserID)
	assert.Empty(t, msg.TargetTenantID)
}

func TestMessage_NewMessageToTenant(t *testing.T) {
	data := map[string]string{"alert": "maintenance"}
	msg, err := NewMessageToTenant(MsgTypeSystemAlert, data, "tenant-A")
	assert.NoError(t, err)
	assert.Equal(t, MsgTypeSystemAlert, msg.Type)
	assert.Equal(t, "tenant-A", msg.TargetTenantID)
	assert.Empty(t, msg.TargetUserID)
}

func TestMessage_EncodeDecode(t *testing.T) {
	original, err := NewMessage(MsgTypeVehicleEntry, map[string]interface{}{
		"plate": "京A12345",
		"lot":   "P1",
	})
	assert.NoError(t, err)

	encoded, err := original.Encode()
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)

	decoded, err := DecodeMessage(encoded)
	assert.NoError(t, err)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Timestamp, decoded.Timestamp)

	var data map[string]interface{}
	err = decoded.DecodeData(&data)
	assert.NoError(t, err)
	assert.Equal(t, "京A12345", data["plate"])
	assert.Equal(t, "P1", data["lot"])
}

func TestMessage_DecodeData(t *testing.T) {
	msg, err := NewMessage(MsgTypeDeviceStatus, map[string]string{
		"device_id": "dev-001",
		"status":    "online",
	})
	assert.NoError(t, err)

	var data map[string]string
	err = msg.DecodeData(&data)
	assert.NoError(t, err)
	assert.Equal(t, "dev-001", data["device_id"])
	assert.Equal(t, "online", data["status"])
}

func TestMessage_DecodeInvalidJSON(t *testing.T) {
	_, err := DecodeMessage([]byte("not json"))
	assert.Error(t, err)
}

func TestMessageTypes(t *testing.T) {
	types := []string{
		MsgTypePaymentSuccess,
		MsgTypeVehicleEntry,
		MsgTypeVehicleExit,
		MsgTypeDeviceStatus,
		MsgTypeSystemAlert,
		MsgTypeNotification,
	}

	for _, msgType := range types {
		msg, err := NewMessage(msgType, nil)
		assert.NoError(t, err)
		assert.Equal(t, msgType, msg.Type)
	}
}

func TestMessage_NilData(t *testing.T) {
	msg, err := NewMessage(MsgTypeNotification, nil)
	assert.NoError(t, err)
	assert.Equal(t, "null", string(msg.Data))
}

func TestMessage_ComplexData(t *testing.T) {
	complexData := map[string]interface{}{
		"string_field": "hello",
		"int_field":    42,
		"bool_field":   true,
		"nested": map[string]string{
			"inner": "value",
		},
		"array": []string{"a", "b", "c"},
	}

	msg, err := NewMessage(MsgTypeSystemAlert, complexData)
	assert.NoError(t, err)

	encoded, err := msg.Encode()
	assert.NoError(t, err)

	decoded, err := DecodeMessage(encoded)
	assert.NoError(t, err)

	var result map[string]interface{}
	err = decoded.DecodeData(&result)
	assert.NoError(t, err)
	assert.Equal(t, "hello", result["string_field"])
}

func TestMessage_Timestamp(t *testing.T) {
	before := json.Number("")
	msg, err := NewMessage(MsgTypeNotification, "test")
	assert.NoError(t, err)
	assert.True(t, msg.Timestamp > 0)

	encoded, _ := msg.Encode()
	decoded, _ := DecodeMessage(encoded)

	var raw map[string]interface{}
	json.Unmarshal(encoded, &raw)
	_ = before

	assert.Equal(t, msg.Timestamp, decoded.Timestamp)
}
