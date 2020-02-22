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

package blockchain

import "fmt"
import "bytes"
import "sort"
import "runtime/debug"

import "github.com/deroproject/derosuite/address"
import "github.com/deroproject/derosuite/storage"
import "github.com/deroproject/derosuite/dvm"
import "github.com/deroproject/derosuite/crypto"
import "github.com/deroproject/derosuite/block"
import "github.com/deroproject/derosuite/transaction"

import "github.com/vmihailenco/msgpack"

// this will process the SC transaction
// the tx should only be processed , if it has been processed

func (chain *Blockchain) Process_SC(dbtx storage.DBTX, bl *block.Block, tx *transaction.Transaction, hard_fork_version_current int64) {

         defer func() {
		// safety so if anything wrong happens, verification fails
		if r := recover(); r != nil {
			logger.Warnf("Recovered while rewinding chain, Stack trace below block_hash ")
			logger.Warnf("Stack trace  \n%s", debug.Stack())
		}
	}()
        
	var err error
	if !tx.Verify_SC_Signature() { // if tx is not SC TX, or Signature could not be verified skip it
		return
	}

	bl_hash := bl.GetHash()
	tx_hash := tx.GetHash()

	addri, _ := tx.Extra_map[transaction.TX_EXTRA_ADDRESS].(address.Address)

	/*
	   addr,result := addri.(address.Address)
	   if !result {
	       return
	   }*/

	fmt.Printf("Processing TX SC  data %d \n", len(tx.Extra_map[transaction.TX_EXTRA_SCDATA].([]byte)))

	if len(tx.Extra_map[transaction.TX_EXTRA_SCDATA].([]byte)) < 3 {
		fmt.Printf("Cannot process SCTX, since data is  less than 3 bytes\n")
		return
	}

	// lets decode SC transaction from msgpack
	var sc_tx transaction.SC_Transaction
	err = msgpack.Unmarshal(tx.Extra_map[transaction.TX_EXTRA_SCDATA].([]byte), &sc_tx)
	if err != nil {
		fmt.Printf("SC msgpack unmarshal err %s\n", err)
		return
	}

	fmt.Printf("sctx %+v\n", sc_tx)

	// dicard any value provided with the tx and calculate from ring signature
	sc_tx.Value = 0 // make sure value is zero

	// check if any DERO value  is attached, if yes, attach it
	for i := 0; i < len(tx.Vout); i++ {
		var zero crypto.Key
		if tx.Vout[i].Amount != 0 && tx.Vout[i].Target.(transaction.Txout_to_key).Key == zero { // allow SC amounts to be open
			// amount has already been verified as genuine by ringct

			sc_tx.Value = tx.Vout[i].Amount
			break

		}
	}

	tx_store := dvm.Initialize_TX_store()

	// used as value loader from disk
	// this function is used to load any data required by the SC

	diskloader := func(key dvm.DataKey, found *uint64) (result dvm.Variable) {
		var exists bool
		keyhash := crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(key)))
		result, exists = chain.LoadSCValue(dbtx, key.SCID, keyhash)

		fmt.Printf("Loading from disk %+v  result %+v found status %+v \n", key, result, exists)
		if exists {

			*found = uint64(1)
		}
		return
	}

	tx_store.DiskLoader = diskloader // hook up loading from chain

	entrypoint := ""
	var scid crypto.Key
	var sc_parsed dvm.SmartContract
	execute := false

	// if we are installing SMART CONTRACT, do it and call initialize
	if len(sc_tx.SC) > 0 {

		pos := ""
		sc_parsed, pos, err = dvm.ParseSmartContract(string(sc_tx.SC))

		if err != nil {
			fmt.Printf("error Parsing sc txid %s err %s pos %s\n", tx_hash, err, pos)
			return
		}

		fmt.Printf("SC parsed\n%+v\n", sc_parsed)

		dbtx.StoreObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, tx_hash[:], PLANET_TX_SC_BYTES, []byte(sc_tx.SC))

		serialized, err := msgpack.Marshal(sc_parsed)

		if err != nil {
			fmt.Printf("err serial SC err %s\n", err)
		}

		dbtx.StoreObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, tx_hash[:], PLANET_TX_SC_PROCESSED, serialized)

		tx_store.DiskLoader = diskloader // hook up loading from chain

		tx_store.Store(dvm.GetBalanceKey(crypto.Key(tx_hash)), dvm.Variable{Type: dvm.Uint64, Value: uint64(0)})

		entrypoint = "Initialize"
		scid = crypto.Key(tx_hash)

		if _, ok := sc_parsed.Functions[entrypoint]; ok {
			execute = true
		} else {
			fmt.Printf("stored SC  does not contain entrypoint '%s' scid %s \n", entrypoint, scid)
		}

		// store state changes
		//chain.store_changes(dbtx, crypto.Key(tx_hash),tx_store)

		// we must also initialize and give the SC 0 balance
		fmt.Printf(" SMART contract parsed \n")

	} else {
		// check if scid can be hex decoded

		// load smart contract bytes, if loading failed , dero value is lost
		sc_parsed_bytes, err := dbtx.LoadObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, sc_tx.SCID[:], PLANET_TX_SC_PROCESSED)
		if err != nil {
			fmt.Printf("No such stored SC found %s\n", sc_tx.SCID)
			return
		}

		// deserialise
		err = msgpack.Unmarshal(sc_parsed_bytes, &sc_parsed)
		if err != nil {
			fmt.Printf("stored SC (parsed) could not be deserialised scid %s err  %s\n", sc_tx.SCID, err)
			return
		}

		if sc_tx.EntryPoint != "Initialize" { // initialize cannot be triggerred again
			execute = true
			entrypoint = sc_tx.EntryPoint
		}
		scid = sc_tx.SCID

	}

	if execute {
		// if we found the SC in parsed form, check whether entrypoint is found
		function, ok := sc_parsed.Functions[entrypoint]
		if !ok {
			fmt.Printf("stored SC  does not contain entrypoint '%s' scid %s \n", entrypoint, scid)
			return
		}

		if len(sc_tx.Params) == 0 { // initialize params if not initialized earlier
			sc_tx.Params = map[string]string{}
		}
		sc_tx.Params["value"] = fmt.Sprintf("%d", sc_tx.Value) // overide value

		// we have an entrypoint, now we must setup parameters and dvm
		// all parameters are in string form to bypass translation issues in middle layers
		params := map[string]interface{}{}
		for _, p := range function.Params {
			if param_value, ok := sc_tx.Params[p.Name]; ok {
				params[p.Name] = param_value
			} else { // necessary parameter is missing, bailout
				fmt.Printf("entrypoint '%s' scid %s  parameter missing '%s' \n", entrypoint, scid, p.Name)
				return
			}
		}

		tx_store.DiskLoader = diskloader // hook up loading from chain

		// setup balance correctly
		tx_store.Balance(scid) // transfer any value to make sure its not lost
		tx_store.ReceiveInternal(scid, sc_tx.Value)

		// setup block hash, height, topoheight correctly
		state := &dvm.Shared_State{
			DERO_Received: sc_tx.Value,
			Store:         tx_store,
			Chain_inputs: &dvm.Blockchain_Input{
				BL_HEIGHT:     uint64(chain.Load_Height_for_BL_ID(dbtx, bl_hash)),
				BL_TOPOHEIGHT: uint64(chain.Load_Block_Topological_order(dbtx, bl_hash)),
				SCID:          scid,
				BLID:          crypto.Key(bl_hash),
				TXID:          crypto.Key(tx_hash),
				Signer:        addri},
		}

		result, err := dvm.RunSmartContract(&sc_parsed, entrypoint, state, params)

		fmt.Printf("result value %+v\n", result)

		if err != nil {
			fmt.Printf("entrypoint '%s' scid %s  err execution '%s' \n", entrypoint, scid, err)
		}

		if err == nil && result.Type == dvm.Uint64 && result.Value.(uint64) == 0 {
			// confirm the changes
		} else { // discard all changes
			tx_store = dvm.Initialize_TX_store()
			tx_store.DiskLoader = diskloader // hook up loading from chain
			tx_store.Balance(scid)           // transfer any value to make sure its not lost arbitrarily to the network
			tx_store.ReceiveInternal(scid, sc_tx.Value)
		}

	}
	// store state changes
	chain.store_changes(dbtx, crypto.Key(tx_hash), tx_store)

	// chain.Revert_SC(dbtx,crypto.Key(tx_hash),hard_fork_version_current)

	fmt.Printf("SC execution finished amount value %d\n", sc_tx.Value)

	fmt.Printf("SC processing finished %d\n", len(sc_tx.SC))

}

// this will revert the SC transaction changes to the DB
func (chain *Blockchain) Revert_SC(dbtx storage.DBTX, tx_hash crypto.Key, hard_fork_version_current int64) {

	changelog := chain.Load_SCChangelog(dbtx, tx_hash)
	// if we are here everything is ok, lets write the values, in the reverse order
	if len(changelog) == 0 {
		return
	}
	for i := len(changelog) - 1; i >= 0; i-- {
		change := changelog[i]
		fmt.Printf("Reverting todb %s %s %s\n", change.SCID, change.Key, change.Previous)
		chain.StoreSCValue(dbtx, change.SCID, change.Key, change.Previous)
	}
}

func (chain *Blockchain) Load_SCChangelog(dbtx storage.DBTX, tx_hash crypto.Key) (changes []TX_SC_storage) {

	object_data, err := dbtx.LoadObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, tx_hash[:], PLANET_TX_SC_CHANGELOG)

	if err != nil { // change log is not present
		return
	}

	if len(object_data) == 0 { // change log is present but 0 bytes
		return
	}

	err = msgpack.Unmarshal(object_data, &changes)
	if err != nil {
		changes = changes[:0]
		return changes
	}

	return changes
}

// this will store the changes
// TODO: FIXME this should be integrated with POW for guarantees
func (chain *Blockchain) store_changes(dbtx storage.DBTX, tx_hash crypto.Key, changes *dvm.TX_Storage) {
	var bulk_changes []TX_SC_storage
	for _, atom := range changes.Atoms {
		/*if atom.Prev_Value == atom.Value {
		    continue
		}*/
		keyhash := crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(atom.Key)))
		prev := dvm.Serialize_Variable(atom.Prev_Value)
		current := dvm.Serialize_Variable(atom.Value)

		bulk_changes = append(bulk_changes, TX_SC_storage{SCID: atom.Key.SCID, Key: keyhash, Previous: prev, Current: current})

	}

	//  map has different order for different iterations,so we sort them in some fixed order,
	var keyarray []crypto.Key
	for k, _ := range changes.Transfers {
		var tmp crypto.Key
		copy(tmp[:], k[:])
		keyarray = append(keyarray, tmp)
	}

	sort.Slice(keyarray, func(i, j int) bool { return bytes.Compare(keyarray[i][:], keyarray[j][:]) == -1 })

	// let calculate any balance updates
	for i := range keyarray {
		k := keyarray[i]
		v := changes.Transfers[k]
		keyhash := crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(dvm.GetBalanceKey(k))))
		prev := dvm.Serialize_Variable(dvm.Variable{Type: dvm.Uint64, Value: v.BalanceAtStart})
		current := dvm.Serialize_Variable(dvm.Variable{Type: dvm.Uint64, Value: changes.Balance(k)})

		bulk_changes = append(bulk_changes, TX_SC_storage{SCID: k, Key: keyhash, Previous: prev, Current: current})

		// this lines processes external outputs
		bulk_changes[0].TransferE = append(bulk_changes[0].TransferE, v.TransferE...)
	}

	// if we are here everything is ok, lets write the values
	for _, change := range bulk_changes {
		fmt.Printf("storing todb %s %s %s\n", change.SCID, change.Key, change.Current)
		chain.StoreSCValue(dbtx, change.SCID, change.Key, change.Current)
	}

	// store the change log itself
	serialized_change_log, _ := msgpack.Marshal(bulk_changes)
	dbtx.StoreObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, tx_hash[:], PLANET_TX_SC_CHANGELOG, serialized_change_log)

}

// this will load the value from the chain
func (chain *Blockchain) LoadSCValue(dbtx storage.DBTX, scid crypto.Key, keyhash crypto.Key) (v dvm.Variable, found bool) {

	var err error
	if dbtx == nil {
		dbtx, err = chain.store.BeginTX(false)
		if err != nil {
			logger.Warnf("Could NOT load SC Value. Error opening writable TX, err %s", err)
			return
		}

		defer dbtx.Rollback()

	}

	fmt.Printf("loading fromdb %s %s \n", scid, keyhash)
	object_data, err := dbtx.LoadObject(SMARTCONTRACT_UNIVERSE, SMARTCONTRACT_UNIVERSE, scid[:], keyhash[:])

	if err != nil {
		return v, false
	}

	if len(object_data) == 0 {
		return v, false
	}

	v = dvm.Deserialize_Variable(object_data).(dvm.Variable)

	return v, true
}

// reads a value from SC, always read balance
func (chain *Blockchain) ReadSC(dbtx storage.DBTX, scid crypto.Key) (sc dvm.SmartContract, found bool) {

	var err error
	if dbtx == nil {
		dbtx, err = chain.store.BeginTX(false)
		if err != nil {
			logger.Warnf("Could NOT load SC Value. Error opening writable TX, err %s", err)
			return
		}

		defer dbtx.Rollback()

	}

	sc_parsed_bytes, err := dbtx.LoadObject(BLOCKCHAIN_UNIVERSE, GALAXY_TRANSACTION, scid[:], PLANET_TX_SC_PROCESSED)
	if err != nil {
		return
	}

	// deserialise
	err = msgpack.Unmarshal(sc_parsed_bytes, &sc)
	if err != nil {
		return
	}

	found = true
	return
}

// reads a value from SC, always read balance
func (chain *Blockchain) ReadSCValue(dbtx storage.DBTX, scid crypto.Key, key interface{}) (balance uint64, value interface{}) {

	keyhash := crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(dvm.GetBalanceKey(scid))))

	balance_var, found := chain.LoadSCValue(dbtx, scid, keyhash)

	if !found { // no balance = no SC
		return
	}
	if balance_var.Type == dvm.Uint64 {
		balance = balance_var.Value.(uint64)
	}

	if key == nil {
		return
	}
	switch k := key.(type) {
	case uint64:
		keyhash = crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(
			dvm.DataKey{Key: dvm.Variable{Type: dvm.Uint64, Value: k}})))
	case string:
		keyhash = crypto.Key(crypto.Keccak256(dvm.Serialize_DataKey(
			dvm.DataKey{Key: dvm.Variable{Type: dvm.String, Value: k}})))
	default:
		return
	}

	value_var, _ := chain.LoadSCValue(dbtx, scid, keyhash)
	fmt.Printf("read value %+v", value_var)
	if value_var.Type != dvm.Invalid {
		value = value_var.Value
	}
	return
}

// store the value in the chain
func (chain *Blockchain) StoreSCValue(dbtx storage.DBTX, scid crypto.Key, keyhash crypto.Key, value []byte) {
	dbtx.StoreObject(SMARTCONTRACT_UNIVERSE, SMARTCONTRACT_UNIVERSE, scid[:], keyhash[:], value[:])
	return
}

// all these are stored in a tx container
type TX_SC_storage struct {
	SCID     crypto.Key `msgpack:"S,omitempty"`
	Key      crypto.Key `msgpack:"K,omitempty"`
	Previous []byte     `msgpack:"P,omitempty"` // previous value
	Current  []byte     `msgpack:"-"`           // current value // this need not be stored

	TransferE []dvm.TransferExternal `msgpack:"T,omitempty"`
}

// get public and ephermal key to pay to address
// TODO we can also payto blobs which even hide address
// both have issues, this requires address to be public
// blobs require blobs as paramters
func GetEphermalKey(txid crypto.Key, index_within_tx uint64, address_input string) (tx_public_key, ehphermal_public_key crypto.Key) {

	var tx_secret_key crypto.Key

	copy(tx_secret_key[:], txid[:])
	crypto.ScReduce32(&tx_secret_key)
	tx_public_key = *tx_secret_key.PublicKey()

	addr, err := address.NewAddress(address_input)

	// test case to pay to devs, if user passed invalid address
	// TODO: expose a function to user to validate an address
	if err != nil {
		//panic(fmt.Sprintf("Invalid input address while generating blob %s err %s", address_input, err ))
		addr, _ = address.NewAddress("dERoYnipRygd8RZxcpbgcMRvjHVRy2Tr2Jzu8rFWQQdfbPGp8SJYN2QLeKSzzJsbqh3CyRA7ebGvVT3ETWTV8FGh8dkb2NaWVt")
	}

	derivation := crypto.KeyDerivation(&addr.ViewKey, &tx_secret_key) // keyderivation using wallet address view key

	// this becomes the key within Vout
	ehphermal_public_key = derivation.KeyDerivation_To_PublicKey(index_within_tx, addr.SpendKey)
	return
}
