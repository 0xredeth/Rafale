package config

import "time"

// NetworkPreset contains network-specific default values.
type NetworkPreset struct {
	// ChainID is the network chain ID.
	ChainID uint64

	// PollInterval is the block polling interval.
	PollInterval time.Duration

	// DefaultRPC is the default public RPC endpoint.
	DefaultRPC string

	// BlockTime is the expected block time.
	BlockTime time.Duration

	// L1ChainID is the L1 chain ID (Ethereum mainnet or Sepolia).
	L1ChainID uint64
}

// NetworkPresets contains all supported network configurations.
var NetworkPresets = map[string]NetworkPreset{
	"linea-mainnet": {
		ChainID:      59144,
		PollInterval: 2 * time.Second,
		DefaultRPC:   "https://rpc.linea.build",
		BlockTime:    2 * time.Second,
		L1ChainID:    1, // Ethereum mainnet
	},
	"linea-sepolia": {
		ChainID:      59141,
		PollInterval: 2 * time.Second,
		DefaultRPC:   "https://rpc.sepolia.linea.build",
		BlockTime:    2 * time.Second,
		L1ChainID:    11155111, // Sepolia
	},
}

// GetNetworkPreset returns the preset for a network name.
//
// Parameters:
//   - network (string): the network name
//
// Returns:
//   - NetworkPreset: the network preset
//   - bool: true if found, false otherwise
func GetNetworkPreset(network string) (NetworkPreset, bool) {
	preset, ok := NetworkPresets[network]
	return preset, ok
}

// SupportedNetworks returns a list of supported network names.
//
// Returns:
//   - []string: list of supported network names
func SupportedNetworks() []string {
	networks := make([]string, 0, len(NetworkPresets))
	for name := range NetworkPresets {
		networks = append(networks, name)
	}
	return networks
}
