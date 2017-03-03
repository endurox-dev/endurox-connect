/*
** Service "object" routines
**
** @file service.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2015, ATR Baltic, SIA. All Rights Reserved.
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
** A commercial use license is available from ATR Baltic, SIA
** contact@atrbaltic.com
** -----------------------------------------------------------------------------
 */
package main

import (
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"time"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

//Dispatch service call
//@param pool     XATMI context pool
//@param nr     Number our object in pool
//@param ctxData        call context data
//@param buf            ATMI buffer with call data
//@param cd             Call descriptor
func XATMIDispatchCall(pool *XATMIPool, nr int, ctxData *atmi.TPSRVCTXDATA,
	buf *atmi.ATMIBuf, cd int, svcName string) {

	ret := SUCCEED
	ac := pool.ctxs[nr]
	buftype := ""
	var retFlags int64 = 0
	returnATMIErrorCode := atmi.TPMINVAL
	/* The error codes sent from network */
	netCode := atmi.TPMINVAL
	netMessage := ""

	//Locate our service defintion
	svc := Mservices[svcName]

	//List all buffers here..
	var bufu *atmi.TypedUBF
	var bufuRsp *atmi.TypedUBF
	var bufj *atmi.TypedJSON
	var bufs *atmi.TypedString
	var bufc *atmi.TypedCarray

	retBuf := buf

	bufu_rsp_parsed := false
	var errG error

	defer func() {

		if SUCCEED == ret {
			ac.TpLogInfo("Dispatch returns SUCCEED")
			ac.TpReturn(atmi.TPSUCCESS, 0, retBuf, retFlags)
		} else {
			ac.TpLogWarn("Dispatch returns FAIL")
			ac.TpReturn(atmi.TPFAIL, 0, retBuf, retFlags)
		}

		//Put back the channel
		//!!!! MUST Be last, otherwise while tpreturn completes
		//Other thread can take this object, and that makes race condition +
		//Corrpuption !!!!
		pool.freechan <- nr
	}()

	ac.TpLogWarn("Dispatching: [%s] -> %p", svcName, svc)

	if nil == svc {
		ac.TpLogError("Invalid service name [%s] - cannot resolve",
			svcName)
		ret = FAIL
		return
	}

	ac.TpLogInfo("Reallocating the incoming buffer for storing the RSP")

	if errA := buf.TpRealloc(atmi.ATMI_MSG_MAX_SIZE); nil != errA {
		ac.TpLogError("Failed to realloc buffer to: %s",
			atmi.ATMI_MSG_MAX_SIZE)
		ret = FAIL
		return
	}

	//Cast the buffer to target format
	datalen, errA := ac.TpTypes(buf, &buftype, nil)

	if nil != errA {
		ac.TpLogError("Invalid buffer format received: %s", errA.Error())
		ret = FAIL
		return
	}

	//Currently empty one
	var content_to_send []byte
	content_type := ""

	switch buftype {
	case "UBF", "UBF32", "FML", "FML32":
		content_type = "application/json"
		ac.TpLogInfo("UBF buffer, len %d - converting to JSON & sending req",
			datalen)

		bufu, errA = ac.CastToUBF(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to UBF: %s", errA.Error())
			ret = FAIL
			return

		}
		json, errA := bufu.TpUBFToJSON()

		if nil == errA {
			ac.TpLogDebug("Got json to send: [%s]", json)
			//Set content to send
			content_to_send = []byte(json)
		} else {

			ac.TpLogError("Failed to cast UBF to JSON: %s", errA.Error())
			ret = FAIL
			return
		}

		if svc.Errors_int != ERRORS_HTTP && svc.Errors_int != ERRORS_JSON2UBF {

			ac.TpLogError("Invalid configuration! Sending UBF buffer "+
				"with non 'http' or 'json2ubf' buffer handling methods. "+
				" Current method: %s", svc.Errors)

			ac.UserLog("Service [%s] configuration error! Processing "+
				"buffer UBF, but errors marked as [%s]. "+
				"Must be 'json2ubf' or 'http'. Check field 'errors' "+
				"in service config block", svc.Errors)
			ret = FAIL
			return
		}

		break
	case "STRING":
		content_type = "text/plain"
		ac.TpLogInfo("STRING buffer, len %d", datalen)

		bufs, errA = ac.CastToString(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to STRING: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = []byte(bufs.GetString())

		if svc.Errors_int != ERRORS_HTTP &&
			svc.Errors_int != ERRORS_TEXT &&
			svc.Errors_int != ERRORS_JSON {
			ac.TpLogError("Invalid configuration! Sending STRING buffer "+
				"with non 'text', 'json', 'http' error handling methods. "+
				" Current method: %s", svc.Errors)

			ac.UserLog("Service [%s] configuration error! Processing "+
				"buffer STRING, but errors marked as [%s]. "+
				"Must be text', 'json', 'http'. Check field 'errors' "+
				"in service config block", svc.Errors)
			ret = FAIL
			return
		}

		break
	case "JSON":
		content_type = "application/json"
		ac.TpLogInfo("JSON buffer, len %d", datalen)

		bufj, errA = ac.CastToJSON(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to JSON: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = bufj.GetJSON()

		if svc.Errors_int != ERRORS_HTTP &&
			svc.Errors_int != ERRORS_TEXT &&
			svc.Errors_int != ERRORS_JSON {
			ac.TpLogError("Invalid configuration! Sending JSON buffer "+
				"with non 'text', 'json', 'http' error handling methods. "+
				" Current method: %s", svc.Errors)

			ac.UserLog("Service [%s] configuration error! Processing "+
				"buffer JSON, but errors marked as [%s]. "+
				"Must be text', 'json', 'http'. Check field 'errors' "+
				"in service config block", svc.Errors)
			ret = FAIL
			return
		}

		break
	case "CARRAY":
		content_type = "application/octet-stream"
		ac.TpLogInfo("CARRAY buffer, len %d", datalen)

		bufc, errA = ac.CastToCarray(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to CARRAY: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = bufc.GetBytes()

		if svc.Errors_int != ERRORS_HTTP &&
			svc.Errors_int != ERRORS_TEXT &&
			svc.Errors_int != ERRORS_JSON {
			ac.TpLogError("Invalid configuration! Sending CARRAY buffer "+
				"with non 'text', 'json', 'http' error handling methods. "+
				" Current method: %s", svc.Errors)

			ac.UserLog("Service [%s] configuration error! Processing "+
				"buffer CARRAY, but errors marked as [%s]. "+
				"Must be text', 'json', 'http'. Check field 'errors' "+
				"in service config block", svc.Errors)
			ret = FAIL
			return
		}

		break
	}

	ac.TpLogInfo("Sending POST request to: [%s]", svc.Url)

	ac.TpLogDump(atmi.LOG_DEBUG, "Data To send", content_to_send, len(content_to_send))
	req, err := http.NewRequest("POST", svc.Url, bytes.NewBuffer(content_to_send))

	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", content_type)

	var client = &http.Client{
		Timeout: time.Second * time.Duration(svc.Timeout),
	}

	resp, err := client.Do(req)

	if err != nil {

		ac.TpLogError("Got error: %s", err.Error())

		if err, ok := err.(net.Error); ok && err.Timeout() {
			//Respond with TPSOFTTIMEOUT
			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		} else {
			//Assume other error
			ret = FAIL
			return
		}
	}

	defer resp.Body.Close()

	ac.TpLogInfo("response Status: %s", resp.Status)

	body, errN := ioutil.ReadAll(resp.Body)

	if nil != errN {
		ac.TpLogError("Failed to read response body - dropping the "+
			"message and responding with tout: %s",
			errN)
		retFlags |= atmi.TPSOFTTIMEOUT
		ret = FAIL
		return
	}

	//If we are nont handling in http way and http is bad
	//then return fail...
	//Check the status now
	if svc.Errors_int != ERRORS_HTTP || resp.Status != strconv.Itoa(http.StatusOK) {

		ac.TpLogError("Expected http status %d, but got: %s - fail",
			http.StatusOK, resp.Status)
		ret = FAIL
		return
	}

	ac.TpLogDump(atmi.LOG_DEBUG, "Got response back", body, len(body))

	stringBody := string(body)

	ac.TpLogDebug("Got string body [%s]", stringBody)

	//Process the resposne status first
	ac.TpLogInfo("Checking status code...")
	switch svc.Errors_int {
	case ERRORS_HTTP:

		ac.TpLogInfo("Error conv mode is HTTP - looking up mapping table by %s",
			resp.Status)

		var lookup map[string]*int
		//Map the resposne codes
		if len(svc.Errors_fmt_http_map) > 0 {
			lookup = svc.Errors_fmt_http_map
		} else {
			lookup = Mdefaults.Errors_fmt_http_map
		}

		if nil != lookup[resp.Status] {

			returnATMIErrorCode = *lookup[resp.Status]
			ac.TpLogDebug("Exact match found, converted to: %s",
				returnATMIErrorCode)
		} else {
			//This is must have in buffer...
			returnATMIErrorCode = *lookup["*"]

			ac.TpLogDebug("Matched wildcard \"*\", converted to: %s",
				returnATMIErrorCode)
		}

		break
	case ERRORS_JSON:
		//Try to find our fields into which we are interested
		var jerr error
		netCode, netMessage, jerr = JSONErrorGet(ac, &stringBody,
			svc.Errfmt_json_code, svc.Errfmt_json_msg)

		if nil != jerr {
			ac.TpLogError("Failed to parse JSON message - dropping/ "+
				"gen timeout: %s", jerr.Error())

			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		//Test the error fields we got
		ac.TpLogWarn("Got response from net, code=%d, msg=[%s]",
			netCode, netMessage)

		if netMessage == "" && svc.Errfmt_json_onsucc {

			ac.TpLogError("Missing response message of [%s] in json "+
				"- Dropping/timing out", svc.Errfmt_json_msg)

			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		break
	case ERRORS_JSON2UBF:
		//Parse the buffer (will read all data right into buffer)
		//Allocate parse buffer - it will be new (because
		//We might not want to return data in error case...)
		//...Depending on flags
		ac.TpLogDebug("Converting to UBF: [%s]", body)

		if errA = bufu.TpJSONToUBF(stringBody); errA != nil {
			ac.TpLogError("Failed to conver buffer to JSON %d:[%s]",
				errA.Code(), errA.Message())

			ac.TpLogError("Failed req: [%s] - dropping msg/tout",
				stringBody)

			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		bufuRsp, errA = ac.NewUBF(atmi.ATMI_MSG_MAX_SIZE)

		if errA != nil {
			ac.TpLogError("Failed to alloc UBF %d:[%s] - drop/timeout",
				errA.Code(), errA.Message())

			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		bufuRsp.TpLogPrintUBF(atmi.LOG_DEBUG, "Got UBF response from net")

		bufu_rsp_parsed = true

		//JSON2UBF response fields are present always
		var errU atmi.UBFError

		netCode, errU = bufuRsp.BGetInt(u.EX_IF_ECODE, 0)

		if nil != errU {
			ac.TpLogError("Missing EX_IF_ECODE: %s - assume format "+
				"error - timeout", errU.Error())
			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		bufuRsp.BDel(u.EX_IF_ECODE, 0)

		netMessage, errU = bufuRsp.BGetString(u.EX_IF_EMSG, 0)

		if nil != errU {
			ac.TpLogError("Missing EX_IF_EMSG: %s - assume format "+
				"error - timeout", errU.Error())
			retFlags |= atmi.TPSOFTTIMEOUT
			ret = FAIL
			return
		}

		bufuRsp.BDel(u.EX_IF_EMSG, 0)

		break
	case ERRORS_TEXT:
		//Try to scanf the string
		erroCodeMsg := regexp.MustCompile(svc.Errfmt_text).FindStringSubmatch(stringBody)

		if len(erroCodeMsg) < 2 {
			ac.TpLogInfo("Error fields not found in text - assume succeed")
		} else {

			ac.TpLogInfo("Parsed response code [%s] message [%s]",
				erroCodeMsg[0], erroCodeMsg[1])

			netCode, errG = strconv.Atoi(erroCodeMsg[0])

			if nil != errG {
				//Assume that is ok? Invalid format, maybe data?
				//Well better fail with timeout...
				//The format must be exact!!

				ac.TpLogError("Invalid message code %d for text!!! "+
					"- Dropping/timeout",
					erroCodeMsg[0])

				retFlags |= atmi.TPSOFTTIMEOUT
				ret = FAIL
				return

			}
			netMessage = erroCodeMsg[1]
		}
		break
	}

	//Fix up error codes
	switch netCode {
	case atmi.TPMINVAL:
		ac.TpLogInfo("got SUCCEED")
		break
	case atmi.TPETIME:
		ac.TpLogInfo("got TPETIME")
		retFlags |= atmi.TPSOFTTIMEOUT
		ret = FAIL
		break
	case atmi.TPENOENT:
		ac.TpLogInfo("got TPEINVAL")
		retFlags |= atmi.TPSOFTNOENT
		ret = FAIL
		break
	case atmi.TPESVCERR:
		ac.TpLogInfo("got TPESVCERR")
		ret = FAIL
		break
	default:
		ac.TpLogInfo("defaulting to TPESVCERR")
		ret = FAIL
		netCode = atmi.TPESVCERR
		break
	}

	ac.TpLogInfo("Status after remap: code: %d message: [%s]",
		netCode, netMessage)

	//Should we parse content in case of error
	//Well we could try that if we have some data returned!
	//This should be done only in http error mapping case.
	//Parse the message (if ok to do so...)

	if !svc.ParseOnError && netCode != atmi.TPMINVAL {
		ac.TpLogWarn("Request failed and 'parseonerror' is false " +
			"- not changing buffer")
		return
	}

	if SUCCEED == ret || svc.ParseOnError {

		switch buftype {
		case "UBF", "UBF32", "FML", "FML32":

			//Parse response back from JSON
			if !bufu_rsp_parsed {
				ac.TpLogDebug("Converting to UBF: [%s]", body)

				if errA = bufu.TpJSONToUBF(stringBody); errA != nil {
					ac.TpLogError("Failed to conver rsp "+
						"buffer from JSON->UBF%d:[%s] - dropping",
						errA.Code(), errA.Message())

					ac.UserLog("Failed to conver rsp "+
						"buffer from JSON->UBF%d:[%s] - dropping",
						errA.Code(), errA.Message())

					retFlags |= atmi.TPSOFTTIMEOUT
					ret = FAIL
					return
				}
			} else {
				//Response is parsed and we will answer with it
				ac.TpLogInfo("Swapping UBF bufers...")

				//TODO: We need to set "auto" mark for the buffer
				buf = bufuRsp.GetBuf()
				ac.TpFree(bufu.GetBuf())
			}
			break
		case "STRING":
			//Load response into string buffer
			if errA = bufs.SetString(stringBody); errA != nil {
				ac.TpLogError("Failed to set rsp "+
					"STRING buffer %d:[%s] - dropping",
					errA.Code(), errA.Message())

				ac.UserLog("Failed to set rsp "+
					"STRING buffer %d:[%s] - dropping",
					errA.Code(), errA.Message())

				retFlags |= atmi.TPSOFTTIMEOUT
				ret = FAIL
				return
			}
			break
		case "JSON":
			//Load response into JSON buffer
			if errA = bufj.SetJSONText(stringBody); errA != nil {
				ac.TpLogError("Failed to set JSON rsp "+
					"buffer %d:[%s]", errA.Code(),
					errA.Message())

				ac.UserLog("Failed to set JSON rsp "+
					"buffer %d:[%s]", errA.Code(),
					errA.Message())

				retFlags |= atmi.TPSOFTTIMEOUT
				ret = FAIL
				return
			}

			break
		case "CARRAY":
			//Load response into CARRAY buffer
			if errA = bufc.SetBytes(body); errA != nil {
				ac.TpLogError("Failed to set CARRAY rsp "+
					"buffer %d:[%s]", errA.Code(),
					errA.Message())
				ac.UserLog("Failed to set CARRAY rsp "+
					"buffer %d:[%s]", errA.Code(),
					errA.Message())

				retFlags |= atmi.TPSOFTTIMEOUT
				ret = FAIL
				return
			}

			break
		}
	}
}
