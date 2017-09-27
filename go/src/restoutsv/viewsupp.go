/*
** View support routines
**
** @file viewsupp.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2015, ATR Baltic, Ltd. All Rights Reserved.
** This software is released under one of the following licenses:
** GPL or ATR Baltic's license for commercial use.
** -----------------------------------------------------------------------------
** GPL license:
**
** This program is free software; you can redistribute it and/or modify it under
** the terms of the GNU General Public License as published by the Free Software
** Foundation; either version 2 of the License, or (at your option) any later
** version.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY
** WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
** PARTICULAR PURPOSE. See the GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License along with
** this program; if not, write to the Free Software Foundation, Inc., 59 Temple
** Place, Suite 330, Boston, MA 02111-1307 USA
**
** -----------------------------------------------------------------------------
** A commercial use license is available from ATR Baltic, Ltd
** contact@atrbaltic.com
** -----------------------------------------------------------------------------
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

//Validate view service
//@param ac ATMI Service context
//@param s Service context
//@return error in case of err or nil
func VIEWValidateService(ac *atmi.ATMICtx, s *ServiceMap) error {

	//Set not NULL flag
	if s.View_notnull {
		ac.TpLogInfo("VIEWs in responses will contain non NULL fields only " +
			"(according to view file)")
		s.View_flags |= atmi.BVACCESS_NOTNULL
	}

	if ERRORS_JSON2VIEW == s.Errors_int &&
		(s.Errfmt_view_code == "" || s.Errfmt_view_msg == "") {
		return fmt.Errorf("For json2view errors parameters 'errfmt_view_code' and " +
			"'errfmt_view_msg' must be defined")
	}

	return nil
}

//Reset view error
//@param ac ATMI Context
//@param s service map
//@param v typed view
//@return in case of error ATMI error or nil
func VIEWResetEchoError(ac *atmi.ATMICtx, s *ServiceMap, v *atmi.TypedVIEW) atmi.ATMIError {

	if s.Errors_int == ERRORS_JSON2VIEW {
		errU := VIEWInstallError(s.echoVIEW, s.echoVIEW.BVName(), s.Errfmt_view_code,
			atmi.TPMINVAL, "SUCCEED", s.Errfmt_view_msg)
		if nil != errU {
			ac.TpLogError("Failed to install response in echo view: %s",
				errU.Error())
			return atmi.NewCustomATMIError(atmi.TPEINVAL, fmt.Sprintf("Failed to install "+
				"response in echo view: %s", errU.Error()))
		}
	}

	return nil
}
