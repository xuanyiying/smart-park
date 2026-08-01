package seata

import "github.com/go-kratos/kratos/v2/log"

// InitSeata initializes the Seata distributed transaction client.
// This is a placeholder implementation - replace with actual Seata Go SDK integration.
func InitSeata(configPath string) error {
	log.Warnf("seata: InitSeata called with config %s, but seata-go SDK is not available. Distributed transactions are disabled.", configPath)
	return nil
}
