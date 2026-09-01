package device

// HikvisionAdapter controls Hikvision barrier and camera controllers.
//
// Hikvision uses the neutral command vocabulary on the device bus, so this adapter adds
// no protocol translation of its own. It exists as a named extension point: when a
// Hikvision deployment needs vendor-specific parameters or a different command name,
// override the relevant method here rather than changing the shared adapter.
type HikvisionAdapter struct {
	*VendorAdapter
}

// NewHikvisionAdapter creates a new HikvisionAdapter.
func NewHikvisionAdapter(model string, transport CommandTransport, statusReader StatusProvider) *HikvisionAdapter {
	return &HikvisionAdapter{
		VendorAdapter: newVendorAdapter("Hikvision", model, transport, statusReader),
	}
}
