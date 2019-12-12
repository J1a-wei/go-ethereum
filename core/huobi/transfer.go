package huobi

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/rlp"
)

type TransferTx struct {
	Type   string `json:"type"`
	Status string `json:"status"`
	ErrMsg string `json:"errMsg,omitempty"`
	Hash   string `json:"hash"`
	From   string `json:"from"`
	To     string `json:"to"`
	Depth  uint   `json:"depth"`
	Amount string `json:"value"`
}

type BlockTransfer struct {
	Hash      string        `json:"hash"`
	Transfers []*TransferTx `json:"transfers"`
}

func SaveTransfers(db *leveldb.Database, hash common.Hash, txLogs []*vm.LogRes) (err error) {
	result, err := rlp.EncodeToBytes(txLogs)
	if err != nil {
		return
	}

	err = db.Put(hash.Bytes(), result)
	if err != nil {
		return
	}

	return
}

const (
	TransferStatusSuccess = "success"
	TransferStatusFailed  = "failed"
)

type node struct {
	// one node for one call
	logs []vm.StructLogRes

	children []*node

	parent *node

	depth uint

	success bool

	errMsg string
}

func GetTransfersByBlockHash(db *leveldb.Database, hash common.Hash) (result BlockTransfer, err error) {
	bytesRes, err := db.Get(hash.Bytes())
	if err != nil {
		return
	}

	txTraces := make([]*vm.LogRes, 0)
	err = rlp.DecodeBytes(bytesRes, &txTraces)
	if err != nil {
		return
	}

	result.Transfers = make([]*TransferTx, 0)
	result.Hash = hash.String()

	for _, item := range txTraces {
		//each item is one transaction

		root := &node{children: make([]*node, 0), depth: 0, success: true}
		current := root

		for i, log := range item.Logs {
			if vm.CaredOps[vm.StringToOp(log.Op)] ||
				(i > 0 && item.Logs[i-1].Depth < log.Depth) { //condition 2: deeper than last  CALL(depth 1) -> REVERT(depth 2 current, has error) -> SWAP(depth 1)

				//|| // condition 1: start op(can create a new depth)
				//(i < len(item.Logs)-1 && log.Depth < item.Logs[i+1].Depth && // condition 3: will go deeper CALL(current, index = 0) --> CALL (should be handled by condition 1, if all call like ops have bee handled
				//	(i == 0 ||
				//		(!vm.CaredOps[vm.StringToOp(item.Logs[i-1].Op)]))) { //CALL --> EQUAL --> CALL(current) --> CALL
				//start a node
				tmp := &node{children: make([]*node, 0), depth: log.Depth, success: true}
				tmp.logs = append(tmp.logs, log)

				if tmp.depth > current.depth {
					tmp.parent = current
					current.children = append(current.children, tmp)
				} else {
					for tmp.depth < current.depth {
						current = current.parent
					}

					tmp.parent = current.parent
					current.parent.children = append(current.parent.children, tmp)
				}
				current = tmp
			} else { //end of a node
				for current.depth > log.Depth {
					current = current.parent
				}

				current.logs = append(current.logs, log)
			}
		}

		calcNodeStateByOp(root)

		result.Transfers = generateTransferData(item.Hash, item.ReceiptStatus, root, result.Transfers)
	}

	//if node is failed, parent node and brother node are all error.

	return
}

//calc if node op is failed then parent node and brother nodes are failed
func calcNodeStateByOp(current *node) {
	if !current.success {
		return
	}

	for _, log := range current.logs {
		if log.ErrMsg == "" {
			continue
		}

		if current.parent != nil {
			current.parent.success = false
			current.parent.errMsg = log.ErrMsg

			for _, child := range current.parent.children {
				child.errMsg = log.ErrMsg
				child.success = false
			}
		}

		break
	}

	if !current.success {
		return
	}

	for _, child := range current.children {
		calcNodeStateByOp(child)
	}
}

func generateTransferData(hash string, receiptStatus uint64, node *node, transfers []*TransferTx) (result []*TransferTx) {
	// node.success = node.parent.success & node.success
	if node.parent != nil && node.success {
		node.success = node.parent.success
		node.errMsg = node.parent.errMsg
	}

	for i, log := range node.logs {
		handler := getLogHandler(vm.StringToOp(log.Op))
		var item *TransferTx

		if handler != nil {
			item = handler(hash, receiptStatus, i, log, node)
		}

		if item != nil {
			transfers = append(transfers, item)
		}
	}

	for _, child := range node.children {
		transfers = generateTransferData(hash, receiptStatus, child, transfers)
	}

	result = transfers

	return
}

func GetTransferLogs(db *leveldb.Database, hash common.Hash) (txTraces []*vm.LogRes, err error) {
	bytesRes, err := db.Get(hash.Bytes())
	if err != nil {
		return
	}

	txTraces = make([]*vm.LogRes, 0)
	err = rlp.DecodeBytes(bytesRes, &txTraces)
	if err != nil {
		return
	}

	return

}
