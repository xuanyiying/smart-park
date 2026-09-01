package device

// DahuaAdapter controls Dahua barrier and camera controllers.
//
// Dahua uses the neutral command vocabulary on the device bus, so this adapter adds
// no protocol translation of its own. It exists as a named extension point: when a
// Dahua deployment needs vendor-specific parameters or a different command name,
// override the relevant method here rather than changing the shared adapter.
type DahuaAdapter struct {
	*VendorAdapter
}

// NewDahuaAdapter creates a new DahuaAdapter.
func NewDahuaAdapter(model string, transport CommandTransport, statusReader StatusProvider) *DahuaAdapter {
	return &DahuaAdapter{
		VendorAdapter: newVendorAdapter("Dahua", model, transport, statusReader),
	}
}
