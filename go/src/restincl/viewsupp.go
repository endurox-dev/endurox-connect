/*
** View support
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
func ViewInstallError(buf *TypedVIEW, view *string, view_code_fld *string,
	code int, view_msg_fld string, msg string) atmi.UBFError {

	if errU := buf.BVchg(view_code_fld, code, 0); nil != errU {
		u.Buf.Ctx.TpLogError("Failed to test/set code in resposne %s.[%s] to [%d]: %s",
			view, view_code_fld, code, errU.Error())
		return errU
	}

	if errU := buf.BVchg(view_msg_fld, msg, 0); nil != errU {
		u.Buf.Ctx.TpLogError("Failed to test/set message in resposne %s.[%s] to [%d]: %s",
			view, view_msg_fld, msg, errU.Error())
		return errU
	}
}

//Validate view service settings
func ViewSvcValidateSettings(ac *atmi.ATMICtx, svc *ServiceMap) error {

	if svc.Errors_int != ERRORS_JSON2UBF {
		return //Nothing to validate
	}

	//For async calls we need response object
	if svc.Asynccall && !svc.Asyncecho && svc.Errfmt_view_rsp == "" {
		err := fmt.Errorf("Tag 'errfmt_view_rsp' must set in case if 'async' " +
			"is set and 'asyncecho' is not set")
		ac.TpLogError(err.Error())
		return err
	}

	//Error fields must be present
	if "" == svc.Errfmt_view_msg || "" == svc.Errfmt_view_code {
		err := fmt.Errorf("Tags 'errfmt_view_code' and 'errfmt_view_msg' " +
			"in case of 'json2view' erros")
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
		buf, errA := ac.NewView(svc.Errfmt_view_rsp, 0)

		if nil != errA {
			err := fmt.Errorf("Failed to alloc VIEW/[%s]: %s",
				svc.Errfmt_view_rsp, errA.Error())
			ac.TpLogError(err.Error())
			return err
		}

		errA = ViewInstallError(buf, svc.Errfmt_view_rsp,
			svc.Errfmt_view_code, 0, svc.Errfmt_view_msg,
			"SUCCEED")
	}
}
