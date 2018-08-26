// Copyright 2018 MPI-SWS and Valentin Wuestholz

// This file is part of Bran.
//
// Bran is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Bran is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Bran.  If not, see <https://www.gnu.org/licenses/>.

package analysis

import (
	"encoding/binary"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/params"
	"github.com/wuestholz/bran/vm"
	"hash"
	"hash/fnv"
	"time"
)

var InvalidOpcodeFail = "invalid-opcode"
var UnsupportedOpcodeFail = "unsupported-opcode"
var MemoryOverflowFail = "memory-overflow-failure"
var TopMemoryResizeFail = "top-memory-resize-failure"
var TopStackFail = "top-stack"
var StackValidationFail = "invalid-stack"
var JumpToTopFail = "jump-to-top"
var TopOffsetFail = "top-offset-failure"
var PrefixComputationFail = "prefix-computation-failure"
var StepExecFail = "step-execution-failure"
var InternalFail = "internal-failure"

type LookaheadAnalyzer struct {
	cpAnalyzer        *constPropAnalyzer
	contract          *vm.Contract
	codeHash          common.Hash
	prefix            execPrefix
	prefixHash        hash.Hash32
	cachedResults     map[prefixHash]result
	coveredAssertions map[string]bool

	numSuccess    uint64
	numFail       uint64
	numPrefixFail uint64
	failureCauses map[string]uint64
	numErrors     uint64
	numSameLID    uint64
	time          time.Duration
	startTime     time.Time
}

type prefixHash uint32

func NewLookaheadAnalyzer() *LookaheadAnalyzer {
	return &LookaheadAnalyzer{
		failureCauses:     map[string]uint64{},
		cachedResults:     map[prefixHash]result{},
		coveredAssertions: map[string]bool{},
	}
}

func (a *LookaheadAnalyzer) Start(code, codeHash []byte) {
	addr := common.HexToAddress(MagicString("0x0123456789abcdef"))
	a.codeHash = common.BytesToHash(codeHash)
	a.contract = newDummyContract(addr, code, a.codeHash)
	a.prefix = nil
	a.prefixHash = fnv.New32()
}

func (a *LookaheadAnalyzer) AppendPrefixSummary(summaryId string) {
	if a.prefixHash != nil {
		a.prefixHash.Write([]byte(summaryId))
	}
}

func (a *LookaheadAnalyzer) AppendPrefixInstruction(pc uint64) {
	a.startTimer()
	defer a.stopTimer()
	if a.prefixHash != nil {
		a.prefix = append(a.prefix, pcType(pc))
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, pc)
		a.prefixHash.Write(b)
	}
}

func (a *LookaheadAnalyzer) CanIgnoreSuffix() (canIgnore, avoidRetry bool, justification, prefixId string, err error) {
	a.startTimer()
	defer a.stopTimer()

	if a.prefixHash == nil {
		return false, false, "", "", fmt.Errorf("analysis not yet started")
	}

	pHash := prefixHash(a.prefixHash.Sum32())
	pid := fmt.Sprintf("%x", pHash)

	if cachedRes, found := a.cachedResults[pHash]; found {
		if cachedRes.mayFail {
			return false, cachedRes.inPrefix, cachedRes.failureCause, pid, nil
		}
		a.recordSuccess()
		return true, cachedRes.inPrefix, "", pid, nil
	}

	if a.cpAnalyzer == nil {
		evm := newDummyEVM()
		interpreter, ok := evm.Interpreter().(*vm.EVMInterpreter)
		if !ok {
			return false, true, "", pid, fmt.Errorf("expected compatible EVM interpreter")
		}
		a.cpAnalyzer = newConstPropAnalyzer(a.contract, a.codeHash, interpreter, a)
	}

	res, prefixErr, suffixErr := a.cpAnalyzer.Analyze(a.prefix)
	if prefixErr != nil {
		a.recordError()
		return false, true, "", pid, prefixErr
	}
	if suffixErr != nil {
		a.recordError()
		return false, false, "", pid, suffixErr
	}

	// We cache both kinds of results, but not errors.
	a.cachedResults[pHash] = result{
		mayFail:      res.mayFail,
		failureCause: res.failureCause,
		inPrefix:     res.inPrefix,
	}

	if res.mayFail {
		a.recordFailure(res.failureCause, res.inPrefix)
		return false, res.inPrefix, res.failureCause, pid, nil
	}

	a.recordSuccess()
	return true, false, "", pid, nil
}

func (a *LookaheadAnalyzer) RecordCoveredAssertion(codeHash []byte, pc uint64) {
	a.coveredAssertions[fmt.Sprintf("%032x:%x", codeHash, pc)] = true
}

func (a *LookaheadAnalyzer) IsCoveredAssertion(codeHash common.Hash, pc uint64) bool {
	return a.coveredAssertions[fmt.Sprintf("%032x:%x", codeHash, pc)]
}

func (a *LookaheadAnalyzer) startTimer() {
	a.startTime = time.Now()
}

func (a *LookaheadAnalyzer) stopTimer() {
	a.time += time.Now().Sub(a.startTime)
}

func (a *LookaheadAnalyzer) RecordPathWithSameLID() {
	a.numSameLID++
}

func (a *LookaheadAnalyzer) recordSuccess() {
	a.numSuccess++
}

func (a *LookaheadAnalyzer) recordFailure(cause string, inPrefix bool) {
	if inPrefix {
		a.numPrefixFail++
	} else {
		a.numFail++
	}
	a.failureCauses[cause]++
}

func (a *LookaheadAnalyzer) recordError() {
	a.numErrors++
}

func (a *LookaheadAnalyzer) NumSuccess() uint64 {
	return a.numSuccess
}

func (a *LookaheadAnalyzer) NumFail() uint64 {
	return a.numFail
}

func (a *LookaheadAnalyzer) NumPrefixFail() uint64 {
	return a.numPrefixFail
}

func (a *LookaheadAnalyzer) NumErrors() uint64 {
	return a.numErrors
}

func (a *LookaheadAnalyzer) NumPathsWithSameLID() uint64 {
	return a.numSameLID
}

func (a *LookaheadAnalyzer) Time() time.Duration {
	return a.time
}

func (a *LookaheadAnalyzer) FailureCauses() map[string]uint64 {
	fcs := map[string]uint64{}
	for cause, cnt := range a.failureCauses {
		fcs[cause] = cnt
	}
	return fcs
}

type dummyContractRef struct {
	address common.Address
}

func (d dummyContractRef) Address() common.Address {
	return common.BytesToAddress(d.address.Bytes())
}

// newDummyContract creates a mock contract that only remembers its address.
func newDummyContract(address common.Address, code []byte, codeHash common.Hash) *vm.Contract {
	dummyRef := dummyContractRef{address: address}
	val := topVal()
	ct := vm.NewContract(dummyRef, dummyRef, val, MagicUInt64(0xffffffffffffffff))
	ct.SetCode(codeHash, code)
	return ct
}

// newDummyEVM creates a EVM object we can use to run code.
func newDummyEVM() *vm.EVM {
	ctx := vm.Context{}
	evmConfig := vm.Config{JumpTable: vm.NewByzantiumInstructionSet()}
	chainConfig := &params.ChainConfig{}
	dummyStateDB := &state.StateDB{}
	return vm.NewEVM(ctx, dummyStateDB, chainConfig, evmConfig)
}
