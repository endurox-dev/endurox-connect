package main

import (
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

var Msequence byte = 0

var Mmsgs int64	= 0 //messages received

//Test message sequence (seqin/seqout)
//@param ac ATMI Context
//@param svc Service call information
func SEQTEST(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	ba, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("TESTERROR Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}

	if ba[0] != Msequence {

		ac.TpLogError("TESTERROR: Expected %d got %d", Msequence, ba[0])

	}

	Mmsgs++;
	Msequence++

	return
}

//Return number of messages sequenced
func SEQRES(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	
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
	
	used, _ := ub.BUsed()
	
	//Resize buffer, to have some more space
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}
	
	if err := ub.BChg(u.T_LONG_FLD, 0, Mmsgs); nil != err {
		ac.TpLogError("Failed to set T_LONG_FLD: %s", err.Message())
		ret = FAIL
		return
	}

}
