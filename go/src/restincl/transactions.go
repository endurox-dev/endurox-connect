/**
 * @brief Transaction API and context handling
 *
 * @file transactions.go
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
	"encoding/json"
	"fmt"
	"net/http"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	TX_REQ_HDR = "endurox-tptranid-req"
	TX_RSP_HDR = "endurox-tptranid-rsp"

	OP_TPBEGIN  = "tpbegin"
	OP_TPCOMMIT = "tpcommit"
	OP_TPABORT  = "tpabort"
)

/**
 * Transaction API request
 */
type TxReqData struct {
	Operation string `json:"operation"`
	Timeout   uint64 `json:"timeout"`
	Flags     int64  `json:"flags"`
	Tptranid  string `json:"tptranid"`
}

/**
 * Transaction API response
 */
type TxRspData struct {
	Operation    string `json:"operation,omitempty"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Tptranid     string `json:"tptranid,omitempty"`
}

/**
 * Transaction handler entry.
 * Assumes that buffers are encoded in "ext" mode
 * TODO: Needs to think about http status codes. Maybe in case of failure, we could
 * give basic indication to caller, that something failed.
 *
 * @param ac ATMI Context
 * @param buf ATMI buffer to call
 * @param svc service mapping to call
 * @param req request object
 * @param w response object
 * @param rctx request context
 * @param flags call flags
 */
func txHandler(ac *atmi.ATMICtx, buf atmi.TypedBuffer, svc *ServiceMap, req *http.Request,
	w http.ResponseWriter, rctx *RequestContext, flags int64) (ret atmi.ATMIError) {

	var reqData TxReqData
	var rspData TxRspData
	var err atmi.ATMIError
	bufu, ok := buf.(*atmi.TypedUBF)

	if !ok {
		ac.TpLogError("ERROR: txHandler - got non UBF buffer")
		return atmi.NewCustomATMIError(atmi.TPESYSTEM, "Invalid buffer")
	}

	body, errU := bufu.BGetByteArr(ubftab.EX_IF_REQDATA, 0)
	if nil != errU {

		ac.TpLogError("ERROR: txHandler - failed to get EX_IF_REQDATA: %s",
			errU.Error())

		return atmi.NewCustomATMIError(atmi.TPESYSTEM,
			fmt.Sprintf("Failed to get ubftab.EX_IF_REQDATA: %s", errU.Error()))
	}

	//If we have valid buffer, we can start to generate
	//Normal json responses
	defer func() {

		http_status := http.StatusOK

		if nil != ret {
			rspData.ErrorCode = ret.Code()
			rspData.ErrorMessage = ret.Message()
		} else {
			rspData.ErrorCode = 0
			rspData.ErrorMessage = "Succeed"
		}

		//Return http response codes correspodingly & marshal the response

		if nil != ret && (ret.Code() == atmi.TPEINVAL || ret.Code() == atmi.TPEPROTO) {

			http_status = http.StatusBadRequest

		} else if nil != ret && ret.Code() > 0 {

			//in this case it is 500
			http_status = http.StatusInternalServerError
		}

		//Load the response body...

		rspBody, err := json.Marshal(&rspData)

		if nil != err {
			ac.TpLogError("Failed to prepare response: %s", err.Error())
			http_status = http.StatusInternalServerError
		} else {

			err := bufu.BChg(ubftab.EX_IF_RSPDATA, 0, rspBody)

			if err != nil {
				ac.TpLogError("Failed to set response body: %s", err.Error())
				http_status = http.StatusInternalServerError
			}
		}

		//Setup the response code finally

		if err := bufu.BChg(ubftab.EX_NETRCODE, 0, http_status); nil != err {
			ac.TpLogError("Failed to set EX_NETRCODE to %d: %s", http_status, err.Error())
		}

	}()

	//Parse the request...
	//Invalid request we
	errJ := json.Unmarshal(body, &reqData)

	if nil != errJ {
		return atmi.NewCustomATMIError(atmi.TPEINVAL,
			fmt.Sprintf("Failed to parse JSON request: %s", errJ.Error()))
	}

	rspData.Operation = reqData.Operation
	rspData.Tptranid = reqData.Tptranid

	//Check do we recognize the function
	ac.TpLogInfo("txHandler: operation:  [%s], timeout: %d, flags: %d tptranid: [%s]",
		reqData.Operation, reqData.Timeout, reqData.Flags, reqData.Tptranid)

	if reqData.Operation == OP_TPCOMMIT || reqData.Operation == OP_TPABORT {
		//Resume transaction
		err = ac.TpResumeString(reqData.Tptranid, 0)

		if nil != err {
			ac.TpLogError("%s: failed to resume transaction: %s",
				reqData.Operation, err.Error())

			return err
		}
	}

	switch reqData.Operation {

	case OP_TPBEGIN:

		err = ac.TpBegin(reqData.Timeout, reqData.Flags)

		if nil != err {
			ac.TpLogError("Failed to begin transaction: %s", err.Error())
			return err
		}

		//Suspend transactions & get TID
		tid, err := ac.TpSuspendString(0)

		if nil != err {
			ac.TpLogError("tpbegin: Failed to suspend transaction: %s", err.Error())
			return err
		}

		rspData.Tptranid = tid

		ac.TpLogInfo("Started transaction: [%s]", rspData.Tptranid)

	case OP_TPCOMMIT:
		err = ac.TpCommit(0)

		if nil != err {
			ac.TpLogError("Failed to commit transaction: %s", err.Error())
			//In any case, context now becomes disasociated from tran
			return err
		}

	case OP_TPABORT:

		err = ac.TpAbort(0)

		if nil != err {
			ac.TpLogError("Failed to abort transaction: %s", err.Error())
			//In any case, context now becomes disasociated from tran
			return err
		}

	default:
		return atmi.NewCustomATMIError(atmi.TPEINVAL,
			fmt.Sprintf("Unsupported operation: [%s]", reqData.Operation))

	}

	return nil

}

/**
 * Transaction service call, in case if transaction headers are present
 * otherwise just normal call
 * @param ac ATMI Context
 * @param buf ATMI buffer to call
 * @param svc service mapping to call
 * @param req request object
 * @param w response object
 * @param rctx request context
 * @param flags call flags
 */
func txCall(ac *atmi.ATMICtx, buf atmi.TypedBuffer, svc *ServiceMap, req *http.Request,
	w http.ResponseWriter, rctx *RequestContext, flags int64) atmi.ATMIError {

	var err atmi.ATMIError

	tidreq := req.Header.Get(TX_REQ_HDR)

	if tidreq != "" {

		var resum_flags int64

		if svc.TxNoOptim {
			resum_flags |= atmi.TPTXNOOPTIM
		}

		err = ac.TpResumeString(tidreq, resum_flags)

		if nil != err {
			ac.TpLogError("Failed to resume transaction [%s] for svc call [%s]",
				tidreq, svc.Svc)
			ac.UserLog("Failed to resume transaction [%s] for svc call [%s]",
				tidreq, svc.Svc)
			return err
		}

		ac.TpLogDebug("Resumed global transaction [%s]", tidreq)
	}

	if svc.NoAbort {
		flags |= atmi.TPNOABORT
	}

	_, err = ac.TpCall(svc.Svc, buf, flags|atmi.TPTRANSUSPEND)

	if ac.TpGetLev() > 0 {

		tidrsp, err_susp := ac.TpSuspendString(0)

		if nil != err_susp {
			ac.TpLogError("Failed to suspend transaction for %s call: %s", svc.Svc,
				err_susp.Message())
			ac.UserLog("Failed to suspend transaction for %s call: %s", svc.Svc,
				err_susp.Message())
			//Ignore and continue... (do not return tran header)
		} else {
			ac.TpLogDebug("Transaction suspended [%s]", tidrsp)
			w.Header().Set(TX_RSP_HDR, tidrsp)
		}
	}

	return err
}

/* vim: set ts=4 sw=4 et smartindent: */
