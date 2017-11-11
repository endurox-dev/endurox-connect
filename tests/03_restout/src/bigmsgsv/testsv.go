package main

import (
	"fmt"
	"os"

	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED = 0
	FAIL    = -1
)

//BIGMSG service
func BIGMSG(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	//ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")
	ac.TpLogInfo("BIGMSG got call!")

	//Set some field
	testdata, err := ub.BGetByteArr(u.T_CARRAY_FLD, 0)

	if err != nil {
		fmt.Printf("Bchg() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		goto out
	}

	for i := 0; i < len(testdata); i++ {
		if testdata[i] != byte((i+1)%255) {
			ac.TpLogError("TESTERROR: Error at index %d expected %d got: %d",
				i, (i+2)%255, testdata[i])
			ret = FAIL
			goto out
		}

		testdata[i] = byte((i + 2) % 255)
	}

	ac.TpLogInfo("About set test data!")

	if err := ub.BChg(u.T_CARRAY_FLD, 0, testdata); err != nil {
		ac.TpLogError("TESTERROR ! Bchg() 2 Got error: %d:[%s]", err.Code(), err.Message())
		ret = FAIL
		goto out
	}

out:
	//Return to the caller
	if SUCCEED == ret {
		ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
	} else {
		ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
	}
	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")

	if err := ac.TpAdvertise("BIGMSG", "BIGMSG", BIGMSG); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	return atmi.SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...")
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
