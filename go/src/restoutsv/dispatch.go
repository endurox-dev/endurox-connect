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
	"time"

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

	//Locate our service defintion
	svc := Mservices[svcName]

	defer func() {

		if SUCCEED == ret {
			ac.TpLogInfo("Dispatch returns SUCCEED")
			ac.TpReturn(atmi.TPSUCCESS, 0, buf, 0)
		} else {
			ac.TpLogWarn("Dispatch returns FAIL")
			ac.TpReturn(atmi.TPFAIL, 0, buf, 0)
		}

		//Put back the channel
		//!!!! MUST Be last, otherwise while tpreturn completes
		//Other thread can take this object, and that makes race condition +
		//Corrpuption !!!!
		pool.freechan <- nr
	}()

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
		ac.TpLogInfo("UBF buffer, len %d - converting to JSON & sending req", datalen)

		bufu, errA := ac.CastToUBF(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to UBF: %s", errA.Error())
			ret = FAIL
			return
		}

		json, errA := bufu.TpUBFToJSON()

		if nil == errA {
			//Generate the resposne buffer...
			content_to_send = []byte(json)
		} else {

			ac.TpLogError("Failed to cast UBF to JSON: %s", errA.Error())
			ret = FAIL
			return
		}

		break
	case "STRING":
		content_type = "text/plain"
		ac.TpLogInfo("STRING buffer, len %d", datalen)

		bufs, errA := ac.CastToString(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to STRING: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = []byte(bufs.GetString())

		break
	case "JSON":
		content_type = "application/json"
		ac.TpLogInfo("JSON buffer, len %d", datalen)

		bufj, errA := ac.CastToJSON(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to JSON: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = bufj.GetJSON()

		break
	case "CARRAY":
		content_type = "application/octet-stream"
		ac.TpLogInfo("CARRAY buffer, len %d", datalen)

		bufc, errA := ac.CastToCarray(buf)
		if errA != nil {
			ac.TpLogError("Failed to cast to CARRAY: %s", errA.Error())
			ret = FAIL
			return
		}

		content_to_send = bufc.GetBytes()

		break
	}

	ac.TpLogInfo("Sending POST request to: [%s]", svc.Url)

	ac.TpLogDump(atmi.LOG_DEBUG, "Data To send", content_to_send, len(content_to_send))
	req, err := http.NewRequest("POST", svc.Url, bytes.NewBuffer(content_to_send))

	//req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", content_type)

	//TODO: How do we get back time-out?
	//And can we report back to caller that it was timeout ATMI way?
	var client = &http.Client{
		Timeout: time.Second * time.Duration(svc.Timeout),
	}

	resp, err := client.Do(req)

	if err != nil {

		if err, ok := err.(net.Error); ok && err.Timeout() {
			//TODO: Respond with TPSOFTTIMEOUT
		} else {
			//TOOD: Assume other error
		}
	}

	defer resp.Body.Close()

	ac.TpLogInfo("response Status: %s", resp.Status)

	body, _ := ioutil.ReadAll(resp.Body)

	ac.TpLogDump(atmi.LOG_DEBUG, "Got response back", body, len(body))

	//If we are nont handling in http way and http is bad
	//then return fail...

	//Process the resposne status first
	switch svc.Errors_int {
	case ERRORS_HTTP:

		break
	case ERRORS_JSON:

		break
	case ERRORS_TEXT:

		break
	}

}
