// Package config provides configuration management for Rafale.
package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all Rafale configuration.
type Config struct {
	// Name is the indexer instance name.
	Name string `mapstructure:"name"`

	// Network is the target network (linea-mainnet, linea-sepolia).
	Network string `mapstructure:"network"`

	// Database is the PostgreSQL connection string.
	Database string `mapstructure:"database"`

	// RPCURL is the Linea RPC endpoint (overrides network preset).
	RPCURL string `mapstructure:"rpc_url"`

	// Contracts defines the contracts to index.
	Contracts map[string]ContractConfig `mapstructure:"contracts"`

	// Server holds API server configuration.
	Server ServerConfig `mapstructure:"server"`

	// Sync holds synchronization configuration.
	Sync SyncConfig `mapstructure:"sync"`

	// Derived fields (populated from network preset).
	ChainID      uint64
	PollInterval time.Duration
}

// ContractConfig defines a contract to index.
type ContractConfig struct {
	// ABI is the path to the ABI JSON file.
	ABI string `mapstructure:"abi"`

	// Address is the contract address.
	Address string `mapstructure:"address"`

	// StartBlock is the block to start indexing from.
	StartBlock uint64 `mapstructure:"start_block"`

	// Events is the list of event names to index.
	Events []string `mapstructure:"events"`
}

// ServerConfig holds API server configuration.
type ServerConfig struct {
	// GraphQLPort is the GraphQL server port.
	GraphQLPort int `mapstructure:"graphql_port"`

	// MetricsPort is the Prometheus metrics port.
	MetricsPort int `mapstructure:"metrics_port"`
}

// SyncConfig holds synchronization configuration.
type SyncConfig struct {
	// BatchSize is the number of blocks to fetch per batch.
	BatchSize uint64 `mapstructure:"batch_size"`

	// MaxRetries is the maximum RPC retry attempts.
	MaxRetries int `mapstructure:"max_retries"`

	// RetryDelay is the initial retry delay.
	RetryDelay time.Duration `mapstructure:"retry_delay"`
}

// Load reads configuration from file and environment.
//
// Returns:
//   - *Config: the loaded configuration
//   - error: nil on success, configuration error on failure
func Load() (*Config, error) {
	cfg := &Config{}

	// Set defaults
	setDefaults()

	// Unmarshal configuration
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	// Apply network preset
	preset, ok := NetworkPresets[cfg.Network]
	if !ok {
		return nil, fmt.Errorf("unknown network: %s (valid: linea-mainnet, linea-sepolia)", cfg.Network)
	}

	cfg.ChainID = preset.ChainID
	cfg.PollInterval = preset.PollInterval

	// Use preset RPC if not overridden
	if cfg.RPCURL == "" {
		cfg.RPCURL = preset.DefaultRPC
	}

	// Allow environment variable override for database
	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		cfg.Database = dbURL
	}

	// Allow environment variable override for RPC
	if rpcURL := os.Getenv("LINEA_RPC_URL"); rpcURL != "" {
		cfg.RPCURL = rpcURL
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks that all required configuration is present.
//
// Returns:
//   - error: nil if valid, validation error otherwise
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name is required")
	}
	if c.Network == "" {
		return fmt.Errorf("network is required")
	}
	if c.Database == "" {
		return fmt.Errorf("database connection string is required (set DATABASE_URL env var or database in config)")
	}
	if len(c.Contracts) == 0 {
		return fmt.Errorf("at least one contract must be defined")
	}

	for name, contract := range c.Contracts {
		if contract.Address == "" {
			return fmt.Errorf("contract %s: address is required", name)
		}
		if contract.ABI == "" {
			return fmt.Errorf("contract %s: abi path is required", name)
		}
		if len(contract.Events) == 0 {
			return fmt.Errorf("contract %s: at least one event must be specified", name)
		}
	}

	return nil
}

// setDefaults sets default configuration values.
func setDefaults() {
	viper.SetDefault("network", "linea-mainnet")
	viper.SetDefault("server.graphql_port", 8080)
	viper.SetDefault("server.metrics_port", 9090)
	viper.SetDefault("sync.batch_size", 1000)
	viper.SetDefault("sync.max_retries", 3)
	viper.SetDefault("sync.retry_delay", "1s")
}
