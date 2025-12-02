package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// testStore holds test database resources.
type testStore struct {
	store     *Store
	container testcontainers.Container
	dsn       string
}

// setupTestStore creates a PostgreSQL container and store for testing.
func setupTestStore(t *testing.T) *testStore {
	t.Helper()
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("rafale_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	// Get connection string
	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create store
	cfg := DefaultConfig()
	cfg.DSN = dsn
	cfg.LogLevel = logger.Silent

	store, err := New(cfg)
	require.NoError(t, err)

	return &testStore{
		store:     store,
		container: container,
		dsn:       dsn,
	}
}

// teardown cleans up test resources.
func (ts *testStore) teardown(t *testing.T) {
	t.Helper()
	if ts.store != nil {
		ts.store.Close()
	}
	if ts.container != nil {
		ts.container.Terminate(context.Background())
	}
}

// --- Config Tests ---

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, 25, cfg.MaxOpenConns)
	require.Equal(t, 5, cfg.MaxIdleConns)
	require.Equal(t, 5*time.Minute, cfg.ConnMaxLifetime)
	require.Equal(t, logger.Warn, cfg.LogLevel)
	require.Empty(t, cfg.DSN)
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		DSN:             "postgres://user:pass@localhost:5432/db",
		MaxOpenConns:    50,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10 * time.Minute,
		LogLevel:        logger.Info,
	}

	require.Equal(t, "postgres://user:pass@localhost:5432/db", cfg.DSN)
	require.Equal(t, 50, cfg.MaxOpenConns)
	require.Equal(t, 10, cfg.MaxIdleConns)
	require.Equal(t, 10*time.Minute, cfg.ConnMaxLifetime)
	require.Equal(t, logger.Info, cfg.LogLevel)
}

// --- TimescaleConfig Tests ---

func TestDefaultTimescaleConfig(t *testing.T) {
	cfg := DefaultTimescaleConfig()

	require.Equal(t, "1 day", cfg.ChunkInterval)
	require.Equal(t, "7 days", cfg.CompressAfter)
	require.Empty(t, cfg.RetainFor)
}

func TestTimescaleConfigStruct(t *testing.T) {
	cfg := TimescaleConfig{
		ChunkInterval: "12 hours",
		CompressAfter: "3 days",
		RetainFor:     "30 days",
	}

	require.Equal(t, "12 hours", cfg.ChunkInterval)
	require.Equal(t, "3 days", cfg.CompressAfter)
	require.Equal(t, "30 days", cfg.RetainFor)
}

// --- Model Tests ---

func TestBaseEventStruct(t *testing.T) {
	now := time.Now()
	be := BaseEvent{
		ID:          1,
		Timestamp:   now,
		BlockNumber: 12345,
		TxHash:      "0xabc123",
		TxIndex:     5,
		LogIndex:    2,
	}

	require.Equal(t, uint64(1), be.ID)
	require.Equal(t, now, be.Timestamp)
	require.Equal(t, uint64(12345), be.BlockNumber)
	require.Equal(t, "0xabc123", be.TxHash)
	require.Equal(t, uint(5), be.TxIndex)
	require.Equal(t, uint(2), be.LogIndex)
}

func TestTransferStruct(t *testing.T) {
	now := time.Now()
	transfer := Transfer{
		BaseEvent: BaseEvent{
			ID:          1,
			Timestamp:   now,
			BlockNumber: 1000,
			TxHash:      "0x123",
			TxIndex:     0,
			LogIndex:    0,
		},
		From:  "0x1111111111111111111111111111111111111111",
		To:    "0x2222222222222222222222222222222222222222",
		Value: "1000000000000000000",
	}

	require.Equal(t, "transfers", transfer.TableName())
	require.Equal(t, "0x1111111111111111111111111111111111111111", transfer.From)
	require.Equal(t, "0x2222222222222222222222222222222222222222", transfer.To)
	require.Equal(t, "1000000000000000000", transfer.Value)
}

func TestEventStruct(t *testing.T) {
	now := time.Now()
	event := Event{
		BaseEvent: BaseEvent{
			ID:          1,
			Timestamp:   now,
			BlockNumber: 1000,
			TxHash:      "0x123",
		},
		ContractName: "USDC",
		ContractAddr: "0x1234",
		EventName:    "Transfer",
		EventSig:     "0xddf252ad",
		Data:         datatypes.JSON(`{"from":"0x1","to":"0x2","value":"100"}`),
	}

	require.Equal(t, "events", event.TableName())
	require.Equal(t, "USDC", event.ContractName)
	require.Equal(t, "Transfer", event.EventName)
	require.Equal(t, "0xddf252ad", event.EventSig)
}

// --- Query Struct Tests ---

func TestTransferQueryStruct(t *testing.T) {
	fromBlock := uint64(100)
	toBlock := uint64(200)
	fromTime := time.Now().Add(-24 * time.Hour)
	toTime := time.Now()
	afterID := uint64(50)

	q := TransferQuery{
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
		FromTime:  &fromTime,
		ToTime:    &toTime,
		OrderBy:   "timestamp",
		OrderDir:  "DESC",
		Limit:     100,
		AfterID:   &afterID,
	}

	require.Equal(t, uint64(100), *q.FromBlock)
	require.Equal(t, uint64(200), *q.ToBlock)
	require.Equal(t, "timestamp", q.OrderBy)
	require.Equal(t, "DESC", q.OrderDir)
	require.Equal(t, 100, q.Limit)
	require.Equal(t, uint64(50), *q.AfterID)
}

func TestEventQueryStruct(t *testing.T) {
	contractName := "USDC"
	eventName := "Transfer"
	fromBlock := uint64(1000)

	q := EventQuery{
		ContractName: &contractName,
		EventName:    &eventName,
		FromBlock:    &fromBlock,
		OrderBy:      "block_number",
		OrderDir:     "ASC",
		Limit:        50,
	}

	require.Equal(t, "USDC", *q.ContractName)
	require.Equal(t, "Transfer", *q.EventName)
	require.Equal(t, uint64(1000), *q.FromBlock)
	require.Equal(t, 50, q.Limit)
}

// --- Integration Tests (require Docker) ---

func TestNewStoreWithPostgres(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	require.NotNil(t, ts.store)
	require.NotNil(t, ts.store.DB())
}

func TestStoreMigrate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{}, &Event{})
	require.NoError(t, err)

	// Verify tables exist
	var exists bool
	ts.store.DB().Raw("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'transfers')").Scan(&exists)
	require.True(t, exists)

	ts.store.DB().Raw("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'events')").Scan(&exists)
	require.True(t, exists)
}

func TestStoreTransaction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()

	// Test successful transaction
	err = ts.store.Transaction(ctx, func(tx *gorm.DB) error {
		return tx.Create(&Transfer{
			BaseEvent: BaseEvent{
				Timestamp:   time.Now(),
				BlockNumber: 1000,
				TxHash:      "0x111",
				TxIndex:     0,
				LogIndex:    0,
			},
			From:  "0xfrom",
			To:    "0xto",
			Value: "100",
		}).Error
	})
	require.NoError(t, err)

	// Verify record exists
	count, err := ts.store.GetTransferCount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}

func TestStoreTransactionRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()

	// Test rollback on error
	err = ts.store.Transaction(ctx, func(tx *gorm.DB) error {
		tx.Create(&Transfer{
			BaseEvent: BaseEvent{
				Timestamp:   time.Now(),
				BlockNumber: 1000,
				TxHash:      "0x111",
			},
			From:  "0xfrom",
			To:    "0xto",
			Value: "100",
		})
		return errForceRollback
	})
	require.Error(t, err)

	// Verify rollback occurred
	count, err := ts.store.GetTransferCount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

var errForceRollback = errors.New("force rollback")

func TestQueryTransfersBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()

	// Insert test data
	now := time.Now()
	transfers := []Transfer{
		{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 100, TxHash: "0x1", LogIndex: 0}, From: "0xa", To: "0xb", Value: "100"},
		{BaseEvent: BaseEvent{Timestamp: now.Add(time.Second), BlockNumber: 101, TxHash: "0x2", LogIndex: 0}, From: "0xb", To: "0xc", Value: "200"},
		{BaseEvent: BaseEvent{Timestamp: now.Add(2 * time.Second), BlockNumber: 102, TxHash: "0x3", LogIndex: 0}, From: "0xc", To: "0xd", Value: "300"},
	}

	for _, tr := range transfers {
		err := ts.store.DB().Create(&tr).Error
		require.NoError(t, err)
	}

	// Query all
	results, total, err := ts.store.QueryTransfers(ctx, TransferQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, results, 3)
}

func TestQueryTransfersWithBlockFilter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert test data
	for i := uint64(100); i < 110; i++ {
		ts.store.DB().Create(&Transfer{
			BaseEvent: BaseEvent{Timestamp: now.Add(time.Duration(i) * time.Second), BlockNumber: i, TxHash: "0x" + string(rune(i)), LogIndex: 0},
			From:      "0xa",
			To:        "0xb",
			Value:     "100",
		})
	}

	// Filter by block range
	fromBlock := uint64(103)
	toBlock := uint64(107)
	results, total, err := ts.store.QueryTransfers(ctx, TransferQuery{
		FromBlock: &fromBlock,
		ToBlock:   &toBlock,
	})
	require.NoError(t, err)
	require.Equal(t, int64(5), total)
	require.Len(t, results, 5)

	// Verify all results are in range
	for _, r := range results {
		require.GreaterOrEqual(t, r.BlockNumber, uint64(103))
		require.LessOrEqual(t, r.BlockNumber, uint64(107))
	}
}

func TestQueryTransfersWithOrdering(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert test data in mixed order
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 200, TxHash: "0x1"}, From: "0xa", To: "0xb", Value: "100"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now.Add(time.Second), BlockNumber: 100, TxHash: "0x2"}, From: "0xa", To: "0xb", Value: "100"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now.Add(2 * time.Second), BlockNumber: 150, TxHash: "0x3"}, From: "0xa", To: "0xb", Value: "100"})

	// Order by block number ascending
	results, _, err := ts.store.QueryTransfers(ctx, TransferQuery{
		OrderBy:  "block_number",
		OrderDir: "ASC",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(100), results[0].BlockNumber)
	require.Equal(t, uint64(150), results[1].BlockNumber)
	require.Equal(t, uint64(200), results[2].BlockNumber)

	// Order by block number descending
	results, _, err = ts.store.QueryTransfers(ctx, TransferQuery{
		OrderBy:  "block_number",
		OrderDir: "DESC",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(200), results[0].BlockNumber)
	require.Equal(t, uint64(150), results[1].BlockNumber)
	require.Equal(t, uint64(100), results[2].BlockNumber)
}

func TestQueryTransfersWithLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert 10 transfers
	for i := 0; i < 10; i++ {
		ts.store.DB().Create(&Transfer{
			BaseEvent: BaseEvent{Timestamp: now.Add(time.Duration(i) * time.Second), BlockNumber: uint64(100 + i), TxHash: "0x" + string(rune(i))},
			From:      "0xa",
			To:        "0xb",
			Value:     "100",
		})
	}

	// Query with limit
	results, total, err := ts.store.QueryTransfers(ctx, TransferQuery{Limit: 5})
	require.NoError(t, err)
	require.Equal(t, int64(10), total) // Total count ignores limit
	require.Len(t, results, 5)
}

func TestQueryTransfersCursorPagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert transfers
	for i := 0; i < 5; i++ {
		ts.store.DB().Create(&Transfer{
			BaseEvent: BaseEvent{Timestamp: now.Add(time.Duration(i) * time.Second), BlockNumber: uint64(100 + i), TxHash: "0x" + string(rune(i))},
			From:      "0xa",
			To:        "0xb",
			Value:     "100",
		})
	}

	// Get first page
	results, _, err := ts.store.QueryTransfers(ctx, TransferQuery{Limit: 2, OrderDir: "ASC"})
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Get next page using cursor
	lastID := results[1].ID
	results, _, err = ts.store.QueryTransfers(ctx, TransferQuery{Limit: 2, AfterID: &lastID, OrderDir: "ASC"})
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Greater(t, results[0].ID, lastID)
}

func TestGetTransferByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()

	// Insert a transfer
	transfer := &Transfer{
		BaseEvent: BaseEvent{Timestamp: time.Now(), BlockNumber: 1000, TxHash: "0x123"},
		From:      "0xfrom",
		To:        "0xto",
		Value:     "1000000",
	}
	ts.store.DB().Create(transfer)

	// Get by ID
	result, err := ts.store.GetTransferByID(ctx, transfer.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, transfer.ID, result.ID)
	require.Equal(t, "0xfrom", result.From)

	// Get non-existent
	result, err = ts.store.GetTransferByID(ctx, 99999)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetTransfersByTxHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert multiple transfers with same tx hash (batch transfer)
	txHash := "0xbatch123"
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 1000, TxHash: txHash, LogIndex: 0}, From: "0xa", To: "0xb", Value: "100"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 1000, TxHash: txHash, LogIndex: 1}, From: "0xb", To: "0xc", Value: "200"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 1000, TxHash: "0xother", LogIndex: 0}, From: "0xc", To: "0xd", Value: "300"})

	// Get by tx hash
	results, err := ts.store.GetTransfersByTxHash(ctx, txHash)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Verify ordered by log_index
	require.Equal(t, uint(0), results[0].LogIndex)
	require.Equal(t, uint(1), results[1].LogIndex)
}

func TestQueryEventsBasic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Event{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert test events
	ts.store.DB().Create(&Event{
		BaseEvent:    BaseEvent{Timestamp: now, BlockNumber: 100, TxHash: "0x1"},
		ContractName: "USDC",
		ContractAddr: "0x1234",
		EventName:    "Transfer",
		EventSig:     "0xddf252ad",
		Data:         datatypes.JSON(`{"from":"0xa","to":"0xb"}`),
	})
	ts.store.DB().Create(&Event{
		BaseEvent:    BaseEvent{Timestamp: now.Add(time.Second), BlockNumber: 101, TxHash: "0x2"},
		ContractName: "USDC",
		ContractAddr: "0x1234",
		EventName:    "Approval",
		EventSig:     "0x8c5be1e5",
		Data:         datatypes.JSON(`{"owner":"0xa","spender":"0xb"}`),
	})

	// Query all
	results, total, err := ts.store.QueryEvents(ctx, EventQuery{})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, results, 2)
}

func TestQueryEventsWithFilters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Event{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Insert events from different contracts
	ts.store.DB().Create(&Event{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 100, TxHash: "0x1"}, ContractName: "USDC", EventName: "Transfer", ContractAddr: "0x1", EventSig: "0x1", Data: datatypes.JSON(`{}`)})
	ts.store.DB().Create(&Event{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 101, TxHash: "0x2"}, ContractName: "USDC", EventName: "Approval", ContractAddr: "0x1", EventSig: "0x2", Data: datatypes.JSON(`{}`)})
	ts.store.DB().Create(&Event{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 102, TxHash: "0x3"}, ContractName: "DAI", EventName: "Transfer", ContractAddr: "0x2", EventSig: "0x1", Data: datatypes.JSON(`{}`)})

	// Filter by contract
	contractName := "USDC"
	results, total, err := ts.store.QueryEvents(ctx, EventQuery{ContractName: &contractName})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, results, 2)

	// Filter by event name
	eventName := "Transfer"
	results, total, err = ts.store.QueryEvents(ctx, EventQuery{EventName: &eventName})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)

	// Filter by both
	results, total, err = ts.store.QueryEvents(ctx, EventQuery{ContractName: &contractName, EventName: &eventName})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "USDC", results[0].ContractName)
	require.Equal(t, "Transfer", results[0].EventName)
}

func TestGetEventByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Event{})
	require.NoError(t, err)

	ctx := context.Background()

	// Insert an event
	event := &Event{
		BaseEvent:    BaseEvent{Timestamp: time.Now(), BlockNumber: 1000, TxHash: "0x123"},
		ContractName: "USDC",
		ContractAddr: "0x1234",
		EventName:    "Transfer",
		EventSig:     "0xddf252ad",
		Data:         datatypes.JSON(`{}`),
	}
	ts.store.DB().Create(event)

	// Get by ID
	result, err := ts.store.GetEventByID(ctx, event.ID)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "USDC", result.ContractName)

	// Non-existent
	result, err = ts.store.GetEventByID(ctx, 99999)
	require.NoError(t, err)
	require.Nil(t, result)
}

func TestGetMaxBlockNumber(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()

	// Empty table should return 0
	maxBlock, err := ts.store.GetMaxBlockNumber(ctx, "transfers")
	require.NoError(t, err)
	require.Equal(t, uint64(0), maxBlock)

	// Insert transfers
	now := time.Now()
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 100, TxHash: "0x1"}, From: "0xa", To: "0xb", Value: "100"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 500, TxHash: "0x2"}, From: "0xa", To: "0xb", Value: "100"})
	ts.store.DB().Create(&Transfer{BaseEvent: BaseEvent{Timestamp: now, BlockNumber: 300, TxHash: "0x3"}, From: "0xa", To: "0xb", Value: "100"})

	// Should return max
	maxBlock, err = ts.store.GetMaxBlockNumber(ctx, "transfers")
	require.NoError(t, err)
	require.Equal(t, uint64(500), maxBlock)
}

func TestCreateInBatches(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)
	defer ts.teardown(t)

	err := ts.store.Migrate(&Transfer{})
	require.NoError(t, err)

	ctx := context.Background()
	now := time.Now()

	// Create 100 transfers
	var transfers []Transfer
	for i := 0; i < 100; i++ {
		transfers = append(transfers, Transfer{
			BaseEvent: BaseEvent{
				Timestamp:   now.Add(time.Duration(i) * time.Second),
				BlockNumber: uint64(1000 + i),
				TxHash:      "0x" + string(rune(i)),
			},
			From:  "0xfrom",
			To:    "0xto",
			Value: "100",
		})
	}

	// Insert in batches
	err = ts.store.CreateInBatches(ctx, &transfers, 25)
	require.NoError(t, err)

	// Verify count
	count, err := ts.store.GetTransferCount(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(100), count)
}

func TestStoreClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ts := setupTestStore(t)

	// Close and verify no error
	err := ts.store.Close()
	require.NoError(t, err)

	// Clean up container only (store already closed)
	ts.store = nil
	ts.teardown(t)
}

func TestNewStoreWithInvalidDSN(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DSN = "postgres://invalid:invalid@localhost:9999/nonexistent"

	_, err := New(cfg)
	require.Error(t, err)
}
