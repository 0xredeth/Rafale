package handler

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/0xredeth/Rafale/pkg/decoder"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	require.NotNil(t, r)
	require.NotNil(t, r.handlers)
	require.Empty(t, r.handlers)
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name     string
		eventID  string
		handler  Func
		wantLen  int
	}{
		{
			name:    "register single handler",
			eventID: "USDC:Transfer",
			handler: func(ctx *Context) error { return nil },
			wantLen: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			r.Register(tc.eventID, tc.handler)
			require.Len(t, r.handlers, tc.wantLen)
		})
	}
}

func TestRegisterOverwrite(t *testing.T) {
	r := NewRegistry()

	var callCount int
	handler1 := func(ctx *Context) error { callCount = 1; return nil }
	handler2 := func(ctx *Context) error { callCount = 2; return nil }

	r.Register("USDC:Transfer", handler1)
	r.Register("USDC:Transfer", handler2) // Overwrite

	h, ok := r.Get("USDC:Transfer")
	require.True(t, ok)

	// Execute handler and verify it's the second one
	err := h(&Context{})
	require.NoError(t, err)
	require.Equal(t, 2, callCount)
}

func TestGet(t *testing.T) {
	r := NewRegistry()
	handler := func(ctx *Context) error { return nil }
	r.Register("USDC:Transfer", handler)

	tests := []struct {
		name    string
		eventID string
		wantOK  bool
	}{
		{
			name:    "existing handler",
			eventID: "USDC:Transfer",
			wantOK:  true,
		},
		{
			name:    "non-existent handler",
			eventID: "DAI:Transfer",
			wantOK:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, ok := r.Get(tc.eventID)
			require.Equal(t, tc.wantOK, ok)
			if tc.wantOK {
				require.NotNil(t, h)
			} else {
				require.Nil(t, h)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	tests := []struct {
		name       string
		setupReg   func(*Registry)
		ctx        *Context
		wantErr    bool
		wantErrMsg string
	}{
		{
			name:     "nil event",
			setupReg: func(r *Registry) {},
			ctx: &Context{
				Event: nil,
			},
			wantErr:    true,
			wantErrMsg: "event is nil",
		},
		{
			name:     "no handler registered",
			setupReg: func(r *Registry) {},
			ctx: &Context{
				Event: &decoder.DecodedEvent{
					EventID:      "USDC:Transfer",
					ContractName: "USDC",
					EventName:    "Transfer",
				},
			},
			wantErr: false, // Silent skip
		},
		{
			name: "handler success",
			setupReg: func(r *Registry) {
				r.Register("USDC:Transfer", func(ctx *Context) error {
					return nil
				})
			},
			ctx: &Context{
				Event: &decoder.DecodedEvent{
					EventID:      "USDC:Transfer",
					ContractName: "USDC",
					EventName:    "Transfer",
				},
				Block: BlockInfo{
					Number: 1000,
					Time:   time.Now(),
				},
			},
			wantErr: false,
		},
		{
			name: "handler error",
			setupReg: func(r *Registry) {
				r.Register("USDC:Transfer", func(ctx *Context) error {
					return errors.New("db error")
				})
			},
			ctx: &Context{
				Event: &decoder.DecodedEvent{
					EventID:      "USDC:Transfer",
					ContractName: "USDC",
					EventName:    "Transfer",
				},
				Block: BlockInfo{
					Number: 1000,
				},
			},
			wantErr:    true,
			wantErrMsg: "handler USDC:Transfer",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRegistry()
			tc.setupReg(r)

			err := r.Handle(tc.ctx)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestHasHandler(t *testing.T) {
	r := NewRegistry()
	r.Register("USDC:Transfer", func(ctx *Context) error { return nil })

	require.True(t, r.HasHandler("USDC:Transfer"))
	require.False(t, r.HasHandler("DAI:Transfer"))
}

func TestListHandlers(t *testing.T) {
	r := NewRegistry()

	// Empty registry
	handlers := r.ListHandlers()
	require.Empty(t, handlers)

	// Add handlers
	r.Register("USDC:Transfer", func(ctx *Context) error { return nil })
	r.Register("DAI:Transfer", func(ctx *Context) error { return nil })
	r.Register("USDC:Approval", func(ctx *Context) error { return nil })

	handlers = r.ListHandlers()
	require.Len(t, handlers, 3)
	require.Contains(t, handlers, "USDC:Transfer")
	require.Contains(t, handlers, "DAI:Transfer")
	require.Contains(t, handlers, "USDC:Approval")
}

func TestGlobal(t *testing.T) {
	g := Global()
	require.NotNil(t, g)
	require.Same(t, globalRegistry, g)
}

func TestGlobalRegisterAndGet(t *testing.T) {
	// Use a unique event ID to avoid conflicts with other tests
	eventID := "TestContract:TestEvent_" + time.Now().Format("150405")
	handler := func(ctx *Context) error { return nil }

	Register(eventID, handler)

	h, ok := Get(eventID)
	require.True(t, ok)
	require.NotNil(t, h)
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			eventID := "Contract:Event"
			r.Register(eventID, func(ctx *Context) error { return nil })
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Get("Contract:Event")
			r.HasHandler("Contract:Event")
			r.ListHandlers()
		}()
	}

	wg.Wait()
	// Test passes if no race condition or panic occurs
	require.True(t, r.HasHandler("Contract:Event"))
}

func TestContextFields(t *testing.T) {
	// Verify Context struct has expected fields
	ctx := &Context{
		DB: &gorm.DB{},
		Block: BlockInfo{
			Number:     12345,
			Hash:       "0xabc",
			Time:       time.Now(),
			ParentHash: "0xdef",
		},
		Log: types.Log{
			Address: common.HexToAddress("0x1234"),
			TxHash:  common.HexToHash("0x5678"),
		},
		Event: &decoder.DecodedEvent{
			EventID: "Test:Event",
		},
	}

	require.NotNil(t, ctx.DB)
	require.Equal(t, uint64(12345), ctx.Block.Number)
	require.Equal(t, "0xabc", ctx.Block.Hash)
	require.NotZero(t, ctx.Block.Time)
	require.Equal(t, "0xdef", ctx.Block.ParentHash)
	require.Equal(t, common.HexToAddress("0x1234"), ctx.Log.Address)
	require.Equal(t, "Test:Event", ctx.Event.EventID)
}

func TestBlockInfoFields(t *testing.T) {
	now := time.Now()
	info := BlockInfo{
		Number:     999999,
		Hash:       "0xhash",
		Time:       now,
		ParentHash: "0xparent",
	}

	require.Equal(t, uint64(999999), info.Number)
	require.Equal(t, "0xhash", info.Hash)
	require.Equal(t, now, info.Time)
	require.Equal(t, "0xparent", info.ParentHash)
}
