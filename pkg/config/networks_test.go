package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNetworkPresets(t *testing.T) {
	tests := []struct {
		name         string
		network      string
		wantChainID  uint64
		wantPoll     time.Duration
		wantL1Chain  uint64
		wantExists   bool
	}{
		{
			name:         "linea-mainnet",
			network:      "linea-mainnet",
			wantChainID:  59144,
			wantPoll:     2 * time.Second,
			wantL1Chain:  1,
			wantExists:   true,
		},
		{
			name:         "linea-sepolia",
			network:      "linea-sepolia",
			wantChainID:  59141,
			wantPoll:     2 * time.Second,
			wantL1Chain:  11155111,
			wantExists:   true,
		},
		{
			name:        "unknown network",
			network:     "ethereum-mainnet",
			wantExists:  false,
		},
		{
			name:        "empty network",
			network:     "",
			wantExists:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset, ok := NetworkPresets[tc.network]

			require.Equal(t, tc.wantExists, ok)

			if !tc.wantExists {
				return
			}

			require.Equal(t, tc.wantChainID, preset.ChainID)
			require.Equal(t, tc.wantPoll, preset.PollInterval)
			require.Equal(t, tc.wantL1Chain, preset.L1ChainID)
			require.NotEmpty(t, preset.DefaultRPC)
			require.NotZero(t, preset.BlockTime)
		})
	}
}

func TestGetNetworkPreset(t *testing.T) {
	tests := []struct {
		name       string
		network    string
		wantOK     bool
		wantChain  uint64
	}{
		{
			name:      "valid mainnet",
			network:   "linea-mainnet",
			wantOK:    true,
			wantChain: 59144,
		},
		{
			name:      "valid sepolia",
			network:   "linea-sepolia",
			wantOK:    true,
			wantChain: 59141,
		},
		{
			name:    "invalid network",
			network: "polygon",
			wantOK:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			preset, ok := GetNetworkPreset(tc.network)

			require.Equal(t, tc.wantOK, ok)

			if tc.wantOK {
				require.Equal(t, tc.wantChain, preset.ChainID)
			}
		})
	}
}

func TestSupportedNetworks(t *testing.T) {
	networks := SupportedNetworks()

	require.Len(t, networks, 2)
	require.Contains(t, networks, "linea-mainnet")
	require.Contains(t, networks, "linea-sepolia")
}

func TestNetworkPresetFields(t *testing.T) {
	mainnet := NetworkPresets["linea-mainnet"]

	require.Equal(t, uint64(59144), mainnet.ChainID)
	require.Equal(t, 2*time.Second, mainnet.PollInterval)
	require.Equal(t, "https://rpc.linea.build", mainnet.DefaultRPC)
	require.Equal(t, 2*time.Second, mainnet.BlockTime)
	require.Equal(t, uint64(1), mainnet.L1ChainID)

	sepolia := NetworkPresets["linea-sepolia"]

	require.Equal(t, uint64(59141), sepolia.ChainID)
	require.Equal(t, 2*time.Second, sepolia.PollInterval)
	require.Equal(t, "https://rpc.sepolia.linea.build", sepolia.DefaultRPC)
	require.Equal(t, 2*time.Second, sepolia.BlockTime)
	require.Equal(t, uint64(11155111), sepolia.L1ChainID)
}

func TestNetworkPresetStruct(t *testing.T) {
	preset := NetworkPreset{
		ChainID:      12345,
		PollInterval: 5 * time.Second,
		DefaultRPC:   "https://example.com/rpc",
		BlockTime:    3 * time.Second,
		L1ChainID:    1,
	}

	require.Equal(t, uint64(12345), preset.ChainID)
	require.Equal(t, 5*time.Second, preset.PollInterval)
	require.Equal(t, "https://example.com/rpc", preset.DefaultRPC)
	require.Equal(t, 3*time.Second, preset.BlockTime)
	require.Equal(t, uint64(1), preset.L1ChainID)
}
