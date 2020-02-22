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

package crypto

//import "fmt"

// this file implements signature generation and verification
type Signature struct {
    C Key 
    R Key 
}


func signature_Generate_I(msg_hash Key, public_key Key, secret_key Key, random_scalar *Key, S *Signature){
    var point ExtendedGroupElement
    var point_key Key
     
    // var array_bytes [3]Key // [0] stores msg_hash 
                            // [1] stores public_key 
                            // [2] stores random point
     var array_bytes [3*32]byte
     
    
    //array_bytes[0] = msg_hash
    copy(array_bytes[0:], msg_hash[:])
    //array_bytes[1]= public_key
    copy(array_bytes[32:], public_key[:])
    GeScalarMultBase(&point, random_scalar)
    point.ToBytes(&point_key)
    copy(array_bytes[64:], point_key[:])
    
   // fmt.Printf("tmp3 %s", point_key)
    
   // fmt.Printf("garray %+v\n", array_bytes)
    
    S.C = *(HashToScalar(array_bytes[:])) // this must be copied to signature
    
    ScMulSub(&S.R, &S.C, &secret_key, random_scalar)
    
    //fmt.Printf("result sig %+v\n", S)
}

// this creates a signature
func Signature_Generate(msg_hash Key, public_key Key, secret_key Key, S *Signature){
    
    random_scalar := RandomScalar()
   /* msg_hash = HexToKey("f63c961bb5086f07773645716d9013a5169590fd7033a3bc9be571c7442c4c98")
    public_key = HexToKey("b8970905fbeaa1d0fd89659bab506c2f503e60670b7afd1cb56a4dfe8383f38f")
    secret_key = HexToKey("7bb35441e077be8bb8d77d849c926bf1dd0e696c1c83017e648c20513d2d6907")
    
    
    random_scalar := HexToKey("64f286b9b369a6702c8112e42cbee4746a5ba514c3dbb211d314674037bb9808")
    */
    
    signature_Generate_I(msg_hash, public_key,secret_key,random_scalar,S)
}
        
// verifies a signature generated above
func Signature_Verify(msg_hash Key, public_key Key, S * Signature) (result bool){
    
    
    var point ExtendedGroupElement
    var tmp Key
    var array_bytes [3*32]byte
    
    
  /*   msg_hash = HexToKey("57fd3427123988a99aae02ce20312b61a88a39692f3462769947467c6e4c3961")
    public_key = HexToKey("a5e61831eb296ad2b18e4b4b00ec0ff160e30b2834f8d1eda4f28d9656a2ec75")
    
    
    S.C = HexToKey("cd89c4cbb1697ebc641e77fdcd843ff9b2feaf37cfeee078045ef1bb8f0efe0b")
    S.R = HexToKey("b5fd0131fbc314121d9c19e046aea55140165441941906a757e574b8b775c008")
    */
    /*if !public_key.Public_Key_Valid()  {
        return false
    }*/
    
    if public_key == Zero || public_key == Identity  {
        return false
    }
    
    if point.FromBytes(&public_key) == false {
        return false
    }
    
    if Sc_check(&S.C) ==  false || Sc_check(&S.R) == false {
        return false
    }
    
    
 //  fmt.Printf("sig C %s\n", S.C);
 //  fmt.Printf("sig R %s\n", S.R);
 //  fmt.Printf("public_key %s\n", public_key);
   
    copy(array_bytes[0:], msg_hash[:])
    copy(array_bytes[32:], public_key[:])
    
    

    
    // equivalent to ge_double_scalarmult_base_vartime(&tmp2, &sig.c, &tmp3, &sig.r);
    AddKeys2(&tmp,  &S.R,  &S.C, &public_key)
    
    //fmt.Printf("buf comm %s\n", tmp)
    copy(array_bytes[64:], tmp[:])
    
    //fmt.Printf("varray %+v\n", array_bytes)
    
    C := HashToScalar(array_bytes[:])
    
    ScSub(C,C,&S.C)
    
    //fmt.Printf("after verification %s  %s %+v\n", C, S.C, ScIsZero(C))
    
    return ScIsZero(C)

}

