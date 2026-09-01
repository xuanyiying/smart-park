package device

// LankaAdapter controls Lanka barrier and camera controllers.
//
// Lanka uses the neutral command vocabulary on the device bus, so this adapter adds
// no protocol translation of its own. It exists as a named extension point: when a
// Lanka deployment needs vendor-specific parameters or a different command name,
// override the relevant method here rather than changing the shared adapter.
type LankaAdapter struct {
	*VendorAdapter
}

// NewLankaAdapter creates a new LankaAdapter.
func NewLankaAdapter(model string, transport CommandTransport, statusReader StatusProvider) *LankaAdapter {
	return &LankaAdapter{
		VendorAdapter: newVendorAdapter("Lanka", model, transport, statusReader),
	}
}
