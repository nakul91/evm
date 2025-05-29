package bitcoin

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	cmn "github.com/cosmos/evm/precompiles/common"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BitcoinNodeData represents data from a Bitcoin node
type BitcoinNodeData struct {
	BlockHeight       uint64
	BlockHash         string
	NetworkHashrate   *big.Int
	NetworkDifficulty *big.Int
	MempoolSize       uint64
	PeerCount         uint32
	LastUpdated       int64
}

// BitcoinData represents the Bitcoin-related data precompile contract.
type BitcoinData struct {
	cmn.Precompile
	stakingKeeper stakingkeeper.Keeper
	nodeData      map[string]*BitcoinNodeData // map of validator address to their node data
}

// NewPrecompile creates a new Bitcoin data precompile
func NewPrecompile(stakingKeeper stakingkeeper.Keeper) (*BitcoinData, error) {
	methods := make(map[string]abi.Method)
	events := make(map[string]abi.Event)

	// Register methods
	methods["setBitcoinPrice"] = abi.Method{
		Name:    "setBitcoinPrice",
		Inputs:  []abi.Argument{{Name: "price", Type: abi.Type{T: abi.UintTy}}},
		Outputs: []abi.Argument{},
	}

	methods["getBitcoinPrice"] = abi.Method{
		Name:    "getBitcoinPrice",
		Inputs:  []abi.Argument{},
		Outputs: []abi.Argument{{Name: "price", Type: abi.Type{T: abi.UintTy}}},
	}

	methods["setBlockHeight"] = abi.Method{
		Name:    "setBlockHeight",
		Inputs:  []abi.Argument{{Name: "height", Type: abi.Type{T: abi.UintTy}}},
		Outputs: []abi.Argument{},
	}

	methods["getBlockHeight"] = abi.Method{
		Name:    "getBlockHeight",
		Inputs:  []abi.Argument{},
		Outputs: []abi.Argument{{Name: "height", Type: abi.Type{T: abi.UintTy}}},
	}

	methods["setNetworkHashrate"] = abi.Method{
		Name:    "setNetworkHashrate",
		Inputs:  []abi.Argument{{Name: "hashrate", Type: abi.Type{T: abi.UintTy}}},
		Outputs: []abi.Argument{},
	}

	methods["getNetworkHashrate"] = abi.Method{
		Name:    "getNetworkHashrate",
		Inputs:  []abi.Argument{},
		Outputs: []abi.Argument{{Name: "hashrate", Type: abi.Type{T: abi.UintTy}}},
	}

	// Create ABI
	abi, err := abi.JSON(strings.NewReader(ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to create ABI: %w", err)
	}

	// Create the precompile
	p := &BitcoinData{
		Precompile: cmn.Precompile{
			ABI:                &abi,
			Methods:            methods,
			Events:             events,
			PrecompileAddress: common.HexToAddress("0x0000000000000000000000000000000000000800"),
			RequiredGas:       3000,
		},
		stakingKeeper: stakingKeeper,
		nodeData:      make(map[string]*BitcoinNodeData),
	}

	// Register the precompile methods
	p.Precompile.RegisterMethod("setBitcoinPrice", p.setBitcoinPrice)
	p.Precompile.RegisterMethod("getBitcoinPrice", p.getBitcoinPrice)
	p.Precompile.RegisterMethod("setBlockHeight", p.setBlockHeight)
	p.Precompile.RegisterMethod("getBlockHeight", p.getBlockHeight)
	p.Precompile.RegisterMethod("setNetworkHashrate", p.setNetworkHashrate)
	p.Precompile.RegisterMethod("getNetworkHashrate", p.getNetworkHashrate)

	return p, nil
}

// Required method for the vm.PrecompiledContract interface
func (p *BitcoinData) RequiredGas(input []byte) uint64 {
	return p.Precompile.RequiredGas(input)
}

// Required method for the vm.PrecompiledContract interface
func (p *BitcoinData) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.Precompile.Run(evm, contract, readonly)
}

// Method implementations

// isValidator checks if the given address is a validator
func (p *BitcoinData) isValidator(ctx sdk.Context, address common.Address) bool {
	valAddr := sdk.ValAddress(address.Bytes())
	validator, found := p.stakingKeeper.GetValidator(ctx, valAddr)
	return found && validator.IsBonded()
}

// updateNodeData updates Bitcoin node data for a validator
func (p *BitcoinData) updateNodeData(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	if len(args) != 6 {
		return nil, fmt.Errorf("invalid number of arguments")
	}

	// Check if caller is a validator
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !p.isValidator(sdkCtx, contract.Caller()) {
		return nil, fmt.Errorf("only validators can update node data")
	}

	// Parse arguments
	height, ok := args[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid block height")
	}

	blockHash, ok := args[1].(string)
	if !ok {
		return nil, fmt.Errorf("invalid block hash")
	}

	hashrate, ok := args[2].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid network hashrate")
	}

	difficulty, ok := args[3].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid network difficulty")
	}

	mempoolSize, ok := args[4].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid mempool size")
	}

	peerCount, ok := args[5].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid peer count")
	}

	// Update node data
	validatorAddr := contract.Caller().Hex()
	p.nodeData[validatorAddr] = &BitcoinNodeData{
		BlockHeight:       height.Uint64(),
		BlockHash:         blockHash,
		NetworkHashrate:   hashrate,
		NetworkDifficulty: difficulty,
		MempoolSize:       mempoolSize.Uint64(),
		PeerCount:         uint32(peerCount.Uint64()),
		LastUpdated:       sdkCtx.BlockTime().Unix(),
	}

	return nil, nil
}

// getNodeData retrieves Bitcoin node data for a validator
func (p *BitcoinData) getNodeData(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("invalid number of arguments")
	}

	validatorAddr, ok := args[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid validator address")
	}

	data, exists := p.nodeData[validatorAddr]
	if !exists {
		return nil, fmt.Errorf("no data found for validator")
	}

	// Pack the data into ABI-encoded bytes
	values := []interface{}{
		new(big.Int).SetUint64(data.BlockHeight),
		data.BlockHash,
		data.NetworkHashrate,
		data.NetworkDifficulty,
		new(big.Int).SetUint64(data.MempoolSize),
		new(big.Int).SetUint64(uint64(data.PeerCount)),
		new(big.Int).SetInt64(data.LastUpdated),
	}

	return p.ABI.Methods["getNodeData"].Outputs.Pack(values...)
}

func (p *BitcoinData) getBitcoinPrice(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	return p.bitcoinPrice.Bytes(), nil
}

func (p *BitcoinData) setBlockHeight(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("invalid number of arguments")
	}

	height, ok := args[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid height argument")
	}

	p.blockHeight = height.Uint64()
	return nil, nil
}

func (p *BitcoinData) getBlockHeight(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	return new(big.Int).SetUint64(p.blockHeight).Bytes(), nil
}

func (p *BitcoinData) setNetworkHashrate(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("invalid number of arguments")
	}

	hashrate, ok := args[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("invalid hashrate argument")
	}

	p.networkHashrate = hashrate
	return nil, nil
}

func (p *BitcoinData) getNetworkHashrate(ctx context.Context, contract *vm.Contract, evm *vm.EVM, args ...interface{}) ([]byte, error) {
	return p.networkHashrate.Bytes(), nil
}

// ABI is the JSON ABI for the Bitcoin Data contract
const ABI = `{
	"methods": {
		"updateNodeData": {
			"name": "updateNodeData",
			"inputs": [
				{"name": "blockHeight", "type": "uint256"},
				{"name": "blockHash", "type": "string"},
				{"name": "networkHashrate", "type": "uint256"},
				{"name": "networkDifficulty", "type": "uint256"},
				{"name": "mempoolSize", "type": "uint256"},
				{"name": "peerCount", "type": "uint256"}
			],
			"outputs": []
		},
		"getNodeData": {
			"name": "getNodeData",
			"inputs": [{"name": "validatorAddress", "type": "string"}],
			"outputs": [
				{"name": "blockHeight", "type": "uint256"},
				{"name": "blockHash", "type": "string"},
				{"name": "networkHashrate", "type": "uint256"},
				{"name": "networkDifficulty", "type": "uint256"},
				{"name": "mempoolSize", "type": "uint256"},
				{"name": "peerCount", "type": "uint256"},
				{"name": "lastUpdated", "type": "uint256"}
			]
		}
	}
	"methods": {
		"setBitcoinPrice": {
			"name": "setBitcoinPrice",
			"inputs": [{"name": "price", "type": "uint256"}],
			"outputs": []
		},
		"getBitcoinPrice": {
			"name": "getBitcoinPrice",
			"inputs": [],
			"outputs": [{"name": "price", "type": "uint256"}]
		},
		"setBlockHeight": {
			"name": "setBlockHeight",
			"inputs": [{"name": "height", "type": "uint256"}],
			"outputs": []
		},
		"getBlockHeight": {
			"name": "getBlockHeight",
			"inputs": [],
			"outputs": [{"name": "height", "type": "uint256"}]
		},
		"setNetworkHashrate": {
			"name": "setNetworkHashrate",
			"inputs": [{"name": "hashrate", "type": "uint256"}],
			"outputs": []
		},
		"getNetworkHashrate": {
			"name": "getNetworkHashrate",
			"inputs": [],
			"outputs": [{"name": "hashrate", "type": "uint256"}]
		}
	}
}`
