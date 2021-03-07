/**
 * @brief Transaction queue interface service. This will add message to queue
 * and read message from queue. The caller via restincl will control the transactions
 * and thus add/gets should follow the transactional nature.
 *
 * @file transv.go
 */
package main

import (
	"fmt"
	"os"

	atmi "github.com/endurox-dev/endurox-go"
)

//Add message to queue
//@param ac ATMI Context
//@param svc Service call information
func QADD(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := atmi.SUCCEED

	var qctl atmi.TPQCTL
	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if atmi.SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Enqueue the string
	if err := ac.TpEnqueue("QSPACE1", "MYQ1", &qctl, ub, 0); nil != err {
		fmt.Printf("TpEnqueue() failed: ATMI Error %d:[%s]\n", err.Code(), err.Message())
		ret = atmi.FAIL
		return
	}

	return
}

//Get message from queue
//@param ac ATMI Context
//@param svc Service call information
func QGET(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := atmi.SUCCEED

	var qctl atmi.TPQCTL

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if atmi.SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get the msg
	if err := ac.TpDequeue("QSPACE1", "MYQ1", &qctl, &svc.Data, 0); nil != err {
		fmt.Printf("TpDequeue() failed: ATMI Error %d:[%s]\n", err.Code(), err.Message())
		ret = atmi.FAIL
		return
	}

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")

	if err := ac.TpAdvertise("QADD", "QADD", QADD); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpOpen(); err != nil {
		ac.TpLogError("Failed to tpopen: %s", err.Error())
		return atmi.FAIL
	}

	return atmi.SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...")
	ac.TpClose()
}

//Executable main entry point
func main() {
	//Have some context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s", err)
		os.Exit(atmi.FAIL)
	} else {
		//Run as server
		ac.TpRun(Init, Uninit)
	}
}
