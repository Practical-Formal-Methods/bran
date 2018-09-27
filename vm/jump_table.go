// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

type (
	executionFunc       func(pc *uint64, interpreter *EVMInterpreter, contract *Contract, memory *Memory, stack *Stack) ([]byte, error)
	gasFunc             func(params.GasTable, *EVM, *Contract, *Stack, *Memory, uint64) (uint64, error) // last parameter is the requested memory size as a uint64
	stackValidationFunc func(*Stack) error
	MemorySizeFunc      func(*Stack) *big.Int
)

var errGasUintOverflow = errors.New("gas uint64 overflow")

type Operation struct {
	// execute is the operation function
	Execute executionFunc
	// gasCost is the gas function and returns the gas required for execution
	gasCost gasFunc
	// validateStack validates the stack (size) for the operation
	ValidateStack stackValidationFunc
	// memorySize returns the memory size required for the operation
	MemorySize MemorySizeFunc

	halts   bool // indicates whether the operation should halt further execution
	jumps   bool // indicates whether the program counter should not increment
	writes  bool // determines whether this a state modifying operation
	Valid   bool // indication whether the retrieved operation is valid and known
	reverts bool // determines whether the operation reverts state (implicitly halts)
	returns bool // determines whether the operations sets the return data content
}

var (
	frontierInstructionSet       = newFrontierInstructionSet()
	homesteadInstructionSet      = newHomesteadInstructionSet()
	byzantiumInstructionSet      = NewByzantiumInstructionSet()
	constantinopleInstructionSet = NewConstantinopleInstructionSet()
)

// NewConstantinopleInstructionSet returns the frontier, homestead
// byzantium and contantinople instructions.
func NewConstantinopleInstructionSet() [256]Operation {
	// instructions that can be executed during the byzantium phase.
	instructionSet := NewByzantiumInstructionSet()
	instructionSet[SHL] = Operation{
		Execute:       opSHL,
		gasCost:       constGasFunc(GasFastestStep),
		ValidateStack: makeStackFunc(2, 1),
		Valid:         true,
	}
	instructionSet[SHR] = Operation{
		Execute:       opSHR,
		gasCost:       constGasFunc(GasFastestStep),
		ValidateStack: makeStackFunc(2, 1),
		Valid:         true,
	}
	instructionSet[SAR] = Operation{
		Execute:       opSAR,
		gasCost:       constGasFunc(GasFastestStep),
		ValidateStack: makeStackFunc(2, 1),
		Valid:         true,
	}
	instructionSet[EXTCODEHASH] = Operation{
		Execute:       opExtCodeHash,
		gasCost:       gasExtCodeHash,
		ValidateStack: makeStackFunc(1, 1),
		Valid:         true,
	}
	instructionSet[CREATE2] = Operation{
		Execute:       opCreate2,
		gasCost:       gasCreate2,
		ValidateStack: makeStackFunc(4, 1),
		MemorySize:    memoryCreate2,
		Valid:         true,
		writes:        true,
		returns:       true,
	}
	return instructionSet
}

// NewByzantiumInstructionSet returns the frontier, homestead and
// byzantium instructions.
func NewByzantiumInstructionSet() [256]Operation {
	// instructions that can be executed during the homestead phase.
	instructionSet := newHomesteadInstructionSet()
	instructionSet[STATICCALL] = Operation{
		Execute:       opStaticCall,
		gasCost:       gasStaticCall,
		ValidateStack: makeStackFunc(6, 1),
		MemorySize:    memoryStaticCall,
		Valid:         true,
		returns:       true,
	}
	instructionSet[RETURNDATASIZE] = Operation{
		Execute:       opReturnDataSize,
		gasCost:       constGasFunc(GasQuickStep),
		ValidateStack: makeStackFunc(0, 1),
		Valid:         true,
	}
	instructionSet[RETURNDATACOPY] = Operation{
		Execute:       opReturnDataCopy,
		gasCost:       gasReturnDataCopy,
		ValidateStack: makeStackFunc(3, 0),
		MemorySize:    memoryReturnDataCopy,
		Valid:         true,
	}
	instructionSet[REVERT] = Operation{
		Execute:       opRevert,
		gasCost:       gasRevert,
		ValidateStack: makeStackFunc(2, 0),
		MemorySize:    memoryRevert,
		Valid:         true,
		reverts:       true,
		returns:       true,
	}
	return instructionSet
}

// NewHomesteadInstructionSet returns the frontier and homestead
// instructions that can be executed during the homestead phase.
func newHomesteadInstructionSet() [256]Operation {
	instructionSet := newFrontierInstructionSet()
	instructionSet[DELEGATECALL] = Operation{
		Execute:       opDelegateCall,
		gasCost:       gasDelegateCall,
		ValidateStack: makeStackFunc(6, 1),
		MemorySize:    memoryDelegateCall,
		Valid:         true,
		returns:       true,
	}
	return instructionSet
}

// NewFrontierInstructionSet returns the frontier instructions
// that can be executed during the frontier phase.
func newFrontierInstructionSet() [256]Operation {
	return [256]Operation{
		STOP: {
			Execute:       opStop,
			gasCost:       constGasFunc(0),
			ValidateStack: makeStackFunc(0, 0),
			halts:         true,
			Valid:         true,
		},
		ADD: {
			Execute:       opAdd,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		MUL: {
			Execute:       opMul,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SUB: {
			Execute:       opSub,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		DIV: {
			Execute:       opDiv,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SDIV: {
			Execute:       opSdiv,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		MOD: {
			Execute:       opMod,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SMOD: {
			Execute:       opSmod,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		ADDMOD: {
			Execute:       opAddmod,
			gasCost:       constGasFunc(GasMidStep),
			ValidateStack: makeStackFunc(3, 1),
			Valid:         true,
		},
		MULMOD: {
			Execute:       opMulmod,
			gasCost:       constGasFunc(GasMidStep),
			ValidateStack: makeStackFunc(3, 1),
			Valid:         true,
		},
		EXP: {
			Execute:       opExp,
			gasCost:       gasExp,
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SIGNEXTEND: {
			Execute:       opSignExtend,
			gasCost:       constGasFunc(GasFastStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		LT: {
			Execute:       opLt,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		GT: {
			Execute:       opGt,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SLT: {
			Execute:       opSlt,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SGT: {
			Execute:       opSgt,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		EQ: {
			Execute:       opEq,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		ISZERO: {
			Execute:       opIszero,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		AND: {
			Execute:       opAnd,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		XOR: {
			Execute:       opXor,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		OR: {
			Execute:       opOr,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		NOT: {
			Execute:       opNot,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		BYTE: {
			Execute:       opByte,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(2, 1),
			Valid:         true,
		},
		SHA3: {
			Execute:       opSha3,
			gasCost:       gasSha3,
			ValidateStack: makeStackFunc(2, 1),
			MemorySize:    memorySha3,
			Valid:         true,
		},
		ADDRESS: {
			Execute:       opAddress,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		BALANCE: {
			Execute:       opBalance,
			gasCost:       gasBalance,
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		ORIGIN: {
			Execute:       opOrigin,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		CALLER: {
			Execute:       opCaller,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		CALLVALUE: {
			Execute:       opCallValue,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		CALLDATALOAD: {
			Execute:       opCallDataLoad,
			gasCost:       constGasFunc(GasFastestStep),
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		CALLDATASIZE: {
			Execute:       opCallDataSize,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		CALLDATACOPY: {
			Execute:       opCallDataCopy,
			gasCost:       gasCallDataCopy,
			ValidateStack: makeStackFunc(3, 0),
			MemorySize:    memoryCallDataCopy,
			Valid:         true,
		},
		CODESIZE: {
			Execute:       opCodeSize,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		CODECOPY: {
			Execute:       opCodeCopy,
			gasCost:       gasCodeCopy,
			ValidateStack: makeStackFunc(3, 0),
			MemorySize:    memoryCodeCopy,
			Valid:         true,
		},
		GASPRICE: {
			Execute:       opGasprice,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		EXTCODESIZE: {
			Execute:       opExtCodeSize,
			gasCost:       gasExtCodeSize,
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		EXTCODECOPY: {
			Execute:       opExtCodeCopy,
			gasCost:       gasExtCodeCopy,
			ValidateStack: makeStackFunc(4, 0),
			MemorySize:    memoryExtCodeCopy,
			Valid:         true,
		},
		BLOCKHASH: {
			Execute:       opBlockhash,
			gasCost:       constGasFunc(GasExtStep),
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		COINBASE: {
			Execute:       opCoinbase,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		TIMESTAMP: {
			Execute:       opTimestamp,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		NUMBER: {
			Execute:       opNumber,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		DIFFICULTY: {
			Execute:       opDifficulty,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		GASLIMIT: {
			Execute:       opGasLimit,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		POP: {
			Execute:       opPop,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(1, 0),
			Valid:         true,
		},
		MLOAD: {
			Execute:       opMload,
			gasCost:       gasMLoad,
			ValidateStack: makeStackFunc(1, 1),
			MemorySize:    memoryMLoad,
			Valid:         true,
		},
		MSTORE: {
			Execute:       opMstore,
			gasCost:       gasMStore,
			ValidateStack: makeStackFunc(2, 0),
			MemorySize:    memoryMStore,
			Valid:         true,
		},
		MSTORE8: {
			Execute:       opMstore8,
			gasCost:       gasMStore8,
			MemorySize:    memoryMStore8,
			ValidateStack: makeStackFunc(2, 0),

			Valid: true,
		},
		SLOAD: {
			Execute:       opSload,
			gasCost:       gasSLoad,
			ValidateStack: makeStackFunc(1, 1),
			Valid:         true,
		},
		SSTORE: {
			Execute:       opSstore,
			gasCost:       gasSStore,
			ValidateStack: makeStackFunc(2, 0),
			Valid:         true,
			writes:        true,
		},
		JUMP: {
			Execute:       opJump,
			gasCost:       constGasFunc(GasMidStep),
			ValidateStack: makeStackFunc(1, 0),
			jumps:         true,
			Valid:         true,
		},
		JUMPI: {
			Execute:       opJumpi,
			gasCost:       constGasFunc(GasSlowStep),
			ValidateStack: makeStackFunc(2, 0),
			jumps:         true,
			Valid:         true,
		},
		PC: {
			Execute:       opPc,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		MSIZE: {
			Execute:       opMsize,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		GAS: {
			Execute:       opGas,
			gasCost:       constGasFunc(GasQuickStep),
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		JUMPDEST: {
			Execute:       opJumpdest,
			gasCost:       constGasFunc(params.JumpdestGas),
			ValidateStack: makeStackFunc(0, 0),
			Valid:         true,
		},
		PUSH1: {
			Execute:       makePush(1, 1),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH2: {
			Execute:       makePush(2, 2),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH3: {
			Execute:       makePush(3, 3),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH4: {
			Execute:       makePush(4, 4),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH5: {
			Execute:       makePush(5, 5),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH6: {
			Execute:       makePush(6, 6),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH7: {
			Execute:       makePush(7, 7),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH8: {
			Execute:       makePush(8, 8),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH9: {
			Execute:       makePush(9, 9),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH10: {
			Execute:       makePush(10, 10),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH11: {
			Execute:       makePush(11, 11),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH12: {
			Execute:       makePush(12, 12),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH13: {
			Execute:       makePush(13, 13),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH14: {
			Execute:       makePush(14, 14),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH15: {
			Execute:       makePush(15, 15),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH16: {
			Execute:       makePush(16, 16),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH17: {
			Execute:       makePush(17, 17),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH18: {
			Execute:       makePush(18, 18),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH19: {
			Execute:       makePush(19, 19),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH20: {
			Execute:       makePush(20, 20),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH21: {
			Execute:       makePush(21, 21),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH22: {
			Execute:       makePush(22, 22),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH23: {
			Execute:       makePush(23, 23),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH24: {
			Execute:       makePush(24, 24),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH25: {
			Execute:       makePush(25, 25),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH26: {
			Execute:       makePush(26, 26),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH27: {
			Execute:       makePush(27, 27),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH28: {
			Execute:       makePush(28, 28),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH29: {
			Execute:       makePush(29, 29),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH30: {
			Execute:       makePush(30, 30),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH31: {
			Execute:       makePush(31, 31),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		PUSH32: {
			Execute:       makePush(32, 32),
			gasCost:       gasPush,
			ValidateStack: makeStackFunc(0, 1),
			Valid:         true,
		},
		DUP1: {
			Execute:       makeDup(1),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(1),
			Valid:         true,
		},
		DUP2: {
			Execute:       makeDup(2),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(2),
			Valid:         true,
		},
		DUP3: {
			Execute:       makeDup(3),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(3),
			Valid:         true,
		},
		DUP4: {
			Execute:       makeDup(4),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(4),
			Valid:         true,
		},
		DUP5: {
			Execute:       makeDup(5),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(5),
			Valid:         true,
		},
		DUP6: {
			Execute:       makeDup(6),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(6),
			Valid:         true,
		},
		DUP7: {
			Execute:       makeDup(7),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(7),
			Valid:         true,
		},
		DUP8: {
			Execute:       makeDup(8),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(8),
			Valid:         true,
		},
		DUP9: {
			Execute:       makeDup(9),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(9),
			Valid:         true,
		},
		DUP10: {
			Execute:       makeDup(10),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(10),
			Valid:         true,
		},
		DUP11: {
			Execute:       makeDup(11),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(11),
			Valid:         true,
		},
		DUP12: {
			Execute:       makeDup(12),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(12),
			Valid:         true,
		},
		DUP13: {
			Execute:       makeDup(13),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(13),
			Valid:         true,
		},
		DUP14: {
			Execute:       makeDup(14),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(14),
			Valid:         true,
		},
		DUP15: {
			Execute:       makeDup(15),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(15),
			Valid:         true,
		},
		DUP16: {
			Execute:       makeDup(16),
			gasCost:       gasDup,
			ValidateStack: makeDupStackFunc(16),
			Valid:         true,
		},
		SWAP1: {
			Execute:       makeSwap(1),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(2),
			Valid:         true,
		},
		SWAP2: {
			Execute:       makeSwap(2),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(3),
			Valid:         true,
		},
		SWAP3: {
			Execute:       makeSwap(3),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(4),
			Valid:         true,
		},
		SWAP4: {
			Execute:       makeSwap(4),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(5),
			Valid:         true,
		},
		SWAP5: {
			Execute:       makeSwap(5),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(6),
			Valid:         true,
		},
		SWAP6: {
			Execute:       makeSwap(6),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(7),
			Valid:         true,
		},
		SWAP7: {
			Execute:       makeSwap(7),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(8),
			Valid:         true,
		},
		SWAP8: {
			Execute:       makeSwap(8),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(9),
			Valid:         true,
		},
		SWAP9: {
			Execute:       makeSwap(9),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(10),
			Valid:         true,
		},
		SWAP10: {
			Execute:       makeSwap(10),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(11),
			Valid:         true,
		},
		SWAP11: {
			Execute:       makeSwap(11),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(12),
			Valid:         true,
		},
		SWAP12: {
			Execute:       makeSwap(12),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(13),
			Valid:         true,
		},
		SWAP13: {
			Execute:       makeSwap(13),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(14),
			Valid:         true,
		},
		SWAP14: {
			Execute:       makeSwap(14),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(15),
			Valid:         true,
		},
		SWAP15: {
			Execute:       makeSwap(15),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(16),
			Valid:         true,
		},
		SWAP16: {
			Execute:       makeSwap(16),
			gasCost:       gasSwap,
			ValidateStack: makeSwapStackFunc(17),
			Valid:         true,
		},
		LOG0: {
			Execute:       makeLog(0),
			gasCost:       makeGasLog(0),
			ValidateStack: makeStackFunc(2, 0),
			MemorySize:    memoryLog,
			Valid:         true,
			writes:        true,
		},
		LOG1: {
			Execute:       makeLog(1),
			gasCost:       makeGasLog(1),
			ValidateStack: makeStackFunc(3, 0),
			MemorySize:    memoryLog,
			Valid:         true,
			writes:        true,
		},
		LOG2: {
			Execute:       makeLog(2),
			gasCost:       makeGasLog(2),
			ValidateStack: makeStackFunc(4, 0),
			MemorySize:    memoryLog,
			Valid:         true,
			writes:        true,
		},
		LOG3: {
			Execute:       makeLog(3),
			gasCost:       makeGasLog(3),
			ValidateStack: makeStackFunc(5, 0),
			MemorySize:    memoryLog,
			Valid:         true,
			writes:        true,
		},
		LOG4: {
			Execute:       makeLog(4),
			gasCost:       makeGasLog(4),
			ValidateStack: makeStackFunc(6, 0),
			MemorySize:    memoryLog,
			Valid:         true,
			writes:        true,
		},
		CREATE: {
			Execute:       opCreate,
			gasCost:       gasCreate,
			ValidateStack: makeStackFunc(3, 1),
			MemorySize:    memoryCreate,
			Valid:         true,
			writes:        true,
			returns:       true,
		},
		CALL: {
			Execute:       opCall,
			gasCost:       gasCall,
			ValidateStack: makeStackFunc(7, 1),
			MemorySize:    memoryCall,
			Valid:         true,
			returns:       true,
		},
		CALLCODE: {
			Execute:       opCallCode,
			gasCost:       gasCallCode,
			ValidateStack: makeStackFunc(7, 1),
			MemorySize:    memoryCall,
			Valid:         true,
			returns:       true,
		},
		RETURN: {
			Execute:       opReturn,
			gasCost:       gasReturn,
			ValidateStack: makeStackFunc(2, 0),
			MemorySize:    memoryReturn,
			halts:         true,
			Valid:         true,
		},
		SELFDESTRUCT: {
			Execute:       opSuicide,
			gasCost:       gasSuicide,
			ValidateStack: makeStackFunc(1, 0),
			halts:         true,
			Valid:         true,
			writes:        true,
		},
	}
}
