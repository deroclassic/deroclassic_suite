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

package transaction

import "fmt"
import "bytes"

//import "runtime/debug"

import "encoding/binary"

import "github.com/romana/rlog"

import "github.com/deroclassic/deroclassic_suite/crypto"
import "github.com/deroclassic/deroclassic_suite/address"

// refer https://cryptonote.org/cns/cns005.txt to understand slightly more ( it DOES NOT cover everything)
// much of these constants are understood from tx_extra.h and cryptonote_format_utils.cpp
// TODO pending test case
type EXTRA_TAG byte

const TX_EXTRA_PADDING EXTRA_TAG = 0 // followed by 1 byte of size, and then upto 255 bytes of padding
const TX_PUBLIC_KEY EXTRA_TAG = 1    // follwed by 32 bytes of tx public key
const TX_EXTRA_NONCE EXTRA_TAG = 2   // followed by 1 byte of size, and then upto 255 bytes of empty nonce

//const TX_EXTRA_MSG = 4   is reserved for fututure 

const TX_EXTRA_ADDRESS = 0x10  // followed by 64 bytes of DERO address  // what about sub address
const TX_EXTRA_SIG = 0x11  // followed by 64 bytes of DERO 25519 signature  
const TX_EXTRA_SCDATA = 0x12 // smart contract to install

const TX_EXTRA_SCID = 3  // followed by 32 bytes of Smart contract TXID

const TX_EXTRA_SC_CALL = 7 // smart contract to call 



const TX_EXTRA_MSG = 4 // followed by varint bytes of of data, and upto 40960 bytes of data  


// TX_EXTRA_MERGE_MINING_TAG  we do NOT suppport merged mining at all
// TX_EXTRA_MYSTERIOUS_MINERGATE_TAG  as the name says mysterious we will not bring it

// these 2 fields have complicated parsing of extra, other the code was really simple
const TX_EXTRA_NONCE_PAYMENT_ID EXTRA_TAG = 0           // extra nonce within a non coinbase tx, can be unencrypted, is 32 bytes in size
const TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID EXTRA_TAG = 1 // this is encrypted and is 9 bytes in size


// TX_EXTRA_SC  has the following structure
//length prefixed public address
//length prefixed 64 byte signature
//length prefixed data


// the field is just named extra and contains CRITICAL information, though some is optional
// parse extra data such as
// tx public key must
// payment id optional
// encrypted payment id optional
func (tx *Transaction) Parse_Extra() (result bool) {
	var err error
	// var length uint64
	var length_int int

	/*defer func (){
		if r := recover(); r != nil {
				fmt.Printf("Recovered while parsing extra, Stack trace below block_hash %s", tx.GetHash())
				fmt.Printf("Stack trace  \n%s", debug.Stack())
				result = false
			}
	        }()*/

	buf := bytes.NewReader(tx.Extra)

	tx.Extra_map = map[EXTRA_TAG]interface{}{}
	tx.PaymentID_map = map[EXTRA_TAG]interface{}{}

	b := make([]byte, 1)
	//var r uint64
	var n int
	for i := 0; ; i++ {
		if buf.Len() == 0 {
			return true
		}
		n, err = buf.Read(b)
		//if err != nil { // we make that the buffer has atleast 1 byte to read
		//	return false
		//}

		switch EXTRA_TAG(b[0]) {
		case TX_EXTRA_PADDING: // this is followed by 1 byte length, then length bytes of padding
			n, err = buf.Read(b)
			if err != nil {
				rlog.Tracef(1, "Extra padding length could not be parsed")
				return false
			}
			length_int = int(b[0])
			padding := make([]byte, length_int, length_int)
			n, err = buf.Read(padding)
			if err != nil || n != int(length_int) {
				rlog.Tracef(1, "Extra padding could not be read  ")
				return false
			}

			// Padding is not added to the extra map

		case TX_PUBLIC_KEY: // next 32 bytes are tx public key
			var pkey crypto.Key
			n, err = buf.Read(pkey[:])
			if err != nil || n != 32 {
				rlog.Tracef(1, "Tx public key could not be parsed len=%d err=%s ", n, err)
				return false
			}
			tx.Extra_map[TX_PUBLIC_KEY] = pkey
		case TX_EXTRA_NONCE: // this is followed by 1 byte length, then length bytes of data
			n, err = buf.Read(b)
			if err != nil {
				rlog.Tracef(1, "Extra nonce length could not be parsed ")
				return false
			}

			length_int = int(b[0])

			extra_nonce := make([]byte, length_int, length_int)
			n, err = buf.Read(extra_nonce)
			if err != nil || n != int(length_int) {
				rlog.Tracef(1, "Extra Nonce could not be read ")
				return false
			}

			switch length_int {
			case 33: // unencrypted 32 byte  payment id
				if extra_nonce[0] == byte(TX_EXTRA_NONCE_PAYMENT_ID) {
					tx.PaymentID_map[TX_EXTRA_NONCE_PAYMENT_ID] = extra_nonce[1:]
				} else {
					rlog.Tracef(1, "Extra Nonce contains invalid payment id ")
					return false
				}

			case 9: // encrypted 9 byte payment id
				if extra_nonce[0] == byte(TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID) {
					tx.PaymentID_map[TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID] = extra_nonce[1:]
				} else {
					rlog.Tracef(1, "Extra Nonce contains invalid encrypted payment id ")
					return false
				}

			default: // consider it as general nonce
				// ignore anything else
			}

			tx.Extra_map[TX_EXTRA_NONCE] = extra_nonce
			
                        case TX_EXTRA_ADDRESS:
                            address_bytes := make([]byte, 64, 64) // 32 byte public spend key, 32 byte public viewkey
                            
                            n, err = buf.Read(address_bytes)
                            if err != nil || n != 64 {
				rlog.Tracef(1, "Extra Address could not be read ")
				return false
                            }
                            
                            var spendkey,viewkey crypto.Key
                            copy(spendkey[:], address_bytes[0:32])
                            copy(viewkey[:],address_bytes[32:])
                            
                            
                            tx.Extra_map[TX_EXTRA_ADDRESS] = *address.NewAddressFromKeys(spendkey,viewkey)
                        case TX_EXTRA_SIG:
                            sig_bytes := make([]byte, 64, 64) // 32 byte Sig C , 32 byte sig R
                            n, err = buf.Read(sig_bytes)
                            if err != nil || n != 64 {
				rlog.Tracef(1, "Extra Signature could not be read ")
				return false
                            }
                            sig := crypto.Signature{}
                            copy(sig.C[:],sig_bytes[0:32])
                            copy(sig.R[:],sig_bytes[32:])
                            tx.Extra_map[TX_EXTRA_SIG] = sig
                            
                        case TX_EXTRA_SCDATA:
                            var sc_data_len uint64
                            sc_data_len,err = binary.ReadUvarint(buf)
                            if err != nil || sc_data_len == 0{
                                rlog.Debugf( "Extra SC Data could not be read ")
                                return false
                            }
                            
                            if sc_data_len > 128*1024 { // stop ddos right now
                                rlog.Debugf( "Extra SC Data attempting Ddos, stopping attack ")
                                return false
                            }
                            
                            data_bytes := make([]byte, sc_data_len, sc_data_len)
                            n, err = buf.Read(data_bytes)
                            if err != nil || n != int(sc_data_len) {
				rlog.Debugf( "Extra SC Data could not be read, err %s n %d sc_data_len %d ", err, n, sc_data_len)
				return false
                            }
                            tx.Extra_map[TX_EXTRA_SCDATA] = data_bytes
                            

			// NO MORE TAGS are present

		default: // any any other unknown tag or data, fails the parsing
			rlog.Tracef(1, "Unhandled TAG %d \n", b[0])
			result = false
			return

		}
	}

	// we should not reach here
	//return true
}

// serialize an extra, this is only required while creating new transactions ( both miner and normal)
// doing this on existing transaction will cause them to fail ( due to different placement order )
func (tx *Transaction) Serialize_Extra() []byte {

	buf := bytes.NewBuffer(nil)

	// this is mandatory
	if _, ok := tx.Extra_map[TX_PUBLIC_KEY]; ok {
		buf.WriteByte(byte(TX_PUBLIC_KEY)) // write marker
		key := tx.Extra_map[TX_PUBLIC_KEY].(crypto.Key)
		buf.Write(key[:]) // write the key
	} else {
		rlog.Tracef(1, "TX does not contain a Public Key, not possible, the transaction will be rejected")
		return buf.Bytes() // as keys are not provided, no point adding other fields
	}

	// extra nonce should be serialized only if other nonce are not provided, tx should contain max 1 nonce
	// it can be either, extra nonce, 32 byte payment id or 8 byte encrypted payment id

	// if payment id are set, they replace nonce
	// first place unencrypted payment id
	if _, ok := tx.PaymentID_map[TX_EXTRA_NONCE_PAYMENT_ID]; ok {
		data_bytes := tx.PaymentID_map[TX_EXTRA_NONCE_PAYMENT_ID].([]byte)
		if len(data_bytes) == 32 { // payment id is valid
			header := append([]byte{byte(TX_EXTRA_NONCE_PAYMENT_ID)}, data_bytes...)
			tx.Extra_map[TX_EXTRA_NONCE] = header // overwrite extra nonce with this
		}
		rlog.Tracef(1, "unencrypted payment id size mismatch expected = %d actual %d", 32, len(data_bytes))
	}

	// if encrypted nonce is provide, it will overwrite 32 byte nonce
	if _, ok := tx.PaymentID_map[TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID]; ok {
		data_bytes := tx.PaymentID_map[TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID].([]byte)
		if len(data_bytes) == 8 { // payment id is valid
			header := append([]byte{byte(TX_EXTRA_NONCE_ENCRYPTED_PAYMENT_ID)}, data_bytes...)
			tx.Extra_map[TX_EXTRA_NONCE] = header // overwrite extra nonce with this
		}
		rlog.Tracef(1, "unencrypted payment id size mismatch expected = %d actual %d", 8, len(data_bytes))
	}

	// TX_EXTRA_NONCE is optional
	// if payment is present, it is packed as extra nonce
	if _, ok := tx.Extra_map[TX_EXTRA_NONCE]; ok {
		buf.WriteByte(byte(TX_EXTRA_NONCE)) // write marker
		data_bytes := tx.Extra_map[TX_EXTRA_NONCE].([]byte)

		if len(data_bytes) > 255 {
			rlog.Tracef(1, "TX extra none is spilling, trimming the nonce to 254 bytes")
			data_bytes = data_bytes[:254]
		}
		buf.WriteByte(byte(len(data_bytes))) // write length of extra nonce single byte
		buf.Write(data_bytes[:])             // write the nonce data
	}
	
	//
	if _, ok := tx.Extra_map[TX_EXTRA_ADDRESS]; ok {
		buf.WriteByte(byte(TX_EXTRA_ADDRESS)) // write marker
		addr := tx.Extra_map[TX_EXTRA_ADDRESS].(address.Address)
		buf.Write(addr.SpendKey[:])             // write the spend key
                buf.Write(addr.ViewKey[:])             // write the spend key
	}
	
	if _, ok := tx.Extra_map[TX_EXTRA_SIG]; ok {
		buf.WriteByte(byte(TX_EXTRA_SIG)) // write marker
		sig := tx.Extra_map[TX_EXTRA_SIG].(crypto.Signature)
		buf.Write(sig.C[:])             // write the C key
                buf.Write(sig.R[:])             // write the R key
	}
	
	if _, ok := tx.Extra_map[TX_EXTRA_SCDATA]; ok {
		buf.WriteByte(byte(TX_EXTRA_SCDATA)) // write marker
                
                
		data_bytes := tx.Extra_map[TX_EXTRA_SCDATA].([]byte)
                
                tbuf := make([]byte, binary.MaxVarintLen64)
                n := binary.PutUvarint(tbuf, uint64(len(data_bytes)))
                buf.Write(tbuf[:n]) 
                fmt.Printf("serializing extra len %x\n", tbuf[:n])
                fmt.Printf("serializing extra data %x\n", data_bytes)
		buf.Write(data_bytes[:])             // write the data bytes
	}

	// NOTE: we do not support adding padding for the sake of it

	return buf.Bytes()

}

// resize the nonce by this much bytes,
// positive means add  byte
// negative means decrease size
// this is only required during miner tx to solve chicken and problem
/*
func (tx *Transaction) Resize_Extra_Nonce(int resize_amount) {

    nonce_bytes := tx.Extra_map[TX_EXTRA_NONCE].([]byte)
    nonce_bytes = make([]byte, len(nonce_bytes)+resize_amount, len(nonce_bytes)+resize_amount)
    tx.Extra_map[TX_EXTRA_NONCE] = nonce_bytes

    tx.Extra = tx.Serialize_Extra()

}
*/
