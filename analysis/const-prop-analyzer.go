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
	maxDisjuncts       int
	failOnTopMemResize bool
	useBoundedJoins    bool
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
		useBoundedJoins:    MagicBool(false),
		maxDisjuncts:       MagicInt(0),
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
	states := map[string]absState{}
	keys := map[pcType]map[string]bool{}
	ppcMap := newPrevPCMap()
	var worklist []string
	workset := map[string]pcType{}

	addNewStates := func(prevPC pcType, newStates []pcAndSt) {
		for _, st := range newStates {
			pc := st.pc
			ppcMap.addPrevPC(pc, prevPC)

			newState := st.st.withStackCopy().withMemCopy()

			stSize := -1
			if !newState.isBot && !newState.stack.isTop && newState.stack.stack != nil {
				stSize = newState.stack.len()
			}
			loc := fmt.Sprintf("%x:%x", pc, stSize)

			oldState, exists := states[loc]
			ks := keys[pc]
			if ks == nil {
				ks = map[string]bool{}
			}
			numDisjs := len(ks)
			if !exists && a.maxDisjuncts <= numDisjs {
				loc = fmt.Sprintf("%x:%x", pc, -1)
				oldState, exists = states[loc]
			}
			if exists {
				var diff bool
				newState, diff = joinStates(oldState, newState)
				if !diff || a.useBoundedJoins {
					continue
				}
			}

			states[loc] = newState
			ks[loc] = true
			keys[pc] = ks
			if _, exists := workset[loc]; !exists {
				worklist = append(worklist, loc)
				workset[loc] = pc
			}
		}
	}

	popState := func() (absState, pcType) {
		ret := worklist[0]
		worklist = worklist[1:]
		pc := workset[ret]
		delete(workset, ret)
		return states[ret], pc
	}

	prefixLen := len(execPrefix)
	if 0 < prefixLen {
		lastPrefixPC := execPrefix[prefixLen-1]
		addNewStates(lastPrefixPC, prefixRes.postStates)
	}

	for 0 < len(worklist) {
		st, pc := popState()
		if st.isBot {
			continue
		}
		opcode := a.contract.GetOp(uint64(pc))
		res, err := a.step(pc, ppcMap, st, concJt[opcode], opcode, absJt, false)
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
		currRes, err = a.step(pc, ppcMap, currSt, concJt[opcode], opcode, absJt, true)
		if err != nil {
			currRes = emptyRes()
		}
	}
	return currRes, nil
}

func (a *constPropAnalyzer) step(pc pcType, ppcMap *prevPCMap, st absState, conc vm.Operation, op vm.OpCode, jt absJumpTable, ignoreTargets bool) (stepRes, error) {
	absOp := jt[op]
	if absOp.valid != conc.Valid {
		return failRes(InternalFail), nil
	}

	if a.analyzer.IsTargetingAssertionFailed() {
		if op == vm.LOG1 {
			// We look for the following event type:
			// event AssertionFailed(string message);
			isAssertionFailed := true
			if st.isBot {
				isAssertionFailed = false
			} else if !st.stack.isTop && 3 <= st.stack.len() {
				topic := st.stack.stack.Back(2)
				magicTopic, _ := math.ParseBig256("0xb42604cb105a16c8f6db8a41e6b00c0c1b4826465e8bc504b3eb3e88b3e6a4a0")
				if !isTop(topic) && topic.Cmp(magicTopic) != 0 {
					isAssertionFailed = false
				}
			}
			if !ignoreTargets && isAssertionFailed {
				return failRes(ReachedAssertionFailed), nil
			}
		}
	} else if a.analyzer.HasTargetInstructions() {
		if !ignoreTargets && a.analyzer.IsTargetInstruction(a.codeHash, uint64(pc)) {
			return failRes(ReachedTargetInstructionFail), nil
		}
		if !absOp.valid {
			return emptyRes(), nil
		}
	} else {
		if !absOp.valid {
			switch op {
			case 0xfe:
				if a.analyzer.IsCoveredAssertion(a.codeHash, uint64(pc)) {
					// No need to report a failure since the assertion has already been covered.
					return emptyRes(), nil
				}
			}
			if !ignoreTargets {
				return failRes(InvalidOpcodeFail), nil
			}
			return emptyRes(), nil
		}
	}

	if st.stack.isTop {
		return failRes(TopStackFail), nil
	}

	if stLen := st.stack.len(); stLen < conc.MinStack || conc.MaxStack < stLen {
		return failRes(StackValidationFail), nil
	}

	postMem := st.mem
	if absOp.memSize != nil {
		if conc.MemorySize == nil {
			return failRes(InternalFail), nil
		}
		msize, overflow, isUnknown, msErr := absOp.memSize(st.stack, conc.MemorySize)
		if msErr != nil {
			return failRes(InternalFail), nil
		}

		if isUnknown {
			if a.failOnTopMemResize {
				return failRes(TopMemoryResizeFail), nil
			}
			postMem = topMem()
		} else {
			if overflow {
				return failRes(MemoryOverflowFail), nil
			}

			var msz uint64
			if msz, overflow = math.SafeMul(vm.ToWordSize(msize), 32); overflow {
				return failRes(MemoryOverflowFail), nil
			}

			if 0 < msz {
				postMem = st.mem.clone()
				postMem.resize(msz)
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
