package main

import (
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

var Msequence byte = 0

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

	Msequence++

	return
}
