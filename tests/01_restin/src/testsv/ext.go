package main

import (
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

//Incoming serivce copy some stuff test fields
//@param ac ATMI Context
//@param svc Service call information
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

//Add string to buffer content
func addToNetData(data string, ub *atmi.TypedUBF) {

	cont, _ := ub.BGetString(ubftab.EX_NETDATA, 0)

	if cont == "" {
		cont = data
	} else {
		cont = cont + "-" + data
	}

	ub.BChg(ubftab.EX_NETDATA, 0, cont)

}

//Incoming opt service
func INOPT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BAdd(ubftab.T_STRING_3_FLD, "IN_OPT")
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

//Incoming error service
func INERR(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BAdd(ubftab.EX_NETDATA, "INERR")
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

//Outgoing error service
func OUTERR(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BChg(ubftab.EX_NETDATA, 0, "OUTERR")
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

//Outgiong mandatory service, fail in case if "T_STRING_2_FLD[0]" is set to "ETEST"
func OUTMAND(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	addToNetData("OUT_MAND", ub)
	//	ub.BAdd(ubftab.EX_NETDATA, "OUT_MAND")

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

//opt out service
func OUTOPT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BAdd(ubftab.T_STRING_3_FLD, "OUT_OPT")

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

//Incoming OK service
func INOK(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BAdd(ubftab.EX_NETDATA, "OK")
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

//IN Fail service
func INFAIL(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//ub.BAdd(ubftab.EX_NETDATA, "FAIL")
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
