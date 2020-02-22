package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
        "reflect"

)

import "github.com/chzyer/readline"
import "github.com/deroproject/derosuite/dvm"
import "github.com/deroproject/derosuite/crypto"

func main() {
	err := run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

type TabCompleter struct{}

var dummy_autocomplete TabCompleter
var global_sc dvm.SmartContract

func (t TabCompleter) Do(line []rune, linelen int) ([][]rune, int) {
        var result [][]rune 
        
        for k,_ := range global_sc.Functions {
            
            runek := []rune(k)
            if linelen == 0 { // add all results
             result = append(result,[]rune(k))
             continue
            }
            if len([]rune(k)) >= linelen && reflect.DeepEqual(runek[:linelen], line[:linelen]) {
                
                result = append(result,[]rune(k[linelen:]))
             continue
                
            }
        }
        
	return result, linelen
}

// this sets up the interpreter
func debug_run(sc * dvm.SmartContract, code string) {
    
    
   /* defer func() {
        if r := recover(); r != nil { 
            fmt.Printf("panic occurred", r)
        }
    }() */
    
    temp_sc, _, err := dvm.ParseSmartContract(`Function REPL() Uint64
                                             50 RETURN ` + code + `
                                             End Function`)
                                             
                                             
    if err != nil {
        fmt.Printf("err while parsing REPL SC err %s\n",err)
        return
    }
    
    // install our function in the original SC 
    sc.Functions["REPL"] = temp_sc.Functions["REPL"]
    
    
    state := &dvm.Shared_State {
        Chain_inputs:&dvm.Blockchain_Input{BL_HEIGHT:5 , BL_TOPOHEIGHT:9,SCID: crypto.Identity,
                    BLID: crypto.Identity,TXID: crypto.Identity,},       
        }
                    
    _,err = dvm.RunSmartContract ( sc, "REPL",state,map[string]interface{}{})
    
    fmt.Printf("err while executing SC err %s\n",err)
    
    fmt.Printf("Recursion %d\n", state.Monitor_recursion)
    fmt.Printf("Interpreted %d\n", state.Monitor_lines_interpreted)
    fmt.Printf("Evaluated %d\n", state.Monitor_ops)

}

func run() error {
	var data []byte
	if len(os.Args) > 1 {
		if strings.HasPrefix(os.Args[1], "-") {
			return fmt.Errorf("usage: dvm [<source file>]")
		}
		f, err := os.Open(os.Args[1])
		if err != nil {
			return err
		}
		defer f.Close()
		data, err = ioutil.ReadAll(f)
		if err != nil {
			return err
		}
		
		
            
	}
	
	 sc, pos, err := dvm.ParseSmartContract(string(data))
                if err != nil {
                    fmt.Printf("Error while parsing smart contract pos %s err : %s\n", pos,err)   
                return err
                }
                
        global_sc = sc
        debug_run(&sc,"fact(20)")
        debug_run(&sc,"factr(20)")
        
        //return nil
        

	var prompt string = "\033[92mDERO DVM:\033[32m>>>\033[0m "
        
	// We need to initialize readline first, so it changes stderr to ansi processor on windows
	l, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     "",
		AutoComplete:    dummy_autocomplete,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()
        


	for {
                line, err := l.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				fmt.Printf("Ctrl-C received, Exit in progress\n")
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}
		
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		if strings.EqualFold(line,"exit"){
                    break;
                }
		
		debug_run(&sc,line)
		
		
		
	}
	fmt.Printf("Exiting")
	return nil
}


// filter out specfic inputs from input processing
// currently we only skip CtrlZ background key
func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
        }
	return r, true
}
