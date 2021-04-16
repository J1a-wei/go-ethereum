package huobi

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

type logHandler func(hash string, receiptStatus uint64, index int, log vm.StructLogRes, node *node) *TransferTx

func getLogHandler(op vm.OpCode) logHandler {
	switch op {
	case vm.CALL:
		return handleCall
	case vm.CREATE, vm.CREATE2:
		return handleCreate
	case vm.STATICCALL, vm.DELEGATECALL, vm.CALLCODE:
		return handleOtherCalls
	}

	return nil
}

func handleCreate(hash string, receiptStatus uint64, index int, log vm.StructLogRes, node *node) *TransferTx {
	transfer := &TransferTx{}

	transfer.Type = strings.ToLower(log.Op)

	transfer.Hash = hash
	//transfer.Amount = hexutil.Encode(common.FromHex(log.Stack[len(log.Stack)-2]))
	transfer.From = log.From
	//transfer.To = common.HexToAddress(log.Stack[len(log.Stack)-1]).String()
	transfer.Depth = log.Depth

	if !node.success {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "node error", node.errMsg)

		return transfer
	}

	//no next log, then can't confirm call success, trade as failed
	if index == len(node.logs)-1 {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "no next op", node.errMsg)

		return transfer
	}

	nextLog := node.logs[index+1]
	// evm call return 0, means failed
	if big.NewInt(0).SetBytes(common.FromHex(nextLog.Stack[len(nextLog.Stack)-1])).Sign() == 0 {
		node.success = false
		node.errMsg = log.ErrMsg

		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s:%s", "next op zero", nextLog.Stack[len(nextLog.Stack)-1], node.errMsg)

		return transfer
	}

	transfer.Amount = hexutil.Encode(common.FromHex(node.logs[index].Stack[len(node.logs[index].Stack)-1]))
	transfer.To = common.HexToAddress(nextLog.Stack[len(nextLog.Stack)-1]).String()

	if receiptStatus == types.ReceiptStatusFailed {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = "failed tx"

		return transfer
	}

	transfer.Status = TransferStatusSuccess
	node.success = true

	return transfer
}

func handleCall(hash string, receiptStatus uint64, index int, log vm.StructLogRes, node *node) *TransferTx {
	if len(log.Stack) <= 3 ||
		big.NewInt(0).SetBytes(common.FromHex(log.Stack[len(log.Stack)-3])).Cmp(big.NewInt(0)) == 0 {
		return nil
	}

	transfer := &TransferTx{}
	transfer.Type = strings.ToLower(log.Op)
	transfer.Hash = hash
	transfer.Amount = hexutil.Encode(common.FromHex(log.Stack[len(log.Stack)-3]))
	transfer.From = log.From
	transfer.To = common.HexToAddress(log.Stack[len(log.Stack)-2]).String()
	transfer.Depth = log.Depth

	if !node.success {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "node error", node.errMsg)

		return transfer
	}

	//no next log, then can't confirm call success, trade as failed
	if index == len(node.logs)-1 {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "no next op", node.errMsg)

		return transfer
	}

	nextLog := node.logs[index+1]
	// evm call return 0, means failed
	if big.NewInt(0).SetBytes(common.FromHex(nextLog.Stack[len(nextLog.Stack)-1])).Cmp(big.NewInt(1)) != 0 {
		node.success = false
		node.errMsg = log.ErrMsg

		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s:%s", "next op zero", nextLog.Stack[len(nextLog.Stack)-1], node.errMsg)

		return transfer
	}

	if receiptStatus == types.ReceiptStatusFailed {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = "failed tx"

		return transfer
	}

	node.success = true
	transfer.Status = TransferStatusSuccess

	return transfer
}

func handleOtherCalls(hash string, receiptStatus uint64, index int, log vm.StructLogRes, node *node) *TransferTx {
	transfer := &TransferTx{}
	transfer.Type = strings.ToLower(log.Op)
	transfer.Hash = hash

	transfer.From = log.From
	transfer.Depth = log.Depth

	if !node.success {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "node error", node.errMsg)

		return transfer
	}

	//no next log, then can't confirm call success, trade as failed
	if index == len(node.logs)-1 {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s", "no next op", node.errMsg)

		return transfer
	}

	nextLog := node.logs[index+1]
	// evm call return 0, means failed
	if big.NewInt(0).SetBytes(common.FromHex(nextLog.Stack[len(nextLog.Stack)-1])).Cmp(big.NewInt(1)) != 0 {
		node.success = false
		node.errMsg = log.ErrMsg

		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = fmt.Sprintf("%s:%s:%s", "next op zero", nextLog.Stack[len(nextLog.Stack)-1], node.errMsg)

		return transfer
	}

	if receiptStatus == types.ReceiptStatusFailed {
		transfer.Status = TransferStatusFailed
		transfer.ErrMsg = "failed tx"

		return transfer
	}

	node.success = true
	transfer.Status = TransferStatusSuccess

	return transfer

}
