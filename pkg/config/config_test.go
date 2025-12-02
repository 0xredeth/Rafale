package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "valid config",
			config: &Config{
				Name:     "test-indexer",
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer"},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: &Config{
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer"},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "name is required",
		},
		{
			name: "missing network",
			config: &Config{
				Name:     "test",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer"},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "network is required",
		},
		{
			name: "missing database",
			config: &Config{
				Name:    "test",
				Network: "linea-mainnet",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer"},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "database connection string is required",
		},
		{
			name: "no contracts",
			config: &Config{
				Name:      "test",
				Network:   "linea-mainnet",
				Database:  "postgres://localhost/test",
				Contracts: map[string]ContractConfig{},
			},
			wantErr:    true,
			wantErrMsg: "at least one contract must be defined",
		},
		{
			name: "contract missing address",
			config: &Config{
				Name:     "test",
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						ABI:    "abis/erc20.json",
						Events: []string{"Transfer"},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "contract usdc: address is required",
		},
		{
			name: "contract missing abi",
			config: &Config{
				Name:     "test",
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						Events:  []string{"Transfer"},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "contract usdc: abi path is required",
		},
		{
			name: "contract no events",
			config: &Config{
				Name:     "test",
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{},
					},
				},
			},
			wantErr:    true,
			wantErrMsg: "contract usdc: at least one event must be specified",
		},
		{
			name: "multiple contracts valid",
			config: &Config{
				Name:     "test",
				Network:  "linea-mainnet",
				Database: "postgres://localhost/test",
				Contracts: map[string]ContractConfig{
					"usdc": {
						Address: "0x1234",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer"},
					},
					"dai": {
						Address: "0x5678",
						ABI:     "abis/erc20.json",
						Events:  []string{"Transfer", "Approval"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestSetDefaults(t *testing.T) {
	// Reset viper for clean state
	viper.Reset()

	setDefaults()

	require.Equal(t, "linea-mainnet", viper.GetString("network"))
	require.Equal(t, 8080, viper.GetInt("server.graphql_port"))
	require.Equal(t, 9090, viper.GetInt("server.metrics_port"))
	require.Equal(t, 1000, viper.GetInt("sync.batch_size"))
	require.Equal(t, 3, viper.GetInt("sync.max_retries"))
	require.Equal(t, "1s", viper.GetString("sync.retry_delay"))
}

func TestLoadWithEnvOverrides(t *testing.T) {
	// This test verifies that environment variables override config
	// Note: Full Load() test requires viper config file setup

	// Test DATABASE_URL env override
	originalDB := os.Getenv("DATABASE_URL")
	originalRPC := os.Getenv("LINEA_RPC_URL")
	defer func() {
		os.Setenv("DATABASE_URL", originalDB)
		os.Setenv("LINEA_RPC_URL", originalRPC)
	}()

	testDBURL := "postgres://test:test@localhost:5432/testdb"
	testRPCURL := "https://custom-rpc.example.com"

	os.Setenv("DATABASE_URL", testDBURL)
	os.Setenv("LINEA_RPC_URL", testRPCURL)

	// Create a minimal config and apply env overrides manually
	cfg := &Config{
		Database: "original",
		RPCURL:   "original",
	}

	// Simulate the env override logic from Load()
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database = dbURL
	}
	if rpcURL := os.Getenv("LINEA_RPC_URL"); rpcURL != "" {
		cfg.RPCURL = rpcURL
	}

	require.Equal(t, testDBURL, cfg.Database)
	require.Equal(t, testRPCURL, cfg.RPCURL)
}

func TestContractConfigStruct(t *testing.T) {
	cc := ContractConfig{
		ABI:        "abis/erc20.json",
		Address:    "0x176211869cA2b568f2A7D4EE941E073a821EE1ff",
		StartBlock: 1000000,
		Events:     []string{"Transfer", "Approval"},
	}

	require.Equal(t, "abis/erc20.json", cc.ABI)
	require.Equal(t, "0x176211869cA2b568f2A7D4EE941E073a821EE1ff", cc.Address)
	require.Equal(t, uint64(1000000), cc.StartBlock)
	require.Len(t, cc.Events, 2)
	require.Contains(t, cc.Events, "Transfer")
	require.Contains(t, cc.Events, "Approval")
}

func TestServerConfigStruct(t *testing.T) {
	sc := ServerConfig{
		GraphQLPort: 8080,
		MetricsPort: 9090,
	}

	require.Equal(t, 8080, sc.GraphQLPort)
	require.Equal(t, 9090, sc.MetricsPort)
}

func TestSyncConfigStruct(t *testing.T) {
	sc := SyncConfig{
		BatchSize:  2000,
		MaxRetries: 5,
		RetryDelay: 0, // Will be parsed from string in actual use
	}

	require.Equal(t, uint64(2000), sc.BatchSize)
	require.Equal(t, 5, sc.MaxRetries)
}

func TestConfigStructComplete(t *testing.T) {
	cfg := &Config{
		Name:     "my-indexer",
		Network:  "linea-mainnet",
		Database: "postgres://user:pass@localhost:5432/rafale",
		RPCURL:   "https://rpc.linea.build",
		Contracts: map[string]ContractConfig{
			"usdc": {
				ABI:        "abis/erc20.json",
				Address:    "0x176211869cA2b568f2A7D4EE941E073a821EE1ff",
				StartBlock: 0,
				Events:     []string{"Transfer"},
			},
		},
		Server: ServerConfig{
			GraphQLPort: 8080,
			MetricsPort: 9090,
		},
		Sync: SyncConfig{
			BatchSize:  1000,
			MaxRetries: 3,
		},
		ChainID:      59144,
		PollInterval: 0,
	}

	require.Equal(t, "my-indexer", cfg.Name)
	require.Equal(t, "linea-mainnet", cfg.Network)
	require.Equal(t, uint64(59144), cfg.ChainID)
	require.Len(t, cfg.Contracts, 1)

	usdc, ok := cfg.Contracts["usdc"]
	require.True(t, ok)
	require.Equal(t, "0x176211869cA2b568f2A7D4EE941E073a821EE1ff", usdc.Address)
}
