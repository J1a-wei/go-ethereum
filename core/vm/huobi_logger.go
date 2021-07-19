package vm

import (
	"fmt"
	"time"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/rlp"
)

type HuobiLogger struct {
	*StructLogger

	hash           common.Hash
	caredOpInDepth map[int]bool

	lastOp *tmpLogger
}

type tmpLogger struct {
	pc        uint64
	op        OpCode
	gas, cost uint64
	memory    *Memory
	stack     []uint256.Int
	contract  *Contract
	depth     int
	err       error
}

var CaredOps = map[OpCode]bool{
	CALL:         true,
	CREATE:       true,
	CREATE2:      true,
	CALLCODE:     true,
	DELEGATECALL: true,
	STATICCALL:   true,
	//SELFDESTRUCT: true,
}

// NewStructLogger returns a new logger
func NewHuobiLogger(cfg *LogConfig) *HuobiLogger {
	logger := &HuobiLogger{}
	logger.StructLogger = NewStructLogger(cfg)

	return logger
}

func (l *HuobiLogger) CaptureTx(hash common.Hash) error {
	l.hash = hash
	l.caredOpInDepth = make(map[int]bool)
	l.lastOp = nil
	return nil
}

func (l *HuobiLogger) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, storage []byte, depth int, err error) {
	memory := scope.Memory
	stack := scope.Stack
	contract := scope.Contract
	stk := make([]uint256.Int, 0, 4)
	stackL := stack.len()
	for i, item := range stack.Data() {
		if stackL > 4 && i < stackL-4 { //  op call need last 4 stack items
			continue
		}

		stk = append(stk, item)
	}

	l.addLogger(&tmpLogger{pc, op, gas, cost, memory, stk, contract, depth, err})
}

func (l *HuobiLogger) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error) {
	if err != nil {
		l.addLogger(&tmpLogger{pc, op, gas, cost, nil, make([]uint256.Int, 0), scope.Contract, depth, err})
	}
}

func (l *HuobiLogger) addLogger(tmpL *tmpLogger) {
	//whether or not to append lastOp
	if l.lastOp != nil &&
		(l.lastOp.err != nil || // last op has err
			CaredOps[l.lastOp.op] || // is cared op
			l.caredOpInDepth[l.lastOp.depth] || // CALL  CALL  EQUAL  PUSH(lastOP) POP(tmpL)
			l.lastOp.depth < tmpL.depth) { // go deeper

		l.logs = append(l.logs, StructLog{l.lastOp.contract.Address(), l.lastOp.pc, l.lastOp.op, l.lastOp.gas, l.lastOp.cost, nil, 0, l.lastOp.stack, nil, nil, l.lastOp.depth, 0, l.lastOp.err})
	}

	if l.lastOp != nil {
		l.caredOpInDepth[l.lastOp.depth] = CaredOps[l.lastOp.op]
	}

	l.lastOp = tmpL
}

func (l *HuobiLogger) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {
	if l.lastOp != nil &&
		(l.lastOp.err != nil || CaredOps[l.lastOp.op] || l.caredOpInDepth[l.lastOp.depth]) {
		l.logs = append(l.logs, StructLog{l.lastOp.contract.Address(), l.lastOp.pc, l.lastOp.op, l.lastOp.gas, l.lastOp.cost, nil, 0, l.lastOp.stack, nil, nil, l.lastOp.depth, 0, l.lastOp.err})
	}

	//l.err = err
	//if l.cfg.Debug {
	//resp := &LogRes{}
	//resp.Hash = l.hash.String()
	//resp.Logs = FormatLogs(l.logs)

	//rlp.EncodeToBytes(resp)

	//tmp, err := json.MarshalIndent(resp, "", "\t")
	//if err != nil {
	//	fmt.Printf(" error: %v\n", err)
	//} else {
	//	fmt.Println(string(tmp))
	//}
	//}
}

func (l *HuobiLogger) FinishCapture() {
	l.caredOpInDepth = make(map[int]bool)
	l.logs = make([]StructLog, 0)
	l.lastOp = nil
}

type LogRes struct {
	Hash          string         `json:"hash"`
	ReceiptStatus uint64         `json:"receiptStatus"`
	TxToAddr      string         `json:"ToAddr"`
	Logs          []StructLogRes `json:"logs"`
}

func (l *HuobiLogger) GetTxLogs() (resp *LogRes, err error) {
	resp = &LogRes{}
	resp.Hash = l.hash.String()
	resp.Logs = FormatLogs(l.logs)

	rlp.EncodeToBytes(resp)

	return
}

type StructLogRes struct {
	From    string   `json:"from"`
	Pc      uint64   `json:"pc"`
	Op      string   `json:"op"`
	Gas     uint64   `json:"gas"`
	GasCost uint64   `json:"gasCost"`
	Depth   uint     `json:"depth"`
	Stack   []string `json:"stack,omitempty"`
	ErrMsg  string   `json:"errMsg,omitempty"`
}

// formatLogs formats EVM returned structured logs for json output
func FormatLogs(logs []StructLog) []StructLogRes {
	formatted := make([]StructLogRes, len(logs))
	for index, trace := range logs {
		formatted[index] = StructLogRes{
			From:    trace.From.String(),
			Pc:      trace.Pc,
			Op:      trace.Op.String(),
			Gas:     trace.Gas,
			GasCost: trace.GasCost,
			Depth:   uint(trace.Depth),
			ErrMsg:  trace.ErrorString(),
		}
		if trace.Stack != nil {
			stack := make([]string, len(trace.Stack))
			for i, stackValue := range trace.Stack {
				stack[i] = fmt.Sprintf("%x", math.PaddedBigBytes(stackValue.ToBig(), 32))
			}
			formatted[index].Stack = stack
		}
	}
	return formatted
}
