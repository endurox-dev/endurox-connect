package main

import (
	atmi "github.com/endurox-dev/endurox-go"
	"fmt"
	"os"
//	u "ubftab"
)

const (
	SUCCEED = 0
	FAIL    = -1
)

//TESTSVC service
//@param ac ATMI Context
//@param svc Service call information
func TESTSVC(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED
	
	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()
	
	
	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
                return
	}
	
	
	//TODO: Run your processing here, and keep the succeed or fail status in 
	//in "ret" flag.

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...");
	//Advertize TESTSVC
	if err := ac.TpAdvertise("TESTSVC", "TESTSVC", TESTSVC); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	return atmi.SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...");
}

//Executable main entry point
func main() {
	//Have some context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Errorf("Failed to allocate context!", err)
		os.Exit(atmi.FAIL)
	} else {
		//Run as server
		ac.TpRun(Init, Uninit)
	}
}
