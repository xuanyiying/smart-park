package device

// KetuoAdapter controls Ketuo barrier and camera controllers.
//
// Ketuo uses the neutral command vocabulary on the device bus, so this adapter adds
// no protocol translation of its own. It exists as a named extension point: when a
// Ketuo deployment needs vendor-specific parameters or a different command name,
// override the relevant method here rather than changing the shared adapter.
type KetuoAdapter struct {
	*VendorAdapter
}

// NewKetuoAdapter creates a new KetuoAdapter.
func NewKetuoAdapter(model string, transport CommandTransport, statusReader StatusProvider) *KetuoAdapter {
	return &KetuoAdapter{
		VendorAdapter: newVendorAdapter("Ketuo", model, transport, statusReader),
	}
}
