package main

import (
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

//Cookies UBF service
//@param ac ATMI Context
//@param svc Service call information
func COOKIES(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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

	ac.TpLogInfo("Got UBF: [%v]", ub)

	return
}
