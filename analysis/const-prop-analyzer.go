// MPI-SWS, Valentin Wuestholz, and ConsenSys AG

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
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"

	"github.com/practical-formal-methods/bran/vm"
)

type pcType uint64

type execPrefix map[int]pcType

type concJumpTable [256]vm.Operation

type result struct {
	mayFail      bool
	failureCause string
	avoidRetry   bool
}

func noFail() result {
	return result{}
}

func mayFail(cause string) result {
	return result{
		mayFail:      true,
		failureCause: cause,
	}
}

func prefixMayFail(cause string) result {
	return result{
		mayFail:      true,
		failureCause: cause,
		avoidRetry:   true,
	}
}

type prevPCMap struct {
	prevPC        map[pcType]pcType
	multiplePreds map[pcType]bool
}

func newPrevPCMap() *prevPCMap {
	return &prevPCMap{
		prevPC:        map[pcType]pcType{},
		multiplePreds: map[pcType]bool{},
	}
}

func (m *prevPCMap) addPrevPC(currPc, prevPc pcType) {
	if !m.multiplePreds[currPc] {
		ppc, exists := m.prevPC[currPc]
		if !exists {
			m.prevPC[currPc] = prevPc
		} else if ppc != prevPc {
			delete(m.prevPC, currPc)
			m.multiplePreds[currPc] = true
		}
	}
}

func (m *prevPCMap) getPrevPC(pc pcType) (pcType, bool) {
	ppc, exists := m.prevPC[pc]
	return ppc, exists
}

type constPropAnalyzer struct {
	contract           *vm.Contract
	codeHash           common.Hash
	interpreter        *vm.EVMInterpreter
	analyzer           *LookaheadAnalyzer
	failOnTopMemResize bool
	verbose            bool
}

func newConstPropAnalyzer(contract *vm.Contract, codeHash common.Hash, interpreter *vm.EVMInterpreter, analyzer *LookaheadAnalyzer) *constPropAnalyzer {
	return &constPropAnalyzer{
		contract:           contract,
		codeHash:           codeHash,
		interpreter:        interpreter,
		analyzer:           analyzer,
		failOnTopMemResize: MagicBool(false),
		verbose:            MagicBool(false),
	}
}

func (a *constPropAnalyzer) Analyze(execPrefix execPrefix) (result, error, error) {
	if a.verbose {
		var pre []uint64
		idx := 0
		for true {
			pc, exists := execPrefix[idx]
			if !exists {
				break
			}
			pre = append(pre, uint64(pc))
			idx++
		}
		fmt.Printf("prefix: %#v\n", pre)
		fmt.Printf("code: %x\n", a.contract.Code)
	}

	concJt := a.interpreter.Cfg.JumpTable
	absJtPrefix := newAbsJumpTable(true)
	prefixRes, err := a.calculatePrecondition(concJt, absJtPrefix, execPrefix)
	if err != nil {
		return prefixMayFail(PrefixComputationFail), err, nil
	}
	if prefixRes.mayFail {
		return prefixMayFail(fmt.Sprintf("%v(%v)", PrefixComputationFail, prefixRes.failureCause)), nil, nil
	}

	absJt := newAbsJumpTable(false)
	states := map[pcType]absState{}
	ppcMap := newPrevPCMap()
	var worklist []pcType
	workset := map[pcType]bool{}

	stOrBot := func(pc pcType) absState {
		st, ok := states[pc]
		if !ok {
			return botState()
		}
		return st
	}

	addNewStates := func(pc pcType, newStates []pcAndSt) {
		for _, st := range newStates {
			ppcMap.addPrevPC(st.pc, pc)
			newSt, diff := joinStates(stOrBot(st.pc), st.st)
			if diff {
				states[st.pc] = newSt
				if !workset[st.pc] {
					worklist = append(worklist, st.pc)
					workset[st.pc] = true
				}
			}
		}
	}

	popState := func() pcType {
		ret := worklist[0]
		worklist = worklist[1:]
		delete(workset, ret)
		return ret
	}

	prefixLen := len(execPrefix)
	if 0 < prefixLen {
		lastPrefixPC := execPrefix[prefixLen-1]
		addNewStates(lastPrefixPC, prefixRes.postStates)
	}

	for 0 < len(worklist) {
		pc := popState()
		opcode := a.contract.GetOp(uint64(pc))
		st := stOrBot(pc)
		if st.isBot {
			continue
		}
		res, err := a.step(pc, ppcMap, st, concJt[opcode], opcode, absJt)
		if err != nil {
			return mayFail(StepExecFail), nil, err
		}
		if res.mayFail {
			return mayFail(res.failureCause), nil, nil
		}
		addNewStates(pc, res.postStates)
	}

	return noFail(), nil, nil
}

func (a *constPropAnalyzer) calculatePrecondition(concJt concJumpTable, absJt absJumpTable, execPrefix execPrefix) (stepRes, error) {
	ppcMap := newPrevPCMap()
	currRes := initRes()
	for idx := 0; true; idx++ {
		pc, exists := execPrefix[idx]
		if !exists {
			break
		}

		// Select from the results only the state that matchesBackwards the next pc in the prefix.
		currSt := botState()
		for _, st := range currRes.postStates {
			if st.pc == pc {
				if 0 < idx {
					ppc := execPrefix[idx-1]
					ppcMap.addPrevPC(pc, ppc)
				}
				currSt, _ = joinStates(currSt, st.st)
			}
		}
		if currSt.isBot {
			return emptyRes(), fmt.Errorf("expected feasible prefix")
		}
		opcode := a.contract.GetOp(uint64(pc))
		var err error
		currRes, err = a.step(pc, ppcMap, currSt, concJt[opcode], opcode, absJt)
		if err != nil {
			return failRes(StepExecFail), err
		}
		if currRes.mayFail {
			return currRes, nil
		}
	}
	return currRes, nil
}

func (a *constPropAnalyzer) step(pc pcType, ppcMap *prevPCMap, st absState, conc vm.Operation, op vm.OpCode, jt absJumpTable) (stepRes, error) {
	absOp := jt[op]
	if absOp.valid != conc.Valid {
		return failRes(InternalFail), nil
	}
	if !absOp.valid {
		switch op {
		case 0xfe:
			if a.analyzer.IsCoveredAssertion(a.codeHash, uint64(pc)) {
				// No need to report a failure since the assertion has already been covered.
				return emptyRes(), nil
			}
		}

		return failRes(InvalidOpcodeFail), nil
	}

	if st.stack.isTop {
		return failRes(TopStackFail), nil
	}
	if conc.ValidateStack == nil {
		return failRes(InternalFail), nil
	}
	// We implicitly assume that all validation functions just look at the stack size and not its contents.
	if valErr := conc.ValidateStack(st.stack.stack); valErr != nil {
		return failRes(StackValidationFail), nil
	}

	postMem := st.mem
	if absOp.memSize != nil {
		if conc.MemorySize == nil {
			return failRes(InternalFail), nil
		}
		msize, msErr := absOp.memSize(st.stack, conc.MemorySize)
		if msErr != nil {
			return failRes(InternalFail), nil
		}

		if isTop(msize) {
			if a.failOnTopMemResize {
				return failRes(TopMemoryResizeFail), nil
			}
			postMem = topMem()
		} else {
			sz, overflow := vm.BigUint64(msize)
			if overflow {
				return failRes(MemoryOverflowFail), nil
			}

			var sz2 uint64
			if sz2, overflow = math.SafeMul(vm.ToWordSize(sz), 32); overflow {
				return failRes(MemoryOverflowFail), nil
			}

			if 0 < sz2 {
				postMem = st.mem.clone()
				postMem.resize(sz2)
			}
		}
	}

	postSt := absState{
		stack: st.stack,
		mem:   postMem,
	}
	env := execEnv{
		pc:          &pc,
		interpreter: a.interpreter,
		contract:    a.contract,
		ppcMap:      ppcMap,
		st:          postSt,
		conc:        conc,
	}
	return absOp.exec(env)
}
