package device

// JieshunAdapter controls Jieshun barrier and camera controllers.
//
// Jieshun uses the neutral command vocabulary on the device bus, so this adapter adds
// no protocol translation of its own. It exists as a named extension point: when a
// Jieshun deployment needs vendor-specific parameters or a different command name,
// override the relevant method here rather than changing the shared adapter.
type JieshunAdapter struct {
	*VendorAdapter
}

// NewJieshunAdapter creates a new JieshunAdapter.
func NewJieshunAdapter(model string, transport CommandTransport, statusReader StatusProvider) *JieshunAdapter {
	return &JieshunAdapter{
		VendorAdapter: newVendorAdapter("Jieshun", model, transport, statusReader),
	}
}
