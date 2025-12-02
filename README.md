# Rafale

<div align="center">

> *"Pas mal, non ? C'est français."* 🇫🇷

💨 **Rafale** — Lightweight Event Indexer for Linea zkEVM

[![License: AGPL-3.0](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat&logo=go)](https://golang.org/)
[![Linea](https://img.shields.io/badge/Linea-zkEVM-000000?style=flat&logo=ethereum)](https://linea.build/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-17-336791?style=flat&logo=postgresql)](https://postgresql.org/)

*A burst of blockchain events — Go-native indexer exploiting Linea's ZK finality & EIP-4844 blobs.*
*Single binary, PostgreSQL + TimescaleDB + GraphQL API. Complements Lion for full-stack Linea development.*

</div>

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

### v1.0 ✅ Complete

- ✅ **Hybrid Auto-Handler System** — zero-config event indexing with optional typed handlers
- ✅ **Any Contract, Any Event** — works with DEX, NFT, lending, governance - not just ERC20
- ✅ **No checkpoint table** — uses `MAX(block_number)` from event tables
- ✅ **Unified sync loop** — no historical vs live distinction
- ✅ **Minimal config** — network presets deduce most values
- ✅ **Single binary** — `--watch` flag for dev mode
- ✅ **GraphQL API** — queries + real-time subscriptions via WebSocket
- ✅ **TimescaleDB** — hypertables for time-series event data
- ✅ **Circuit breaker** — RPC resilience with exponential backoff
- ✅ **Prometheus metrics** — full observability out of the box

### v2.0 (Roadmap)

- 🔄 Blob-based indexing via EIP-4844
- 🔄 Conflation-aware syncing
- 🔄 Shnarf-based caching

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

#### macOS (Homebrew)

```bash
# Add TimescaleDB tap and install
brew tap timescale/tap
brew install timescaledb libpq

# Link psql to PATH
brew link --force libpq

# Run TimescaleDB tuner (configures postgresql.conf automatically)
timescaledb-tune --quiet --yes

# Start PostgreSQL (version 17 minimum)
brew services start postgresql@<version>

# Create database and enable TimescaleDB
createdb rafale
psql -d rafale -c "CREATE EXTENSION IF NOT EXISTS timescaledb;"

# Verify installation
psql -d rafale -c "\dx"
```

#### Linux

> 🚧 **Work in Progress** — Linux installation guide coming soon.

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
  # ERC20 tokens
  usdc:
    abi: ./abis/erc20.json
    address: "0x176211869cA2b568f2A7D4EE941E073a821EE1ff"
    start_block: 14000000
    events:
      - Transfer

  # DEX pools (SyncSwap, etc.)
  syncswap_pool:
    abi: ./abis/syncswap_pool.json
    address: "0x..."
    start_block: 14000000
    events:
      - Swap
      - Mint
      - Burn

  # Any contract, any event - no handler code required!
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
Linea RPC → Engine → Decoder → [Auto-Store + Handlers] → PostgreSQL/TimescaleDB → GraphQL API
```

### Hybrid Auto-Handler System

Rafale uses a **zero-config event indexing** approach:

```
Decoded Event
     │
     ├──► Generic Events Table (always stored, JSONB data)
     │         └─► GraphQL: events(filter: { contract, event })
     │
     └──► Typed Handler (if registered)
               └─► Typed Table (indexed columns)
```

| Mode | Use Case | Setup Required |
|------|----------|----------------|
| **Generic Only** | Exploration, prototyping | Just add contract to YAML |
| **Hybrid** | Production with typed queries | Add handler for specific events |

**Benefits:**
- Start indexing immediately - no handler code required
- Events queryable via GraphQL out of the box
- Add typed handlers later for performance-critical queries
- Works with ANY Ethereum event (DEX Swap, NFT Transfer, Lending Borrow, etc.)

```
rafale/
├── cmd/rafale/              # CLI entry point + commands
├── pkg/
│   ├── config/              # Viper config + network presets
│   ├── decoder/             # ABI event decoding
│   └── handler/             # Handler registry + context
├── internal/
│   ├── api/                 # GraphQL server + resolvers
│   ├── codegen/             # Code generation templates
│   ├── engine/              # Unified sync loop + metrics
│   ├── pubsub/              # Real-time event broadcasting
│   ├── rpc/                 # Linea RPC client
│   ├── store/               # GORM + PostgreSQL + TimescaleDB
│   └── watcher/             # Hot-reload file watcher
├── abis/                    # Contract ABIs
└── rafale.yaml              # Config file
```

---

## Usage

> 📖 For detailed usage instructions, see [use.md](use.md)

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
| Binary | **~33 MB** | Single Go binary, no dependencies |
| Memory | **~30 MB** | Idle indexer with handlers loaded |
| Startup | **<1s** | Cold start to first block fetch |
| Codebase | **~14K LOC** | 39 Go source files |
| Events/block | **40+** | Varies by contract activity |

> 💡 **Lightweight by design** — Rafale uses minimal memory compared to Node.js-based indexers (typically 200-500MB+). The single 33MB binary includes everything needed to run.

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

AGPL-3.0 — see [LICENSE](LICENSE)

---

<div align="center">

**"A burst of blockchain events. Index fast. Query faster."** 💨

</div>
