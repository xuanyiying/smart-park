package device

import (
	"context"
	"errors"
	"testing"
	"time"
)

// recordingTransport captures every command an adapter sends.
type recordingTransport struct {
	calls []sentCommand
	err   error
}

type sentCommand struct {
	DeviceID string
	Command  string
	Params   map[string]string
}

func (t *recordingTransport) Send(ctx context.Context, deviceID, command string, params map[string]string) error {
	t.calls = append(t.calls, sentCommand{DeviceID: deviceID, Command: command, Params: params})
	return t.err
}

// TestOpenGatePublishesRealCommand is the core regression guard: the previous adapters
// only printed to stdout and returned nil, so the system recorded gates as opened while
// no command ever reached a device.
func TestOpenGatePublishesRealCommand(t *testing.T) {
	transport := &recordingTransport{}
	adapter := NewHikvisionAdapter("DS-TMG", transport, nil)

	if err := adapter.OpenGate(context.Background(), "GATE001"); err != nil {
		t.Fatalf("OpenGate returned error: %v", err)
	}

	if len(transport.calls) != 1 {
		t.Fatalf("expected exactly one command to be sent, got %d", len(transport.calls))
	}
	if transport.calls[0].DeviceID != "GATE001" {
		t.Errorf("command targeted device %q, want GATE001", transport.calls[0].DeviceID)
	}
	if transport.calls[0].Command != "open_gate" {
		t.Errorf("command was %q, want open_gate", transport.calls[0].Command)
	}
}

func TestCloseGatePublishesRealCommand(t *testing.T) {
	transport := &recordingTransport{}
	adapter := NewDahuaAdapter("", transport, nil)

	if err := adapter.CloseGate(context.Background(), "GATE002"); err != nil {
		t.Fatalf("CloseGate returned error: %v", err)
	}
	if len(transport.calls) != 1 || transport.calls[0].Command != "close_gate" {
		t.Fatalf("expected a close_gate command, got %+v", transport.calls)
	}
}

// TestCommandFailsWithoutTransport locks in the safety property: an adapter that cannot
// reach a device must say so, never report success.
func TestCommandFailsWithoutTransport(t *testing.T) {
	adapter := NewLankaAdapter("", nil, nil)

	err := adapter.OpenGate(context.Background(), "GATE001")
	if err == nil {
		t.Fatal("OpenGate must fail when no transport is configured")
	}
	if !errors.Is(err, ErrNoTransport) {
		t.Errorf("expected ErrNoTransport, got %v", err)
	}
}

func TestCommandRequiresDeviceID(t *testing.T) {
	adapter := NewJieshunAdapter("", &recordingTransport{}, nil)

	if err := adapter.OpenGate(context.Background(), ""); err == nil {
		t.Fatal("OpenGate must fail without a device id")
	}
}

// TestTransportErrorIsPropagated ensures a broker failure surfaces to the caller rather
// than being swallowed as success.
func TestTransportErrorIsPropagated(t *testing.T) {
	transport := &recordingTransport{err: errors.New("broker unavailable")}
	adapter := NewKetuoAdapter("", transport, nil)

	if err := adapter.OpenGate(context.Background(), "GATE001"); err == nil {
		t.Fatal("transport failure must be reported to the caller")
	}
}

// TestGetDeviceStatusReflectsHeartbeat replaces the old behaviour of unconditionally
// answering "online", which hid offline barriers indefinitely.
func TestGetDeviceStatusReflectsHeartbeat(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-30 * time.Second)
	stale := now.Add(-30 * time.Minute)

	tests := []struct {
		name          string
		lastHeartbeat *time.Time
		wantOnline    bool
	}{
		{name: "recent heartbeat", lastHeartbeat: &recent, wantOnline: true},
		{name: "stale heartbeat", lastHeartbeat: &stale, wantOnline: false},
		{name: "never seen", lastHeartbeat: nil, wantOnline: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewRegistryStatusProvider(
				staticStateReader{status: "active", lastHeartbeat: tt.lastHeartbeat},
				5*time.Minute,
				func() time.Time { return now },
			)

			adapter := NewHikvisionAdapter("DS-TMG", &recordingTransport{}, provider)

			status, err := adapter.GetDeviceStatus(context.Background(), "GATE001")
			if err != nil {
				t.Fatalf("GetDeviceStatus returned error: %v", err)
			}
			if status["online"] != tt.wantOnline {
				t.Errorf("online = %v, want %v", status["online"], tt.wantOnline)
			}
			if status["manufacturer"] != "Hikvision" {
				t.Errorf("manufacturer = %v, want Hikvision", status["manufacturer"])
			}
		})
	}
}

func TestGetDeviceStatusFailsWithoutProvider(t *testing.T) {
	adapter := NewDahuaAdapter("", &recordingTransport{}, nil)

	if _, err := adapter.GetDeviceStatus(context.Background(), "CAM001"); err == nil {
		t.Fatal("GetDeviceStatus must not invent a status when no provider is configured")
	}
}

func TestFactoryResolvesVendorAdapters(t *testing.T) {
	factory := NewAdapterFactory(&recordingTransport{}, nil)

	for _, manufacturer := range []string{"Hikvision", "Dahua", "Jieshun", "Ketuo", "Lanka"} {
		adapter, err := factory.GetAdapter(manufacturer, "any-model")
		if err != nil {
			t.Fatalf("GetAdapter(%s) returned error: %v", manufacturer, err)
		}
		if adapter.GetManufacturer() != manufacturer {
			t.Errorf("manufacturer = %q, want %q", adapter.GetManufacturer(), manufacturer)
		}
	}

	if _, err := factory.GetAdapter("UnknownVendor", ""); err == nil {
		t.Error("GetAdapter must fail for an unsupported vendor")
	}
}

func TestSendCommandForwardsParams(t *testing.T) {
	transport := &recordingTransport{}
	adapter := NewJieshunAdapter("", transport, nil)

	result, err := adapter.SendCommand(context.Background(), "DISP001", "display", map[string]interface{}{
		"text": "欢迎光临",
	})
	if err != nil {
		t.Fatalf("SendCommand returned error: %v", err)
	}
	if result["result"] != "accepted" {
		t.Errorf("result = %v, want accepted", result["result"])
	}

	if len(transport.calls) != 1 {
		t.Fatalf("expected one command, got %d", len(transport.calls))
	}
	if transport.calls[0].Params["text"] != "欢迎光临" {
		t.Errorf("params not forwarded: %+v", transport.calls[0].Params)
	}
}

type staticStateReader struct {
	status        string
	lastHeartbeat *time.Time
}

func (s staticStateReader) DeviceState(ctx context.Context, deviceID string) (string, *time.Time, error) {
	return s.status, s.lastHeartbeat, nil
}
