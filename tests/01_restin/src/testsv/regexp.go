package main

import atmi "github.com/endurox-dev/endurox-go"

//Regexp UBF service
//@param ac ATMI Context
//@param svc Service call information
func REGEXP(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	ac.TpLogInfo("Got UBF: [%v]", ub)

	return
}

//Regexp JSON service
//@param ac ATMI Context
//@param svc Service call information
func REGEXPJSON(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	jb, _ := ac.CastToJSON(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, jb, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, jb, 0)
		}
	}()

	ac.TpLogInfo("Got json: [%v]", jb)

	return
}
