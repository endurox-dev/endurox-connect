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

//VIEW service - return different view W/O RSP
//@param ac ATMI Context
//@param svc Service call information
func VIEWSV2(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	tlong1, errU := v.BVGetInt64("tlong1", 0, 0)
	ac.TpAssertEqualPanic(errU, nil, "tlong1=> must be nil")

	tstring1, errU := v.BVGetString("tstring1", 1, 0)
	ac.TpAssertEqualPanic(errU, nil, "tstring1=> must be nil")

	v2, errA := ac.NewVIEW("REQUEST2", 0)
	ac.TpAssertEqualPanic(errA, nil, "Request2: errA must be nil")

	errU = v2.BVChg("tshort2", 0, tshort1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tshort2")

	errU = v2.BVChg("tlong2", 0, tlong1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tlong2")

	errU = v2.BVChg("tstring2", 0, tstring1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tstring2")

	v = v2

	return
}

//VIEW service - Failure service, with different buffer
//@param ac ATMI Context
//@param svc Service call information
func VIEWFAIL(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	//Get UBF Handler
	v, _ := ac.CastToVIEW(&svc.Data)

	//Return to the caller
	defer func() {
		ac.TpReturn(atmi.TPFAIL, 0, v, 0)
	}()

	//Test data received
	tshort1, errU := v.BVGetInt16("tshort1", 0, 0)
	ac.TpAssertEqualPanic(errU, nil, "tshort1=> must be nil")

	tlong1, errU := v.BVGetInt64("tlong1", 0, 0)
	ac.TpAssertEqualPanic(errU, nil, "tlong1=> must be nil")

	tstring1, errU := v.BVGetString("tstring1", 1, 0)
	ac.TpAssertEqualPanic(errU, nil, "tstring1=> must be nil")

	v2, errA := ac.NewVIEW("REQUEST2", 0)
	ac.TpAssertEqualPanic(errA, nil, "Request2: errA must be nil")

	errU = v2.BVChg("tshort2", 0, tshort1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tshort2")

	errU = v2.BVChg("tlong2", 0, tlong1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tlong2")

	errU = v2.BVChg("tstring2", 0, tstring1)
	ac.TpAssertEqualPanic(errU, nil, "Request2: tstring2")

	v = v2

	return
}

//VIEW service, FAIL2
//@param ac ATMI Context
//@param svc Service call information
func VIEWFAIL2(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	//Get UBF Handler
	v, _ := ac.CastToVIEW(&svc.Data)

	//Return to the caller
	defer func() {
		ac.TpReturn(atmi.TPFAIL, 0, v, 0)
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
		return
	}

	if errU = v.BVChg("tlong1", 0, 11111); nil != errU {
		ac.TpLogError("Failed to set tlong1: %s", errU.Error())
		return
	}

	if errU = v.BVChg("tstring1", 0, "HELLO RESPONSE"); nil != errU {
		ac.TpLogError("Failed to set tstring1: %s", errU.Error())
		return
	}

	return
}
