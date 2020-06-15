package huobi

import (
	"strconv"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb/leveldb"
	"github.com/ethereum/go-ethereum/rlp"
	level "github.com/syndtr/goleveldb/leveldb"
)

func OpenDB() (traceDB *leveldb.Database, err error) {
	traceDB, err = leveldb.New("innerTx", 128, 1024, "innerTx")
	if err != nil {
		return
	}

	err = upgrade(traceDB)

	return traceDB, err
}

var DBVersion = []byte("version_code")

type upgradeFunc func(db *leveldb.Database, oldVersion int) (int, error)

var funcs = []upgradeFunc{
	upgradeAddStatusAndType,
}

func upgrade(db *leveldb.Database) (err error) {

	for _, f := range funcs {
		var versionByte []byte
		versionByte, err = db.Get(DBVersion)
		if err != nil && err != level.ErrNotFound {
			return
		} else if err == level.ErrNotFound {
			versionByte = []byte("0")
		}

		var version int
		version, err = strconv.Atoi(string(versionByte))
		if err != nil {
			return
		}

		var newVersion int
		newVersion, err = f(db, version)
		if err != nil {
			return
		}

		if newVersion > 0 {
			db.Put(DBVersion, []byte(strconv.Itoa(newVersion)))
		}
	}

	return
}

func upgradeAddStatusAndType(traceDB *leveldb.Database, oldVersion int) (newVersion int, err error) {
	//already handled
	if oldVersion >= 1 {
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
	}

	type LogRes struct {
		Hash     string
		TxToAddr string
		Logs     []StructLogRes
	}

	itr := traceDB.NewIterator(nil, nil)
	for itr.Next() {
		key := itr.Key()
		value := itr.Value()

		txTraces := make([]*LogRes, 0)
		err = rlp.DecodeBytes(value, &txTraces)
		if err != nil {
			return
		}

		newTxTrace := make([]*vm.LogRes, 0)
		for _, old := range txTraces {
			newT := &vm.LogRes{}
			newT.Hash = old.Hash
			newT.TxToAddr = old.TxToAddr
			newT.Logs = make([]vm.StructLogRes, 0)
			for _, oldLog := range old.Logs {
				newT.Logs = append(newT.Logs, vm.StructLogRes{oldLog.From, oldLog.Pc, oldLog.Op, oldLog.Gas, oldLog.GasCost, oldLog.Depth, oldLog.Stack, ""})
			}
		}

		var newB []byte
		newB, err = rlp.EncodeToBytes(newTxTrace)
		if err != nil {
			return
		}

		err = traceDB.Put(key, newB)
		if err != nil {
			return
		}

	}

	newVersion = 1
	return
}
