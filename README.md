# Rafale

<div align="center">

> *"Pas mal, non ? C'est français."* 🇫🇷

💨 **Rafale** — Lightweight Event Indexer for Linea zkEVM

[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Linea](https://img.shields.io/badge/Linea-zkEVM-000000?style=flat&logo=ethereum)](https://linea.build/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18+-336791?style=flat&logo=postgresql)](https://postgresql.org/)

*A burst of blockchain events — Go-native indexer exploiting Linea's ZK finality & EIP-4844 blobs.*
*Single binary, PostgreSQL + TimescaleDB + GraphQL API. Complements Lion for full-stack Linea development.*

</div>

---

> ⚠️ **Work in Progress** — This project is under active development. Features, APIs, and performance numbers are estimates and subject to change. Not production-ready yet.

---

## Overview

Rafale is a minimal, high-performance blockchain event indexer built specifically for Linea zkEVM. It indexes smart contract events into PostgreSQL with TimescaleDB and exposes them via a type-safe GraphQL API.

> **Core Philosophy:** Lightweight by design. One chain. One binary. Zero complexity.

### Why Linea-only?

| Reason | Benefit |
|--------|---------|
| **ZK-based finality** | No complex reorg handling needed |
| **Focused scope** | Smaller codebase, easier maintenance |
| **Lion complement** | Complete read/write stack for Linea |
| **ConsenSys ecosystem** | MetaMask, Infura, Besu alignment |

---

## Design Philosophy

### Why Linea-Only?

By targeting Linea exclusively, Rafale eliminates the abstraction overhead of multi-chain indexers. No generic EVM fallbacks, no reorg recovery logic, no chain-specific edge cases. The result: a smaller codebase, lower memory footprint, and optimizations tailored to Linea's ZK-finality model.

### Finalized State, Not Sequencer State

Rafale indexes **finalized blocks** - data that has been proven on L1 and cannot reorg. This is intentional:

- **Dashboards & Analytics**: Need accurate historical data, not millisecond latency
- **Accounting Systems**: Require immutable records
- **Backend APIs**: Serve consistent state to users

For sub-second trading data, query the sequencer directly. For everything else, use Rafale.

---

## Features

### v1.0 (Current Target)

- ✅ **No checkpoint table** — uses `MAX(block_number)` from event tables
- ✅ **Unified sync loop** — no historical vs live distinction
- ✅ **Minimal config** — network presets deduce most values
- ✅ **Single binary** — `--watch` flag for dev mode
- ✅ **GraphQL only** — no gRPC complexity
- ✅ **TimescaleDB** — hypertables for time-series event data
- ✅ **Circuit breaker** — RPC resilience with exponential backoff

### v2.0 (Roadmap)

- 🔄 Blob-based indexing via EIP-4844
- 🔄 Conflation-aware syncing
- 🔄 Shnarf-based caching
- 🔄 WebSocket streaming

---

## Quick Start

### Prerequisites

- Go 1.25+
- PostgreSQL 18+ with TimescaleDB 2.23+
- Linea RPC endpoint

### Installation

```bash
git clone https://github.com/0xredeth/rafale.git
cd rafale
go build -o rafale ./cmd/rafale
```

### Database Setup

```bash
# Install TimescaleDB extension
sudo apt install postgresql-18 timescaledb-2-postgresql-18

# Enable in postgresql.conf
# shared_preload_libraries = 'timescaledb'

# Create database
createdb rafale
psql -d rafale -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"
```

### Configuration

Copy the example configuration and customize:

```bash
cp rafale.example.yaml rafale.yaml
```

```yaml
# rafale.yaml
name: my-indexer
network: linea-mainnet
database: ${DATABASE_URL}

contracts:
  usdc:  # lowercase name - must match handler registration
    abi: ./abis/erc20.json
    address: "0x176211869cA2b568f2A7D4EE941E073a821EE1ff"
    start_block: 1000000
    events:
      - Transfer   # Must match ABI event name exactly (case-sensitive)
      - Approval
```

See [rafale.example.yaml](rafale.example.yaml) for a complete configuration reference.

### Run

```bash
# Set environment
export DATABASE_URL="postgres://user:pass@localhost/rafale"
export LINEA_RPC_URL="https://linea-mainnet.infura.io/v3/YOUR_KEY"

# Start indexing
./rafale start

# Development mode (hot reload)
./rafale start --watch
```

---

## Architecture

```
Linea RPC → Engine → Decoder → Handlers → PostgreSQL/TimescaleDB → GraphQL API
```

```
rafale/
├── cmd/rafale/main.go       # CLI entry point
├── pkg/
│   ├── config/              # Viper config + network presets
│   ├── engine/              # Unified sync loop
│   ├── handler/             # Handler registry + context
│   ├── rpc/                 # Linea RPC client + circuit breaker
│   ├── store/               # GORM + PostgreSQL + TimescaleDB
│   └── api/graphql/         # gqlgen server
├── generated/               # Code-generated bindings
├── abis/                    # Contract ABIs
└── rafale.yaml              # Config file
```

---

## Usage

### Define Schema

```go
// schema.go
type Transfer struct {
    schema.Model
    From        string    `gorm:"index;type:varchar(42)"`
    To          string    `gorm:"index;type:varchar(42)"`
    Amount      string    `gorm:"type:numeric"`
    BlockNumber uint64    `gorm:"index"`
    TxHash      string    `gorm:"type:varchar(66);uniqueIndex"`
    Timestamp   time.Time `gorm:"index"`
}
```

### Write Handlers

```go
// handlers.go
func init() {
    // Format: "contractName:EventName" (contractName is lowercase config key)
    handler.Register("usdc:Transfer", handleTransfer)
}

// handleTransfer processes USDC Transfer events.
//
// Parameters:
//   - ctx (*handler.Context): handler context with DB, block info, and decoded event
//
// Returns:
//   - error: nil on success, database error on failure
func handleTransfer(ctx *handler.Context) error {
    // Extract decoded event data from map
    data := ctx.Event.Data
    from := data["from"].(common.Address).Hex()
    to := data["to"].(common.Address).Hex()
    value := data["value"].(*big.Int).String()

    return ctx.DB.Create(&Transfer{
        From:        from,
        To:          to,
        Amount:      value,
        BlockNumber: ctx.Block.Number,
        TxHash:      ctx.Log.TxHash.Hex(),
        Timestamp:   ctx.Block.Time,
    }).Error
}
```

### Query via GraphQL

```graphql
query {
  events(first: 10) {
    totalCount
    edges {
      cursor
      node {
        id
        blockNumber
        txHash
        eventName
        data
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
  syncStatus {
    network
    chainID
    currentBlock
    headBlock
    lag
    isSynced
  }
}
```

---

## Network Presets

| Network | Chain ID | Poll Interval | Default RPC |
|---------|----------|---------------|-------------|
| `linea-mainnet` | 59144 | 2s | https://rpc.linea.build |
| `linea-sepolia` | 59141 | 2s | https://rpc.sepolia.linea.build |

---

## CLI Commands

```bash
rafale start              # Start indexing
rafale start --watch      # Dev mode with hot reload
rafale codegen            # Generate code from ABIs
rafale status             # Check sync status
rafale reset              # Reset indexed data
```

---

## Deployment

### Docker

> 🚧 **Coming Soon** — Docker support and docker-compose configurations are planned for a future release.

---

## Monitoring

### Endpoints

| Endpoint | Port | Description |
|----------|------|-------------|
| `/` | 8080 | GraphQL Playground (interactive IDE) |
| `/graphql` | 8080 | GraphQL API |
| `/health` | 8080 | Liveness probe |
| `/metrics` | 9090 | Prometheus metrics |

### Prometheus Metrics

```
rafale_blocks_indexed_total
rafale_events_processed_total{contract,event}
rafale_sync_lag_blocks
rafale_rpc_request_duration_seconds
rafale_circuit_breaker_state{name}
```

---

## Performance

Measured on local development machine (Apple Silicon, PostgreSQL local):

| Metric | Rafale | Notes |
|--------|--------|-------|
| Memory | ~30 MB | Idle indexer with handlers loaded |
| Startup | <1s | Cold start to first block fetch |
| GraphQL | ~6 req/s | Simple queries via curl |
| Events/block | 40+ | Varies by contract activity |

> 💡 **Lightweight by design** — Rafale uses minimal memory compared to Node.js-based indexers (typically 200-500MB+).

---

## Comparison

| | Rafale | Ponder | The Graph |
|---|--------|--------|-----------|
| **Language** | Go | TypeScript | AssemblyScript |
| **Chain** | Linea only | Any EVM | Any EVM |
| **Deploy** | Single binary | Node.js | IPFS |
| **Reorgs** | None (ZK) | Full | Full |
| **Time-series** | TimescaleDB | None | None |
| **Best for** | Go + Linea | TS teams | Decentralized |

---

## Companion Projects

| Project | Description |
|---------|-------------|
| [**Lion**](https://github.com/0xredeth/Lion) | Write layer for Linea (transactions, ZK proofs) |

---

## Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push branch (`git push origin feature/amazing`)
5. Open Pull Request

---

## License

Apache 2.0 — see [LICENSE](LICENSE)

---

<div align="center">

**"A burst of blockchain events. Index fast. Query faster."** 💨

</div>
