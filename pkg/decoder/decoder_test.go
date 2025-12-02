package decoder

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// ERC20 ABI for testing
const erc20ABI = `[
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "name": "from", "type": "address"},
      {"indexed": true, "name": "to", "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ],
    "name": "Transfer",
    "type": "event"
  },
  {
    "anonymous": false,
    "inputs": [
      {"indexed": true, "name": "owner", "type": "address"},
      {"indexed": true, "name": "spender", "type": "address"},
      {"indexed": false, "name": "value", "type": "uint256"}
    ],
    "name": "Approval",
    "type": "event"
  }
]`

// Test addresses
var (
	testContractAddr = common.HexToAddress("0x176211869cA2b568f2A7D4EE941E073a821EE1ff")
	testFromAddr     = common.HexToAddress("0x1111111111111111111111111111111111111111")
	testToAddr       = common.HexToAddress("0x2222222222222222222222222222222222222222")
)

// Transfer event signature: keccak256("Transfer(address,address,uint256)")
var transferEventSig = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

func TestNew(t *testing.T) {
	d := New()
	require.NotNil(t, d)
	require.NotNil(t, d.abis)
	require.NotNil(t, d.events)
	require.NotNil(t, d.sigToID)
	require.Empty(t, d.abis)
	require.Empty(t, d.events)
}

func TestRegisterContract(t *testing.T) {
	tests := []struct {
		name        string
		contractNm  string
		address     common.Address
		abiJSON     string
		eventNames  []string
		wantErr     bool
		wantErrMsg  string
		wantEvents  int
		wantAddrs   int
	}{
		{
			name:       "valid ERC20 all events",
			contractNm: "USDC",
			address:    testContractAddr,
			abiJSON:    erc20ABI,
			eventNames: nil, // register all
			wantErr:    false,
			wantEvents: 2,
			wantAddrs:  1,
		},
		{
			name:       "valid ERC20 specific event",
			contractNm: "USDC",
			address:    testContractAddr,
			abiJSON:    erc20ABI,
			eventNames: []string{"Transfer"},
			wantErr:    false,
			wantEvents: 1,
			wantAddrs:  1,
		},
		{
			name:       "invalid ABI JSON",
			contractNm: "Bad",
			address:    testContractAddr,
			abiJSON:    "not valid json",
			eventNames: nil,
			wantErr:    true,
			wantErrMsg: "parsing ABI",
		},
		{
			name:       "empty ABI",
			contractNm: "Empty",
			address:    testContractAddr,
			abiJSON:    "[]",
			eventNames: nil,
			wantErr:    false,
			wantEvents: 0,
			wantAddrs:  1,
		},
		{
			name:       "non-existent event filter",
			contractNm: "USDC",
			address:    testContractAddr,
			abiJSON:    erc20ABI,
			eventNames: []string{"NonExistent"},
			wantErr:    false,
			wantEvents: 0,
			wantAddrs:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := New()
			err := d.RegisterContract(tc.contractNm, tc.address, tc.abiJSON, tc.eventNames)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			require.Len(t, d.events, tc.wantEvents)
			require.Len(t, d.abis, tc.wantAddrs)
		})
	}
}

func TestGetEventSignatures(t *testing.T) {
	d := New()

	// Empty decoder
	sigs := d.GetEventSignatures()
	require.Empty(t, sigs)

	// After registration
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, []string{"Transfer"})
	require.NoError(t, err)

	sigs = d.GetEventSignatures()
	require.Len(t, sigs, 1)
	require.Equal(t, transferEventSig, sigs[0])
}

func TestGetAddresses(t *testing.T) {
	d := New()

	// Empty decoder
	addrs := d.GetAddresses()
	require.Empty(t, addrs)

	// After registration
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, nil)
	require.NoError(t, err)

	addrs = d.GetAddresses()
	require.Len(t, addrs, 1)
	require.Equal(t, testContractAddr, addrs[0])
}

func TestDecode(t *testing.T) {
	// Create decoder with Transfer event
	d := New()
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, []string{"Transfer"})
	require.NoError(t, err)

	// Encode a value for the data field (uint256)
	value := big.NewInt(1000000) // 1 USDC (6 decimals)
	valueBytes := common.LeftPadBytes(value.Bytes(), 32)

	tests := []struct {
		name       string
		log        types.Log
		wantErr    bool
		wantErrMsg string
		checkEvent func(t *testing.T, event *DecodedEvent)
	}{
		{
			name: "valid Transfer event",
			log: types.Log{
				Address: testContractAddr,
				Topics: []common.Hash{
					transferEventSig,
					common.BytesToHash(testFromAddr.Bytes()),
					common.BytesToHash(testToAddr.Bytes()),
				},
				Data: valueBytes,
			},
			wantErr: false,
			checkEvent: func(t *testing.T, event *DecodedEvent) {
				require.Equal(t, "USDC", event.ContractName)
				require.Equal(t, "Transfer", event.EventName)
				require.Equal(t, "USDC:Transfer", event.EventID)
				require.Equal(t, testFromAddr, event.Data["from"])
				require.Equal(t, testToAddr, event.Data["to"])
				require.Equal(t, value, event.Data["value"])
			},
		},
		{
			name: "no topics",
			log: types.Log{
				Address: testContractAddr,
				Topics:  nil,
				Data:    nil,
			},
			wantErr:    true,
			wantErrMsg: "no topics",
		},
		{
			name: "unknown event signature",
			log: types.Log{
				Address: testContractAddr,
				Topics: []common.Hash{
					common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"),
				},
				Data: nil,
			},
			wantErr:    true,
			wantErrMsg: "unknown event signature",
		},
		{
			name: "empty data field is valid",
			log: types.Log{
				Address: testContractAddr,
				Topics: []common.Hash{
					transferEventSig,
					common.BytesToHash(testFromAddr.Bytes()),
					common.BytesToHash(testToAddr.Bytes()),
				},
				Data: nil, // No data - value will be missing but indexed fields decoded
			},
			wantErr: false,
			checkEvent: func(t *testing.T, event *DecodedEvent) {
				require.Equal(t, "USDC:Transfer", event.EventID)
				require.Equal(t, testFromAddr, event.Data["from"])
				require.Equal(t, testToAddr, event.Data["to"])
				// value won't be present since data is empty
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			event, err := d.Decode(tc.log)

			if tc.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErrMsg)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, event)
			if tc.checkEvent != nil {
				tc.checkEvent(t, event)
			}
		})
	}
}

func TestCanDecode(t *testing.T) {
	d := New()
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, []string{"Transfer"})
	require.NoError(t, err)

	tests := []struct {
		name string
		log  types.Log
		want bool
	}{
		{
			name: "registered event",
			log: types.Log{
				Topics: []common.Hash{transferEventSig},
			},
			want: true,
		},
		{
			name: "unregistered event",
			log: types.Log{
				Topics: []common.Hash{
					common.HexToHash("0x1234"),
				},
			},
			want: false,
		},
		{
			name: "no topics",
			log: types.Log{
				Topics: nil,
			},
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := d.CanDecode(tc.log)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestGetEventID(t *testing.T) {
	d := New()
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, []string{"Transfer"})
	require.NoError(t, err)

	tests := []struct {
		name   string
		log    types.Log
		wantID string
		wantOK bool
	}{
		{
			name: "registered event",
			log: types.Log{
				Topics: []common.Hash{transferEventSig},
			},
			wantID: "USDC:Transfer",
			wantOK: true,
		},
		{
			name: "unregistered event",
			log: types.Log{
				Topics: []common.Hash{common.HexToHash("0x1234")},
			},
			wantID: "",
			wantOK: false,
		},
		{
			name: "no topics",
			log: types.Log{
				Topics: nil,
			},
			wantID: "",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			id, ok := d.GetEventID(tc.log)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantID, id)
		})
	}
}

func TestClear(t *testing.T) {
	d := New()
	err := d.RegisterContract("USDC", testContractAddr, erc20ABI, nil)
	require.NoError(t, err)

	require.NotEmpty(t, d.events)
	require.NotEmpty(t, d.abis)
	require.NotEmpty(t, d.sigToID)

	d.Clear()

	require.Empty(t, d.events)
	require.Empty(t, d.abis)
	require.Empty(t, d.sigToID)
}

func TestDecodeMultipleContracts(t *testing.T) {
	d := New()

	addr1 := common.HexToAddress("0x1111111111111111111111111111111111111111")
	addr2 := common.HexToAddress("0x2222222222222222222222222222222222222222")

	err := d.RegisterContract("USDC", addr1, erc20ABI, []string{"Transfer"})
	require.NoError(t, err)

	err = d.RegisterContract("DAI", addr2, erc20ABI, []string{"Transfer", "Approval"})
	require.NoError(t, err)

	// Should have 2 addresses registered
	addrs := d.GetAddresses()
	require.Len(t, addrs, 2)

	// Both use same Transfer signature, but different contract names
	// The last registration wins for same event signature
	sigs := d.GetEventSignatures()
	require.Len(t, sigs, 2) // Transfer + Approval (from DAI)
}

func TestDecodeBoolIndexedField(t *testing.T) {
	// ABI with a bool indexed field
	boolABI := `[{
		"anonymous": false,
		"inputs": [
			{"indexed": true, "name": "success", "type": "bool"},
			{"indexed": false, "name": "data", "type": "bytes"}
		],
		"name": "Result",
		"type": "event"
	}]`

	d := New()
	err := d.RegisterContract("Test", testContractAddr, boolABI, nil)
	require.NoError(t, err)

	sigs := d.GetEventSignatures()
	require.Len(t, sigs, 1)

	// Create a log with bool=true (non-zero)
	log := types.Log{
		Topics: []common.Hash{
			sigs[0],
			common.BigToHash(big.NewInt(1)), // true
		},
		Data: nil,
	}

	event, err := d.Decode(log)
	require.NoError(t, err)
	require.Equal(t, true, event.Data["success"])

	// Test bool=false (zero)
	log.Topics[1] = common.BigToHash(big.NewInt(0))
	event, err = d.Decode(log)
	require.NoError(t, err)
	require.Equal(t, false, event.Data["success"])
}
