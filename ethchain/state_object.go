package ethchain

import (
	"github.com/ethereum/eth-go/ethutil"
	"math/big"
)

type StateObject struct {
	// Address of the object
	address []byte
	// Shared attributes
	Amount *big.Int
	Nonce  uint64
	// Contract related attributes
	state      *State
	script     []byte
	initScript []byte
}

func NewContract(address []byte, Amount *big.Int, root []byte) *StateObject {
	contract := &StateObject{address: address, Amount: Amount, Nonce: 0}
	contract.state = NewState(ethutil.NewTrie(ethutil.Config.Db, string(root)))

	return contract
}

// Returns a newly created account
func NewAccount(address []byte, amount *big.Int) *StateObject {
	account := &StateObject{address: address, Amount: amount, Nonce: 0}

	return account
}

func NewStateObjectFromBytes(address, data []byte) *StateObject {
	object := &StateObject{address: address}
	object.RlpDecode(data)

	return object
}

func (c *StateObject) Addr(addr []byte) *ethutil.Value {
	return ethutil.NewValueFromBytes([]byte(c.state.trie.Get(string(addr))))
}

func (c *StateObject) SetAddr(addr []byte, value interface{}) {
	c.state.trie.Update(string(addr), string(ethutil.NewValue(value).Encode()))
}

func (c *StateObject) State() *State {
	return c.state
}

func (c *StateObject) GetMem(num *big.Int) *ethutil.Value {
	nb := ethutil.BigToBytes(num, 256)

	return c.Addr(nb)
}

func (c *StateObject) GetInstr(pc *big.Int) *ethutil.Value {
	if int64(len(c.script)-1) < pc.Int64() {
		return ethutil.NewValue(0)
	}

	return ethutil.NewValueFromBytes([]byte{c.script[pc.Int64()]})
}

func (c *StateObject) SetMem(num *big.Int, val *ethutil.Value) {
	addr := ethutil.BigToBytes(num, 256)
	c.state.trie.Update(string(addr), string(val.Encode()))
}

// Return the gas back to the origin. Used by the Virtual machine or Closures
func (c *StateObject) ReturnGas(val *big.Int, state *State) {
	c.AddAmount(val)
}

func (c *StateObject) AddAmount(amount *big.Int) {
	c.Amount.Add(c.Amount, amount)
}

func (c *StateObject) SubAmount(amount *big.Int) {
	c.Amount.Sub(c.Amount, amount)
}

func (c *StateObject) Address() []byte {
	return c.address
}

func (c *StateObject) Script() []byte {
	return c.script
}

func (c *StateObject) Init() []byte {
	return c.initScript
}

func (c *StateObject) RlpEncode() []byte {
	var root interface{}
	if c.state != nil {
		root = c.state.trie.Root
	} else {
		root = nil
	}
	return ethutil.Encode([]interface{}{c.Amount, c.Nonce, root, c.script})
}

func (c *StateObject) RlpDecode(data []byte) {
	decoder := ethutil.NewValueFromBytes(data)

	c.Amount = decoder.Get(0).BigInt()
	c.Nonce = decoder.Get(1).Uint()
	c.state = NewState(ethutil.NewTrie(ethutil.Config.Db, decoder.Get(2).Interface()))
	c.script = decoder.Get(3).Bytes()
}

func MakeContract(tx *Transaction, state *State) *StateObject {
	// Create contract if there's no recipient
	if tx.IsContract() {
		// FIXME
		addr := tx.Hash()[12:]

		value := tx.Value
		contract := NewContract(addr, value, []byte(""))
		state.UpdateStateObject(contract)

		contract.script = tx.Data
		contract.initScript = tx.Init

		state.UpdateStateObject(contract)

		return contract
	}

	return nil
}

// The cached state and state object cache are helpers which will give you somewhat
// control over the nonce. When creating new transactions you're interested in the 'next'
// nonce rather than the current nonce. This to avoid creating invalid-nonce transactions.
type StateObjectCache struct {
	cachedObjects map[string]*CachedStateObject
}

func NewStateObjectCache() *StateObjectCache {
	return &StateObjectCache{cachedObjects: make(map[string]*CachedStateObject)}
}

func (s *StateObjectCache) Add(addr []byte, object *StateObject) *CachedStateObject {
	state := &CachedStateObject{Nonce: object.Nonce, Object: object}
	s.cachedObjects[string(addr)] = state

	return state
}

func (s *StateObjectCache) Get(addr []byte) *CachedStateObject {
	return s.cachedObjects[string(addr)]
}

type CachedStateObject struct {
	Nonce  uint64
	Object *StateObject
}