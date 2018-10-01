/**
 * @brief View support
 *
 * @file viewsupp.go
 */
/* -----------------------------------------------------------------------------
 * Enduro/X Middleware Platform for Distributed Transaction Processing
 * Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
 * Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
 * This software is released under one of the following licenses:
 * AGPL or Mavimax's license for commercial use.
 * -----------------------------------------------------------------------------
 * AGPL license:
 * 
 * This program is free software; you can redistribute it and/or modify it under
 * the terms of the GNU Affero General Public License, version 3 as published
 * by the Free Software Foundation;
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU Affero General Public License, version 3
 * for more details.
 *
 * You should have received a copy of the GNU Affero General Public License along 
 * with this program; if not, write to the Free Software Foundation, Inc., 
 * 59 Temple Place, Suite 330, Boston, MA 02111-1307 USA
 *
 * -----------------------------------------------------------------------------
 * A commercial use license is available from Mavimax, Ltd
 * contact@mavimax.com
 * -----------------------------------------------------------------------------
 */
package main

import (
	"fmt"

	atmi "github.com/endurox-dev/endurox-go"
)

//Install error code in the view, Note max allowed error message size is 1024 bytes
//including EOS
//@param buf VIEW buffer
//@param code error code to install
//@param msg error message to install
//@return nil on OK, UBF error on failure
func VIEWInstallError(buf *atmi.TypedVIEW, view string, view_code_fld string,
	code int, view_msg_fld string, msg string) atmi.UBFError {

	//Check the length of the message field

	_, _, _, dim_size, _, errU := buf.BVOccur(view_msg_fld)

	if nil != errU {
		buf.Buf.Ctx.TpLogError("Failed to get %s.%s infos: %s",
			view, view_msg_fld, errU.Error())
		return errU
	}

	buf.Buf.Ctx.TpLogInfo("message field dim size: %d", dim_size)
	//Conver message to bytes:

	byteStr := []byte(msg)

	//+1 for C EOS
	if int64(len(byteStr)+1) > dim_size {
		byteStr = byteStr[0 : dim_size-1] //1 for EOS
	}

	if errU := buf.BVChg(view_code_fld, 0, code); nil != errU {
		buf.Buf.Ctx.TpLogError("Failed to test/set code in resposne %s.[%s] to [%d]: %s",
			view, view_code_fld, code, errU.Error())
		return errU
	}

	if errU := buf.BVChg(view_msg_fld, 0, byteStr); nil != errU {
		buf.Buf.Ctx.TpLogError("Failed to test/set message in resposne %s.[%s] to [%d]: %s",
			view, view_msg_fld, string(byteStr), errU.Error())
		return errU
	}

	return nil
}

//Validate view service settings
//@param ac ATMI context
//@param svc 	Service map
//@return error or nil
func VIEWSvcValidateSettings(ac *atmi.ATMICtx, svc *ServiceMap) error {

	//Set not NULL flag
	if svc.View_notnull {
		ac.TpLogInfo("VIEWs in responses will contain non NULL fields only " +
			"(according to view file)")
		svc.View_flags |= atmi.BVACCESS_NOTNULL
	}

	if svc.Errors_int != ERRORS_JSON2VIEW {
		return nil //Nothing to validate
	}

	//For async calls we need response object
	if svc.Asynccall && !svc.Asyncecho && svc.Errfmt_view_rsp == "" {
		err := fmt.Errorf("Tag 'errfmt_view_rsp' must set in case if 'async' " +
			"is set and 'asyncecho' is not set")
		ac.TpLogError(err.Error())
		return err
	}

	//Error fields must be present
	ac.TpLogInfo("Errfmt_view_msg=[%s] Errfmt_view_code=[%s]",
		svc.Errfmt_view_msg, svc.Errfmt_view_code)
	if "" == svc.Errfmt_view_msg || "" == svc.Errfmt_view_code {
		err := fmt.Errorf("Tags 'errfmt_view_code' and 'errfmt_view_msg' " +
			"must be present in case of 'json2view' errors")
		ac.TpLogError(err.Error())
		return err
	}

	//If response goes first, the response view must be set
	if svc.Errfmt_view_rsp_first && "" == svc.Errfmt_view_rsp {
		err := fmt.Errorf("If responding with response view first " +
			"('errfmt_view_rsp_first' true), tag 'errfmt_view_rsp' must be set")
		ac.TpLogError(err.Error())
		return err
	}

	//Test the response object if have one.

	ac.TpLogInfo("Testing view: %s setting code in %s and message in %s",
		svc.Errfmt_view_rsp, svc.Errfmt_view_code,
		svc.Errfmt_view_msg)

	if svc.Errfmt_view_rsp != "" {
		buf, errA := ac.NewVIEW(svc.Errfmt_view_rsp, 0)

		if nil != errA {
			err := fmt.Errorf("Failed to alloc VIEW/[%s]: %s",
				svc.Errfmt_view_rsp, errA.Error())
			ac.TpLogError(err.Error())
			return err
		}

		errA = VIEWInstallError(buf, svc.Errfmt_view_rsp,
			svc.Errfmt_view_code, 0, svc.Errfmt_view_msg,
			"SUCCEED")
	}

	return nil
}

//Generate response from view configured
//@param ac	ATMI Context
//@param svc	Servic map
//@param atmiErr ATMI error to put in response
//@return In case of error return []
func VIEWGenDefaultResponse(ac *atmi.ATMICtx, svc *ServiceMap, atmiErr atmi.ATMIError) []byte {
	//In this case response VIEW buffer must be set.

	if nil == atmiErr {
		atmiErr = atmi.NewCustomATMIError(atmi.TPMINVAL, "SUCCEED")
	}
	bufv, errA := ac.NewVIEW(svc.Errfmt_view_rsp, 0)

	if nil != errA {
		ac.TpLogError("Failed to alloc VIEW/[%s] - dropping response: %s",
			svc.Errfmt_view_rsp, errA.Error())
		ac.UserLog("Failed to alloc VIEW/[%s] - dropping response: %s",
			svc.Errfmt_view_rsp, errA.Error())
		return []byte("{}")
	}

	if errU := VIEWInstallError(bufv, svc.Errfmt_view_rsp,
		svc.Errfmt_view_code, atmiErr.Code(), svc.Errfmt_view_msg,
		atmiErr.Message()); nil != errU {

		ac.TpLogError("Failed to set viewe response - dropping: %s",
			atmiErr.Message())

		ac.UserLog("Failed to set view response - dropping: %s",
			atmiErr.Message())

		return []byte("{}")

	}

	//The resposne view contains all field no matter of the non-null setting
	ret, err1 := bufv.TpVIEWToJSON(0)

	if nil == err1 {
		//Generate the resposne buffer...
		rsp := []byte(ret)

		return rsp

	} else {
		ac.TpLogError("Failed to convert VIEW to JSON - dropping response: %s",
			err1.Error())

		ac.UserLog("Failed to convert VIEW to JSON - dropping response: %s",
			err1.Error())

		return nil
	}
}
/* vim: set ts=4 sw=4 et smartindent: */
