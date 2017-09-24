package main

import atmi "github.com/endurox-dev/endurox-go"

//VIEW service
//@param ac ATMI Context
//@param svc Service call information
func VIEWSV1(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	v, _ := ac.CastToVIEW(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, v, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, v, 0)
		}
	}()

	//Test data received
	tshort1, errU := v.BVGetInt16("tshort1", 0, 0)
	ac.TpAssertEqualPanic(errU, nil, "tshort1=> must be nil")
	ac.TpAssertEqualPanic(tshort1, 5, "tshort1 value")

	tlong1, errU := v.BVGetInt64("tlong1", 0, 0)
	ac.TpAssertEqualPanic(errU, nil, "tlong1=> must be nil")
	ac.TpAssertEqualPanic(tlong1, 77777, "tlong1 value")

	tstring1, errU := v.BVGetString("tstring1", 1, 0)
	ac.TpAssertEqualPanic(errU, nil, "tstring1=> must be nil")
	ac.TpAssertEqualPanic(tstring1, "INCOMING TEST", "tstring1 value")

	//Set response data

	if errU = v.BVChg("tshort1", 0, 8); nil != errU {
		ac.TpLogError("Failed to set tshort1: %s", errU.Error())
		ret = FAIL
		return
	}

	if errU = v.BVChg("tlong1", 0, 11111); nil != errU {
		ac.TpLogError("Failed to set tlong1: %s", errU.Error())
		ret = FAIL
		return
	}

	if errU = v.BVChg("tstring1", 0, "HELLO RESPONSE"); nil != errU {
		ac.TpLogError("Failed to set tstring1: %s", errU.Error())
		ret = FAIL
		return
	}

	return
}
