package main

import atmi "github.com/endurox-dev/endurox-go"

//Text service
//@param ac ATMI Context
//@param svc Service call information
func TEXTSV(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	sb, _ := ac.CastToString(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, sb, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, sb, 0)
		}
	}()

	ac.TpLogWarn("Got string request...")

	ac.TpLogInfo("Got text: [%s]", sb.GetString())

	sb.SetString("Hello from EnduroX")

	return
}
