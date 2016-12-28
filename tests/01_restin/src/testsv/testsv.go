package main

import (
	"fmt"
	"os"
	"time"

	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED = 0
	FAIL    = -1
)

//Will set the trace file
func GETFILE(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	ub.BChg(u.EX_NREQLOGFILE, 0, fmt.Sprintf("%s/log/TRACE_%d",
		os.Getenv("NDRX_APPHOME"), time.Now().UnixNano()))

	return
}

//DATASV1 service
//@param ac ATMI Context
//@param svc Service call information
func DATASV1(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	//Set request logging
	ac.TpLogSetReqFile(&ub, "", "")

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//TODO: Run your processing here, and keep the succeed or fail status in
	//in "ret" flag.

	//Copy the incoming data from 1 to second field
	char_val, err := ub.BGetByte(u.T_CHAR_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_CHAR_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_CHAR_2_FLD, 0, char_val)

	if nil != err {

		ac.TpLogError("Failed to set T_CHAR_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")
	//Advertize TESTSVC
	if err := ac.TpAdvertise("DATASV1", "DATASV1", DATASV1); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("GETFILE", "GETFILE", GETFILE); err != nil {
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
