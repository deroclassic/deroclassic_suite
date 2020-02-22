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

import "runtime/debug"

import "github.com/romana/rlog"

import "github.com/deroclassic/deroclassic_suite/crypto"
import "github.com/deroclassic/deroclassic_suite/address"

// this structure is json/msgpack encodeable to enable seamless RPC support
// this is passes as tx data
type SC_Transaction struct {
        SC  string  `msgpack:"SC,omitempty" json:"sc,omitempty"`  // smart contract  to be installed is provided here
	SCID  crypto.Key  `msgpack:"I,omitempty" json:"scid,omitempty"` // to which smart contract is the entrypoint directed, 64 bytes hex
	EntryPoint string  `msgpack:"E,omitempty" json:"entrypoint,omitempty"`
	Params map[string]string `msgpack:"P,omitempty" json:"params,omitempty"`// all parameters in named form
	
	Value  uint64   `msgpack:"-" json:"value,omitempty"` // DERO to transfer to SC
}



// verifies  a SC signature
func (tx *Transaction) Verify_SC_Signature() (result bool) {
    
    defer func (){
		if r := recover(); r != nil {
				rlog.Warnf("Recovered while Verify SC Signature, Stack trace below block_hash %s", tx.GetHash())
				rlog.Warnf("Stack trace  \n%s", debug.Stack())
				result = false
			}
	        }()
                
                // if extra is not parsed, parse it now
    if len(tx.Extra_map) <= 0  {
        if !tx.Parse_Extra() { // parsing extra failed
         return   
        }
    }
    
    
    // if public address is not provided, reject it
    // if type is not what we expect, reject
    var addri, sigi, datai  interface{}
    addri , result =  tx.Extra_map[TX_EXTRA_ADDRESS]
    if !result {
        return
    }
    
    addr,result := addri.(address.Address)
    if !result {
        return
    }
    
    sigi , result =  tx.Extra_map[TX_EXTRA_SIG]
    if !result {
        return
    }
    sig,result := sigi.(crypto.Signature)
    if !result {
        return
    }
    
    datai , result =  tx.Extra_map[TX_EXTRA_SCDATA]
    if !result {
        return
    }
    data,result := datai.([]byte)
    if !result {
        return
    }
    
    
    // use the first key image as mechanism to stop replay attacks of all forms
    first_keyimage := crypto.Key(tx.Vin[0].(Txin_to_key).K_image)
    msg_hash := crypto.Key(crypto.Keccak256( data,first_keyimage[:]))
    
    return crypto.Signature_Verify(msg_hash, addr.SpendKey, &sig)
}
