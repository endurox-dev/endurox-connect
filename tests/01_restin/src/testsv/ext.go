package main

import (
	"fmt"
	"strconv"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

// Reply back with request headers
// See Bug #800
func HEADER_ECHO(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//sz, _ := ub.BSizeof()
	//Get return buf...
	ub_ret, errA := ac.NewUBF(1024)

	if nil != errA {
		ac.TpLogError("Failed to alloc buffer: %s", errA.Message())
		ret = FAIL
		return
	}

	first := true
	for true {

		id, occ, err := ub.BNext(first)
		first = false
		if nil != err {
			break
		}

		id_add := id

		//Translate to response headers.
		if id_add == ubftab.EX_IF_REQHN {
			id_add = ubftab.EX_IF_RSPHN
		} else if id_add == ubftab.EX_IF_REQHV {
			id_add = ubftab.EX_IF_RSPHV
		}

		val, errU := ub.BGet(id, occ)
		if nil != errU {
			ac.TpLogError("failed to get %d[%d]: %s", id, occ, errU.Message())
			ret = FAIL
			return
		}

		ac.TpLogDebug("Mapping out %d[%d] [%v]", id, occ, val)

		if errU := ub_ret.BAdd(id_add, val); nil != errU {
			ac.TpLogError("failed to set %d[%d]: %s", id_add, occ, errU.Message())
			ret = FAIL
			return
		}
	}

	//Return new buff...
	ub = ub_ret

}

// Incoming serivce copy some stuff test fields
// @param ac ATMI Context
// @param svc Service call information
func INMAND(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	ac.TpLogDebug("Set Header & Cookies data")

	ub.BAdd(ubftab.EX_IF_RSPHN, "Accept-Language")
	ub.BAdd(ubftab.EX_IF_RSPHV, "EN-US")

	ub.BAdd(ubftab.EX_IF_RSPHN, "Last-Modified")
	ub.BAdd(ubftab.EX_IF_RSPHV, "Tue, 31 Aug 2063 23:59:59 GMT")

	// Set Cookie data
	ub.BAdd(ubftab.EX_IF_RSPCN, "RspCookie")
	ub.BAdd(ubftab.EX_IF_RSPCV, "qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq")
	ub.BAdd(ubftab.EX_IF_RSPCPATH, "/cookie/path")
	ub.BAdd(ubftab.EX_IF_RSPCDOMAIN, "localhost.com")
	ub.BAdd(ubftab.EX_IF_RSPCEXPIRES, "Thu, 08 Nov 2018 10:13:34 GMT")
	ub.BAdd(ubftab.EX_IF_RSPCMAXAGE, "3600")
	ub.BAdd(ubftab.EX_IF_RSPCSECURE, "AAA")
	ub.BAdd(ubftab.EX_IF_RSPCHTTPONLY, "true")

	//Get header
	formName, errU := ub.BGetString(ubftab.EX_IF_REQFORMN, 0)

	if errU != nil {
		ac.TpLogError("Missing EX_IF_REQFORMN")
		ret = FAIL
		return
	}

	formValue, errU := ub.BGetString(ubftab.EX_IF_REQFORMV, 0)

	if errU != nil {
		ac.TpLogError("Missing EX_IF_REQFORMV")
		ret = FAIL
		return
	}

	//Check also some mandatory fields
	//Feature #509
	//EX_IF_METHOD now must be present
	method, errU := ub.BGetString(ubftab.EX_IF_METHOD, 0)

	if nil != errU {
		ac.TpLogError("TESTERROR! Failed to get EX_IF_METHOD: %s", errU.Error())
		ret = FAIL
		return
	}

	if method != "GET" && method != "POST" {
		ac.TpLogError("TESTERROR Test error method expected GET or POST, "+
			"but recieved: [%s]", method)
		ret = FAIL
		return
	}

	//Load the test fields

	ub.BAdd(ubftab.T_STRING_2_FLD, formName)
	ub.BAdd(ubftab.T_STRING_2_FLD, formValue)

	addToNetData("IN_MAND", ub)

	if formName == "E_INMAND" {
		ret = FAIL
	}

	ac.TpLogInfo("Got UBF: [%v]", ub)

	return
}

// Add string to buffer content
func addToNetData(data string, ub *atmi.TypedUBF) {

	//Check is there req data present, we will add there

	cont := ""

	if ub.BPres(ubftab.EX_IF_REQDATA, 0) {
		cont, _ = ub.BGetString(ubftab.EX_IF_REQDATA, 0)
		ub.BDel(ubftab.EX_IF_REQDATA, 0)
	} else {
		cont, _ = ub.BGetString(ubftab.EX_IF_RSPDATA, 0)
	}

	if cont == "" {
		cont = data
	} else {
		cont = cont + "-" + data
	}

	ub.BChg(ubftab.EX_IF_RSPDATA, 0, cont)

}

// Incoming opt service
func INOPT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("IN_OPT", ub)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_INOPT" {
		ret = FAIL
	}
}

// Incoming error service
func INERR(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("INERR", ub)
	ub.BAdd(ubftab.EX_NETRCODE, 503)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_INERR" {
		ret = FAIL
	}

}

// Outgoing error service
func OUTERR(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("OUTERR", ub)
	ub.BChg(ubftab.EX_NETRCODE, 0, 504)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_OUTERR" {
		ret = FAIL
	}

}

// Outgiong mandatory service, fail in case if "T_STRING_2_FLD[0]" is set to "ETEST"
func OUTMAND(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("OUT_MAND", ub)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_OUTMAND" {
		ret = FAIL
	}

}

// opt out service
func OUTOPT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Add the URL to the opt path
	url, _ := ub.BGetString(ubftab.EX_IF_URL, 0)

	addToNetData("OUT_OPT"+url, ub)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_OUTOPT" {
		ret = FAIL
	}

	//Set header type
	ub.BAdd(ubftab.EX_IF_RSPHN, "Content-Type")
	ub.BAdd(ubftab.EX_IF_RSPHV, "application/test")
}

// Incoming OK service
func INOK(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("OK", ub)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	formKey, _ := ub.BGetString(ubftab.T_STRING_2_FLD, 0)

	if formKey == "E_INOK" {
		ret = FAIL
	}

}

// IN Fail service
func INFAIL(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("FAIL", ub)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()
}

// Test Request params
func REQPARAMS(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request (REQPARAMS):")

	arg1 := false
	arg2 := false

	occs, _ := ub.BOccur(ubftab.EX_IF_REQQUERYN)

	for i := 0; i < occs; i++ {

		nam, _ := ub.BGetString(ubftab.EX_IF_REQQUERYN, i)
		val, _ := ub.BGetString(ubftab.EX_IF_REQQUERYV, i)

		if nam == "arg1" && val == "val1" {
			arg1 = true
		} else if nam == "arg2" && val == "val2" {
			arg2 = true
		}
	}

	if arg1 {
		addToNetData("ARG1OK", ub)
	}

	if arg2 {
		addToNetData("ARG2OK", ub)
	}

	ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)

}

// Just receive some request
// Set the tpurcode and in case of data 3, set error response too
func REQERRCODES(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED
	urcode := 0
	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)
	ac.TpLogSetReqFile(ub, "", "")

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, int64(urcode), ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, int64(urcode), ub, 0)
		}
	}()

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request (REQERRCODES):")

	data, _ := ub.BGetString(ubftab.EX_IF_REQDATA, 0)

	urcode, _ = strconv.Atoi(data)

	ac.TpLogInfo("Got data: %s - %d", data, urcode)

	if urcode == 3 {
		ret = FAIL
	}

}

// Filter & prepare body with reporting the actual
// Response
func RSPERRFILTER(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	ac.TpLogSetReqFile(ub, "", "")

	errCode, _ := ub.BGetInt(ubftab.EX_IF_ECODE, 0)
	errMsg, _ := ub.BGetString(ubftab.EX_IF_EMSG, 0)
	urCode, _ := ub.BGetInt(ubftab.EX_IF_TPURCODE, 0)
	src, _ := ub.BGetString(ubftab.EX_IF_ERRSRC, 0)

	rspmsg := fmt.Sprintf("ERR-URCODE-%d-%d-%s-%s", errCode, urCode, src, errMsg)

	ub.BChg(ubftab.EX_IF_RSPDATA, 0, rspmsg)
	ub.BChg(ubftab.EX_NETRCODE, 0, "200")

	ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
}
