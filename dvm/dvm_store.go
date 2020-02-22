// Copyright 2017-2018 DERO Project. All rights reserved.
// Use of this source code in any form is governed by RESEARCH license.
// license can be found in the LICENSE file.
// GPG: 0F39 E425 8C65 3947 702A  8234 08B2 0360 A03A 9DE8
//
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
// MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL
// THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
// STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF
// THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package dvm

import "fmt"
import "encoding/binary"
import "github.com/deroproject/derosuite/crypto"

const DVAL = "DERO_BALANCE" // DERO Values are stored in this variable
const CHANGELOG = "CHANGELOG"
const GENERIC byte = 'G' // appended to contruct generic values

// this package exports an interface which is used by blockchain to persist/query data

type DataKey struct {
	SCID    crypto.Key // tx which created the the contract or contract ID
	Key     Variable
	Special bool // whether the value is generic or special , special is used to store DERO value
}

type DataAtom struct {
	Key DataKey

	Prev_Value Variable // previous Value if any
	Value      Variable // current value if any
}

type TransferInternal struct {
	Received []uint64
	Sent     []uint64
}

// any external tranfers
type TransferExternal struct {
	Address string `msgpack:"A,omitempty" json:"A,omitempty"` //  transfer to this blob
	Amount  uint64 `msgpack:"V,omitempty" json:"V,omitempty"` // Amount in Atomic units
}

type SC_Transfers struct {
	BalanceAtStart uint64             // value at start
	TransferI      TransferInternal   // all internal transfers
	TransferE      []TransferExternal // all external transfers
}

// all SC load and store operations will go though this
type TX_Storage struct {
	DiskLoader func(DataKey, *uint64) Variable
	Atoms      []DataAtom           // all modification operations have to played/reverse in this order
	Keys       map[DataKey]Variable // this keeps the in-transit DB updates, just in case we have to discard instantly

	Transfers map[crypto.Key]SC_Transfers // all transfers ( internal/external )
}

var DVM_STORAGE_BACKEND DVM_Storage_Loader // this variable can be hijacked at runtime to offer different stores such as RAM/file/DB etc

type DVM_Storage_Loader interface {
	Load(DataKey, *uint64) Variable
	Store(DataKey, Variable)
}

// initialize tx store
func Initialize_TX_store() (tx_store *TX_Storage) {
	tx_store = &TX_Storage{Keys: map[DataKey]Variable{}, Transfers: map[crypto.Key]SC_Transfers{}}
	return
}

// this will load the variable, and if the key is found
func (tx_store *TX_Storage) Load(dkey DataKey, found_value *uint64) (value Variable) {

	fmt.Printf("Loading %+v   \n", dkey)

	*found_value = 0
	// if it was modified in current TX, use it
	if result, ok := tx_store.Keys[dkey]; ok {
		*found_value = 1
		return result
	}

	/*if DVM_STORAGE_BACKEND == nil {
	   panic("DVM_STORAGE_BACKEND is not ready")
	  }

	  // else look in actual store
	  //value = DVM_STORAGE_BACKEND.Load(dkey, found_value)
	*/

	if tx_store.DiskLoader == nil {
		panic("DVM_STORAGE_BACKEND is not ready")
	}

	value = tx_store.DiskLoader(dkey, found_value)

	return
}

// store variable
func (tx_store *TX_Storage) Store(dkey DataKey, v Variable) {

	fmt.Printf("Storing request %+v   : %+v\n", dkey, v)

	var found uint64
	old_value := tx_store.Load(dkey, &found)

	var atom DataAtom
	atom.Key = dkey
	atom.Value = v
	if found != 0 {
		atom.Prev_Value = old_value
	} else {
		atom.Prev_Value = Variable{}
	}

	tx_store.Keys[atom.Key] = atom.Value
	tx_store.Atoms = append(tx_store.Atoms, atom)

}

// store variable
func (tx_store *TX_Storage) SendExternal(sender_scid crypto.Key, addr_str string, amount uint64) {

	fmt.Printf("Transfer to  external address   : %+v\n", addr_str)

	tx_store.Balance(sender_scid) // load from disk if required
	transfer := tx_store.Transfers[sender_scid]
	transfer.TransferE = append(transfer.TransferE, TransferExternal{Address: addr_str, Amount: amount})
	tx_store.Transfers[sender_scid] = transfer
	tx_store.Balance(sender_scid) //  recalculate balance panic if any issues

}

// if TXID is not already loaded, load it
func (tx_store *TX_Storage) ReceiveInternal(scid crypto.Key, amount uint64) {

	tx_store.Balance(scid) // load from disk if required
	transfer := tx_store.Transfers[scid]
	transfer.TransferI.Received = append(transfer.TransferI.Received, amount)
	tx_store.Transfers[scid] = transfer
	tx_store.Balance(scid) //  recalculate balance panic if any issues
}

func (tx_store *TX_Storage) SendInternal(sender_scid crypto.Key, receiver_scid crypto.Key, amount uint64) {

	//sender side
	{
		tx_store.Balance(sender_scid) // load from disk if required
		transfer := tx_store.Transfers[sender_scid]
		transfer.TransferI.Sent = append(transfer.TransferI.Sent, amount)
		tx_store.Transfers[sender_scid] = transfer
		tx_store.Balance(sender_scid) //  recalculate balance panic if any issues
	}

	{
		tx_store.Balance(receiver_scid) // load from disk if required
		transfer := tx_store.Transfers[receiver_scid]
		transfer.TransferI.Received = append(transfer.TransferI.Received, amount)
		tx_store.Transfers[receiver_scid] = transfer
		tx_store.Balance(receiver_scid) //  recalculate balance panic if any issues
	}

}

func GetBalanceKey(scid crypto.Key) (x DataKey) {
	x.SCID = scid
	x.Special = true
	x.Key = Variable{Type: String, Value: DVAL}
	return x
}

/*
func GetNormalKey(scid crypto.Key,  v Variable) (x DataKey) {
    x.SCID = scid
    x.Key = Variable {Type:v.Type, Value: v.Value}
    return x
}
*/

// this will give the balance, will load the balance from disk
func (tx_store *TX_Storage) Balance(scid crypto.Key) uint64 {

	if _, ok := tx_store.Transfers[scid]; !ok {

		var transfer SC_Transfers
		found_value := uint64(0)
		value := tx_store.Load(GetBalanceKey(scid), &found_value)

		if found_value == 0 {
			panic(fmt.Sprintf("SCID %s is not loaded", scid)) // we must load  it from disk
		}

		if value.Type != Uint64 {
			panic(fmt.Sprintf("SCID %s balance is not uint64, HOW ??", scid)) // we must load  it from disk
		}

		transfer.BalanceAtStart = value.Value.(uint64)
		tx_store.Transfers[scid] = transfer
	}

	transfers := tx_store.Transfers[scid]
	balance := transfers.BalanceAtStart

	// replay all receives/sends

	//  handle all internal receives
	for _, amt_received := range transfers.TransferI.Received {
		c := balance + amt_received

		if c >= balance {
			balance = c
		} else {
			panic("uint64 overflow wraparound attack")
		}
	}

	// handle all internal sends
	for _, amt_sent := range transfers.TransferI.Sent {
		if amt_sent >= balance {
			panic("uint64 underflow wraparound attack")
		}
		balance = balance - amt_sent
	}

	// handle all external sends
	for _, trans := range transfers.TransferE {
		if trans.Amount >= balance {
			panic("uint64 underflow wraparound attack")
		}
		balance = balance - trans.Amount
	}

	return balance

}

// whether the scid has enough balance
func (tx_store *TX_Storage) HasBalance(scid crypto.Key, amount uint64) {

}

// why should we not hash the return value to return a hash value
// using entire key could be useful, if DB can somehow link between  them in the form of buckets and all
func Serialize_DataKey(dkey DataKey) (ser []byte) {
	//ser=append(ser,dkey.SCID[:]...)
	if !dkey.Special { // special will not have generic marker, this protects from number of attacks
		ser = append(ser, GENERIC) // add generic marker
	}
	ser = append(ser, Serialize_Variable(dkey.Key)...) // add object type

	return ser
}

// these are used by lowest layers
func Serialize_Variable(v Variable) (ser []byte) {
	ser = append(ser, byte(v.Type)) // add object type
	switch v.Type {
	case Invalid:
		return []byte{} // return empty array //panic("Invalid cannot be serialized")
	case Uint64:
		num := itob(v.Value.(uint64)) // uint64 data type
		ser = append(ser, num...)
	case Address:
		ser = append(ser, ([]byte(v.Value.(string)))...) // string
	case Blob:
		panic("blob not implemented") // an encrypted blob, used to add data to blockchain without knowing address
	case String:
		ser = append(ser, ([]byte(v.Value.(string)))...) // string

	default:
		panic("unknown variable type not implemented")

	}
	return ser
}

func Deserialize_Variable(v []byte) interface{} {

	if len(v) < 1 || Vtype(v[0]) == Invalid {
		return nil
	}

	switch Vtype(v[0]) {
	case Invalid:
		panic("Invalid cannot be deserialized")
	case Uint64:
		if len(v) != 9 {
			panic("corruption in DB")
		}
		return Variable{Type: Uint64, Value: binary.BigEndian.Uint64(v[1:])} // uint64 data type

	case String:
		fallthrough // string
	case Address:
		return Variable{Type: String, Value: string(v[1:])} //  we should verify whether string is an address
	case Blob:
		panic("blob not implemented") // an encrypted blob, used to add data to blockchain without knowing address

	default:
		panic("unknown variable type not implemented")

	}
	// we can never reach here
	panic("we can never reach here")
	return nil
}

// itob returns an 8-byte big endian representation of v.
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
