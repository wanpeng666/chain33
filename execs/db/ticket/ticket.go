package ticket

//database opeartion for execs ticket
import (
	"fmt"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var tlog = log.New("module", "ticket.db")
var genesisKey = []byte("mavl-acc-genesis")
var addrSeed = []byte("address seed bytes for public key")

type Ticket struct {
	types.Ticket
}

func NewTicket(id, minerAddress, returnWallet string, blocktime int64) *Ticket {
	t := &Ticket{}
	t.TicketId = id
	t.MinerAddress = minerAddress
	t.ReturnAddress = returnWallet
	t.CreateTime = blocktime
	t.Status = 1
	t.IsGenesis = true
	return t
}

func (t *Ticket) GetReceiptLog() *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogNewTicket
	r := &types.ReceiptNewTicket{}
	r.TicketId = t.TicketId
	log.Log = types.Encode(r)
	return log
}

func (t *Ticket) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&t.Ticket)
	kvset = append(kvset, &types.KeyValue{TicketKey(t.TicketId), value})
	return kvset
}

func (t *Ticket) Save(db dbm.KVDB) {
	set := t.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

//address to save key
func TicketKey(id string) (key []byte) {
	key = append(key, []byte("mavl-ticket-")...)
	key = append(key, []byte(id)...)
	return key
}

func GenesisInit(db dbm.KVDB, hash []byte, execaddr string, genesis *types.TicketGenesis, blocktime int64) (*types.Receipt, error) {
	prefix := common.ToHex(hash)
	prefix = genesis.MinerAddress + ":" + prefix + ":"
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	for i := 0; i < int(genesis.Count); i++ {
		id := prefix + fmt.Sprintf("%010d", i)
		t := NewTicket(id, genesis.MinerAddress, genesis.ReturnAddress, blocktime)

		//冻结子账户资金
		receipt, err := account.ExecFrozen(db, genesis.ReturnAddress, execaddr, 1000*types.Coin)
		if err != nil {
			tlog.Error("GenesisInit.Frozen", "addr", genesis.ReturnAddress, "execaddr", execaddr)
			panic(err)
		}
		t.Save(db)
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		logs = append(logs, t.GetReceiptLog())
		kv = append(kv, t.GetKVSet()...)
	}
	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}