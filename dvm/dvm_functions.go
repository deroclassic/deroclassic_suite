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

import "go/ast"
import "strings"

import "github.com/deroclassic/deroclassic_suite/address"

// this files defines  external functions which can be called in DVM
// for example to load and store data from the blockchain and other basic functions

// random number generator is the basis
// however, further investigation is needed if we would like to enable users to use pederson commitments
// they can be used like
// original SC developers delivers a pederson commitment to SC as external oracle
// after x users have played lottery, dev reveals the commitment using which the winner is finalised
// this needs more investigation
// also, more investigation is required to enable predetermined external oracles

// this will handle all internal functions which may be required/necessary to expand DVM functionality
func (dvm *DVM_Interpreter) Handle_Internal_Function(expr *ast.CallExpr, func_name string) (handled bool, result interface{}) {
	var err error
	_ = err
	switch {
        
            // TODO evaluate why not use a blackbox function which can be used for as many returns as possible
            // the function should behave similar to how RDMSR intel instruction works.
            // this can allow as future compatibility etc
        case strings.EqualFold(func_name, "MAJOR_VERSION"):
		if len(expr.Args) != 0 { // expression without limit
			panic("MAJOR_VERSION expects no parameters")
		}else{
                    return true, 0
                }
                    

	case strings.EqualFold(func_name, "Load"):
		if len(expr.Args) != 1 {
			panic("Load function expects a single varible as parameter")
		}
		// evaluate the argument and use the result
		key := dvm.eval(expr.Args[0])
		switch k := key.(type) {

		case uint64:
			return true, dvm.Load(Variable{Type: Uint64, Value: k})
		case string:
			return true, dvm.Load(Variable{Type: String, Value: k})
		default:
			panic("This variable cannot be loaded")
		}
	case strings.EqualFold(func_name, "Exists"):
		if len(expr.Args) != 1 {
			panic("Exists function expects a single varible as parameter")
		}
		// evaluate the argument and use the result
		key := dvm.eval(expr.Args[0])
		switch k := key.(type) {

		case uint64:
			return true, dvm.Exists(Variable{Type: Uint64, Value: k})
		case string:
			return true, dvm.Exists(Variable{Type: String, Value: k})
		default:
			panic("This variable cannot be loaded")
		}

	case strings.EqualFold(func_name, "Store"):
		if len(expr.Args) != 2 {
			panic("Store function expects 2 variables as parameter")
		}
		key_eval := dvm.eval(expr.Args[0])
		value_eval := dvm.eval(expr.Args[1])
		var key, value Variable
		switch k := key_eval.(type) {
		case uint64:
			key = Variable{Type: Uint64, Value: k}

		case string:
			key = Variable{Type: String, Value: k}
		default:
			panic("This variable cannot be stored")
		}

		switch k := value_eval.(type) {
		case uint64:
			value = Variable{Type: Uint64, Value: k}

		case string:
			value = Variable{Type: String, Value: k}
		default:
			panic("This variable cannot be stored")
		}

		dvm.Store(key, value)
		return true, nil

	case strings.EqualFold(func_name, "RANDOM"):
		if len(expr.Args) >= 2 {
			panic("RANDOM function expects 0 or 1 number as parameter")
		}

		if len(expr.Args) == 0 { // expression without limit
			return true, dvm.State.RND.Random()
		}

		range_eval := dvm.eval(expr.Args[0])
		switch k := range_eval.(type) {
		case uint64:
			return true, dvm.State.RND.Random_MAX(k)

		default:
			panic("This variable cannot be randomly generated")
		}
	case strings.EqualFold(func_name, "SCID"):
		if len(expr.Args) != 0 {
			panic("SCID function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.SCID.String()
	case strings.EqualFold(func_name, "BLID"):
		if len(expr.Args) != 0 {
			panic("BLID function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.BLID.String()
	case strings.EqualFold(func_name, "TXID"):
		if len(expr.Args) != 0 {
			panic("TXID function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.TXID.String()

	case strings.EqualFold(func_name, "BLOCK_HEIGHT"):
		if len(expr.Args) != 0 {
			panic("BLOCK_HEIGHT function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.BL_HEIGHT
	case strings.EqualFold(func_name, "BLOCK_TOPOHEIGHT"):
		if len(expr.Args) != 0 {
			panic("BLOCK_TOPOHEIGHT function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.BL_TOPOHEIGHT

	case strings.EqualFold(func_name, "SIGNER"):
		if len(expr.Args) != 0 {
			panic("SIGNER function expects 0 parameters")
		}
		return true, dvm.State.Chain_inputs.Signer.String()

	case strings.EqualFold(func_name, "IS_ADDRESS_VALID"): // checks whether the address is valid DERO address
		if len(expr.Args) != 1 {
			panic("IS_ADDRESS_VALID function expects 1 parameters")
		}

		addr_eval := dvm.eval(expr.Args[0])
		switch k := addr_eval.(type) {
		case string:
			_, err := address.NewAddress(k)
			if err == nil {
				return true, uint64(1)
			}
			return true, uint64(0) // fallthrough not supported in type switch

		default:
			return true, uint64(0)
		}

	case strings.EqualFold(func_name, "ADDRESS_RAW"): // returns a string of 64 bytes if everything is okay
		if len(expr.Args) != 1 {
			panic("ADDRESS_RAW function expects 1 parameters")
		}

		addr_eval := dvm.eval(expr.Args[0])
		switch k := addr_eval.(type) {
		case string:
			addr, err := address.NewAddress(k)
			if err == nil {
				var array [64]byte
				copy(array[:], addr.SpendKey[:])
				copy(array[32:], addr.ViewKey[:])
				return true, string(array[:])
			}

			return true, "" // fallthrough not supported in type switch

		default:
			return true, ""
		}

	case strings.EqualFold(func_name, "SEND_DERO_TO_ADDRESS"):
		if len(expr.Args) != 2 {
			panic("SEND_DERO_TO_ADDRESS function expects 2 parameters")
		}

		addr_eval := dvm.eval(expr.Args[0])
		amount_eval := dvm.eval(expr.Args[1])

		if _, ok := addr_eval.(string); !ok {
			panic("address must be valid DERO network address")
		}

		if _, ok := amount_eval.(uint64); !ok {
			panic("amount must be valid  uint64")
		}

		if amount_eval.(uint64) == 0 {
			return true, amount_eval
		}

		dvm.State.Store.SendExternal(dvm.State.Chain_inputs.SCID, addr_eval.(string), amount_eval.(uint64)) // add record for external transfer

		return true, amount_eval

	}

	return false, nil
}

// the load/store functions are sandboxed and thus cannot affect any other SC storage
// loads  a variable from store
func (dvm *DVM_Interpreter) Load(key Variable) interface{} {

	var found uint64

	result := dvm.State.Store.Load(DataKey{SCID: dvm.State.Chain_inputs.SCID, Key: key}, &found)
	return result.Value
}

// whether a variable exists in store or not
func (dvm *DVM_Interpreter) Exists(key Variable) uint64 {
	var found uint64

	dvm.State.Store.Load(DataKey{SCID: dvm.State.Chain_inputs.SCID, Key: key}, &found)
	return found
}

func (dvm *DVM_Interpreter) Store(key Variable, value Variable) {
	dvm.State.Store.Store(DataKey{SCID: dvm.State.Chain_inputs.SCID, Key: key}, value)
}
