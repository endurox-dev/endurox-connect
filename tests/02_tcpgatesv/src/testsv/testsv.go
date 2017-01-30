package main

import (
	"fmt"
	"os"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "testsv"
)

//Connection status service
func CONSTAT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "CONSTAT: Incoming request:")

	gateway, _ := ub.BGetString(u.EX_NETGATEWAY, 0)
	con, _ := ub.BGetInt64(u.EX_NETCONNID, 0)
	flag, _ := ub.BGetString(u.EX_NETFLAGS, 0)

	ac.TpLogInfo("CONSTAT: Gatway %s Connection %d status %s",
		gateway, con, flag)
}

//Correlation service
//Will return correlator as first 4x bytes, if buffer larger than 4x bytes
func CORSVC(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "CORSVC: Incoming request:")

	arr, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}
	if arr[0] == 1 && arr[1] == 1 {
		ac.TpLogInfo("Test case 11 - no need for correlation")
	} else if len(arr) > 4 {

		corr := string(arr[:4])

		ac.TpLogInfo("Extracted correlator: [%s]", corr)

		if err := ub.BChg(u.EX_NETCORR, 0, corr); nil != err {
			ac.TpLogError("Failed to set EX_NETCORR: %s", err.Message())
			ret = FAIL
			return
		}

	}

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply buffer afrer correl")

}

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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "TESTSVC: Incoming request:")

	used, _ := ub.BUsed()
	//Resize buffer, to have some more space
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	arr, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}

	//Test case 'A' - with correlation, reply back with cor
	if arr[0] == 'A' {
		ac.TpLogInfo("Running test case A")
		for i := 4; i < len(arr); i++ {
			arr[i] = byte((int(arr[i]+1) % 256))
			//Avoid stx/etx for later tests
			if arr[i] == 2 {
				arr[i] = 5
			}

			if arr[i] == 3 {
				arr[i] = 6
			}
		}

		err = ub.BChg(u.EX_NETDATA, 0, arr)

		if nil != err {
			ac.TpLogError("Failed to set EX_NETDATA: %s", err.Message())
			ret = FAIL
			return
		}

		//Kill the outgoing correlator, otherwise service will not just
		//Send the message, but also put it in waiters list!
		//But we need to send a reply to caller service....

		ub.BDel(u.EX_NETCORR, 0)

		ac.TpACall("TCP_P_ASYNC_A", ub, atmi.TPNOREPLY)

		//Check the if it is first test case (11), then
		//Verify all data sent
	} else if arr[0] == 1 && arr[1] == 1 {

		ac.TpLogInfo("First test case")
		for i := 2; i < 2048; i++ {
			if arr[i] != byte(i%256) {
				ac.TpLogError("TESTERROR: buffer index %d got "+
					"%d expected %d", i, arr[i], byte(i%256))
			}
		}

		ac.TpLogInfo("Test case 11 OK")

	} else {
		//NOTE: This basically is dumped, because we do not do reply back
		//and we were invoked in async way.
		ub.BDel(u.EX_NETDATA, 0)
	}

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply buffer")

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")
	if err := ac.TpInit(); err != nil {
		return FAIL
	}

	//Get the configuration

	//Allocate configuration buffer
	buf, err := ac.NewUBF(16 * 1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s]", err.Error())
		return FAIL
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", PROGSECTION, os.Getenv("NDRX_CCTAG")))

	if _, err := ac.TpCall("@CCONF", buf, 0); nil != err {
		ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return FAIL
	}

	//Dump to log the config read
	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	occs, _ := buf.BOccur(u.EX_CC_KEY)

	// Load in the config...
	for occ := 0; occ < occs; occ++ {
		ac.TpLogDebug("occ %d", occ)
		fldName, err := buf.BGetString(u.EX_CC_KEY, occ)

		if nil != err {
			ac.TpLogError("Failed to get field "+
				"%d occ %d", u.EX_CC_KEY, occ)
			return FAIL
		}

		ac.TpLogDebug("Got config field [%s]", fldName)

		switch fldName {

		case "mykey1":
			myval, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, myval)
			break

		default:

			break
		}
	}
	//Advertize TESTSVC
	if err := ac.TpAdvertise("TESTSVC", "TESTSVC", TESTSVC); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("CORSVC", "CORSVC", CORSVC); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("CONSTAT", "CONSTAT", CONSTAT); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	return SUCCEED
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
