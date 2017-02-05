package main

import (
	atmi "github.com/endurox-dev/endurox-go"
)

//Binary service
//@param ac ATMI Context
//@param svc Service call information
func BINARYSV(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	bb, _ := ac.CastToCarray(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, bb, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, bb, 0)
		}
	}()

	bb.GetBytes()
	ac.TpLogDump(atmi.LOG_INFO, "Got binary request buffer",
		bb.GetBytes(), len(bb.GetBytes()))

	bb.SetBytes([]byte{9,8,7,6,5,4,3,2,1,0})

	ac.TpLogDump(atmi.LOG_INFO, "Responding with buffer", bb.GetBytes(), len(bb.GetBytes()))

	return
}
