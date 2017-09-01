/*
** Enduro/X XATMI session pool & call handlers
**
** @file workerpool.go
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
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

/*

So we will have a pool of goroutines which will wait on channel. This is needed
for doing less works with ATMI initialization and uninit.

Scheme will be following

- there will be array of goroutines, number is set in M_workwers
- there will be number same number of channels M_waitjobchan[M_workers]
- there will be M_freechan which will identify the free channel number (
when worker will complete its work, it will submit it's number to this channel)

So handler on new message will do <-M_freechan and then send message to -> M_waitjobchan[M_workers]
Workes will wait on <-M_waitjobchan[M_workers], when complete they will do Nr -> M_freechan

*/

var M_freechan chan int //List of free channels submitted by wokers

var M_ctxs []*atmi.ATMICtx //List of contexts

//Generate response in the service configured way...
//@w	handler for writting response to
func genRsp(ac *atmi.ATMICtx, buf atmi.TypedBuffer, svc *ServiceMap,
	w http.ResponseWriter, atmiErr atmi.ATMIError, reqlogOpen bool) {

	var rsp []byte
	var err atmi.ATMIError
	/*	application/json */
	rspType := "text/plain"

	//Remove request logfile if was open and not needed in rsp.
	if reqlogOpen && svc.Noreqfilersp {
		ac.TpLogDebug("Removing request file")
		u, _ := ac.CastToUBF(buf.GetBuf())
		_ = u.BDel(ubftab.EX_NREQLOGFILE, 0)
	}

	//Have a common error handler
	if nil == atmiErr {
		err = atmi.NewCustomATMIError(atmi.TPMINVAL, "SUCCEED")
	} else {
		err = atmiErr
	}

	//Generate response accordingly...
	ac.TpLogDebug("Conv %d errors %d", svc.Conv_int, svc.Errors_int)

	switch svc.Conv_int {
	case CONV_JSON2UBF:
		//rsp_type = "text/json"
		//Convert buffer back to JSON & send it back..
		//But we could append the buffer with error here...

		bufu, ok := buf.(*atmi.TypedUBF)

		if svc.Asynccall && !svc.Asyncecho {
			if svc.Errors_int == ERRORS_JSON2UBF {
				rsp = []byte(fmt.Sprintf("{\"EX_IF_ECODE\":%d,\"EX_IF_EMSG\":\"%s\"}",
					err.Code(), err.Message()))
			}
		} else if !ok {
			ac.TpLogError("Failed to cast TypedBuffer to TypedUBF!")
			//Create empty buffer for generating response...

			if err.Code() == atmi.TPMINVAL {
				err = atmi.NewCustomATMIError(atmi.TPESYSTEM, "Invalid buffer")
			}

			if svc.Errors_int == ERRORS_JSON2UBF {
				rsp = []byte(fmt.Sprintf("{\"EX_IF_ECODE\":%d,\"EX_IF_EMSG\":\"%s\"}",
					err.Code(), err.Message()))
			}
		} else {

			if svc.Errors_int == ERRORS_JSON2UBF {
				ac.TpLogInfo("Setting JSON2UBF buffer error codes to: %d/%s",
					err.Code(), err.Message())

				if e1 := bufu.BChg(ubftab.EX_IF_ECODE, 0, err.Code()); nil != e1 {
					ac.TpLogError("Failed to set EX_IF_ECODE: %s/%s ",
						e1.Code(), e1.Message())

				}

				if e2 := bufu.BChg(ubftab.EX_IF_EMSG, 0, err.Message()); nil != e2 {
					ac.TpLogError("Failed to set EX_IF_EMSG: %s/%s ",
						e2.Code(), e2.Message())
				}
			}

			ret, err1 := bufu.TpUBFToJSON()

			if nil == err1 {
				//Generate the resposne buffer...
				rsp = []byte(ret)
			} else {

				if err.Code() == atmi.TPMINVAL {
					err = err1
				}

				if svc.Errors_int == ERRORS_JSON2UBF {
					rsp = []byte(fmt.Sprintf("{EX_IF_ECODE:%d, EX_IF_EMSG:\"%s\"}",
						err1.Code(), err1.Message()))
				}

			}
		}

		break
	case CONV_TEXT: //This is string buffer...
		//If there is no error & it is sync call, then just plot
		//a buffer back
		/* if !svc.Asynccall && atmi.TPMINVAL == err.Code() {
		Lets reply back with same buffer...
		*/
		if !svc.Asynccall || svc.Asyncecho {

			bufs, ok := buf.(*atmi.TypedString)

			if !ok {
				ac.TpLogError("Failed to cast buffer to TypedString")

				if err.Code() == atmi.TPMINVAL {
					err = atmi.NewCustomATMIError(atmi.TPEINVAL,
						"Failed to cast buffer to TypedString")
				}
			} else {
				//Set the bytes to string we got
				rsp = []byte(bufs.GetString())
			}

		}

		break
	case CONV_RAW: //This is carray..
		rspType = "application/octet-stream"

		/*
			if !svc.Asynccall && atmi.TPMINVAL == err.Code() {
			Lets reply back with same buffer.
		*/
		if !svc.Asynccall || svc.Asyncecho {
			bufs, ok := buf.(*atmi.TypedCarray)

			if !ok {
				ac.TpLogError("Failed to cast buffer to TypedCarray")

				if err.Code() == atmi.TPMINVAL {
					err = atmi.NewCustomATMIError(atmi.TPEINVAL,
						"Failed to cast buffer to TypedCarray")
				}
			} else {
				//raw/Set the bytes to string we got
				rsp = bufs.GetBytes()
			}
		}
		break
	case CONV_JSON:
		rspType = "text/json"
		/*		if !svc.Asynccall && atmi.TPMINVAL == err.Code() { why?
				Lets reply back with same incoming buffer...
		*/
		if !svc.Asynccall || svc.Asyncecho {
			bufs, ok := buf.(*atmi.TypedJSON)

			if !ok {
				ac.TpLogError("Failed to cast buffer to TypedJSON")
				err = atmi.NewCustomATMIError(atmi.TPEINVAL,
					"Failed to cast buffer to ypedJSON")
			} else {
				//Set the bytes to string we got
				rsp = []byte(bufs.GetJSON())
			}
		}
		break
	}

	//OK Now if all ok, there is stuff in buffer (from JSONUBF) it will
	//be there in any case, thus we do not handle that

	switch svc.Errors_int {
	case ERRORS_HTTP:
		var lookup map[string]int
		//Map the resposne codes
		if len(svc.Errors_fmt_http_map) > 0 {
			lookup = svc.Errors_fmt_http_map
		} else {
			lookup = M_defaults.Errors_fmt_http_map
		}

		estr := strconv.Itoa(err.Code())

		httpCode := 500

		if 0 != lookup[estr] {
			httpCode = lookup[estr]
		} else {
			httpCode = lookup["*"]
		}

		//Generate error response and pop out of the funcion
		if 200 != httpCode {
			ac.TpLogWarn("Mapped response: tp %d -> http %d",
				err.Code(), httpCode)
			w.WriteHeader(httpCode)
		}

		break
	case ERRORS_JSON:
		//Send JSON error block, togher with buffer, if buffer empty
		//Send simple json...

		if atmi.TPMINVAL == err.Code() && !svc.Errfmt_json_onsucc && !svc.Asyncecho {
			break //Do no generate on success.
		}
		strrsp := string(rsp)

		match, _ := regexp.MatchString("^\\s*{\\s*}\\s*$", strrsp)

		if i := strings.LastIndex(strrsp, "}"); i > -1 {
			//Add the trailing response code in JSON block
			substring := strrsp[0:i]

			errs := ""

			if match {
				ac.TpLogInfo("Empty JSON rsp")
				errs = fmt.Sprintf("%s,%s}",
					fmt.Sprintf(svc.Errfmt_json_code, err.Code()),
					fmt.Sprintf(svc.Errfmt_json_msg, err.Message()))
			} else {

				ac.TpLogInfo("Have some data in JSON rsp")
				errs = fmt.Sprintf(",%s,%s}",
					fmt.Sprintf(svc.Errfmt_json_code, err.Code()),
					fmt.Sprintf(svc.Errfmt_json_msg, err.Message()))
			}

			ac.TpLogWarn("Error code generated: [%s]", errs)
			strrsp = substring + errs
			ac.TpLogDebug("JSON Response generated: [%s]", strrsp)
		} else {
			//rsp_type = "text/json"
			//Send plaint json
			strrsp = fmt.Sprintf("{%s,%s}",
				fmt.Sprintf(svc.Errfmt_json_code, err.Code()),
				fmt.Sprintf(svc.Errfmt_json_msg, err.Message()))
			ac.TpLogDebug("JSON Response generated (2): [%s]", strrsp)
		}

		rsp = []byte(strrsp)
		break
	case ERRORS_TEXT:
		//Send plain text error if have one.
		//rsp_type = "text/json"
		//Send plaint json
		if (svc.Asynccall && !svc.Asyncecho) || atmi.TPMINVAL != err.Code() {
			strrsp := fmt.Sprintf(svc.Errfmt_text, err.Code(), err.Message())
			ac.TpLogDebug("TEXT Response generated (2): [%s]", strrsp)
			rsp = []byte(strrsp)
		}

		break
	}

	//Send response back
	ac.TpLogDebug("Returning context type: %s, len: %d", rspType, len(rsp))
	ac.TpLogDump(atmi.LOG_INFO, "Sending response back", rsp, len(rsp))
	w.Header().Set("Content-Length", strconv.Itoa(len(rsp)))
	w.Write(rsp)
}

//Request handler
//@param ac	ATMI Context
//@param w	Response writer (as usual)
//@param req	Request message (as usual)
func handleMessage(ac *atmi.ATMICtx, svc *ServiceMap, w http.ResponseWriter, req *http.Request) int {

	var flags int64 = 0
	var buf atmi.TypedBuffer
	var err atmi.ATMIError
	reqlogOpen := false
	ac.TpLog(atmi.LOG_DEBUG, "Got URL [%s]", req.URL)

	if "" != svc.Svc || svc.Echo {

		body, _ := ioutil.ReadAll(req.Body)

		ac.TpLogDebug("Requesting service [%s] buffer [%s]", svc.Svc, string(body))

		//Prepare outgoing buffer...
		switch svc.Conv_int {
		case CONV_JSON2UBF:
			//Convert JSON 2 UBF...
			//Bug #200, use max buffer size
			bufu, err1 := ac.NewUBF(atmi.ATMI_MSG_MAX_SIZE)

			if nil != err1 {
				ac.TpLogError("failed to alloca ubf buffer %d:[%s]\n",
					err1.Code(), err1.Message())

				genRsp(ac, nil, svc, w, err1, false)
				return atmi.FAIL
			}

			ac.TpLogDebug("Converting to UBF: [%s]", body)

			if err1 := bufu.TpJSONToUBF(string(body)); err1 != nil {
				ac.TpLogError("Failed to conver buffer to JSON %d:[%s]\n",
					err1.Code(), err1.Message())

				ac.TpLogError("Failed req: [%s]", string(body))

				genRsp(ac, nil, svc, w, err1, false)
				return atmi.FAIL
			}

			buf = bufu
			break
		case CONV_TEXT:
			//Use request buffer as string

			bufs, err1 := ac.NewString(string(body))

			if nil != err1 {
				ac.TpLogError("failed to alloc string/text buffer %d:[%s]\n",
					err1.Code(), err1.Message())

				genRsp(ac, nil, svc, w, err1, false)
				return atmi.FAIL
			}

			buf = bufs

			break
		case CONV_RAW:
			//Use request buffer as binary

			bufc, err1 := ac.NewCarray(body)

			if nil != err1 {
				ac.TpLogError("failed to alloc carray/bin buffer %d:[%s]\n",
					err1.Code(), err1.Message())
				genRsp(ac, nil, svc, w, err1, false)
				return atmi.FAIL
			}

			buf = bufc

			break
		case CONV_JSON:
			//Use request buffer as JSON

			bufj, err1 := ac.NewJSON(body)

			if nil != err1 {
				ac.TpLogError("failed to alloc carray/bin buffer %d:[%s]\n",
					err1.Code(), err1.Message())
				genRsp(ac, nil, svc, w, err1, false)
				return atmi.FAIL
			}

			buf = bufj

			break
		}

		if err != nil {
			ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())

			genRsp(ac, buf, svc, w, err, false)
			return atmi.FAIL
		}

		if svc.Notime {
			ac.TpLogWarn("No timeout flag for service call")
			flags |= atmi.TPNOTIME
		}

		//Open then PAN file if needed & buffer type is UBF
		var btype string

		if "" != svc.Reqlogsvc {
			if _, err := buf.GetBuf().TpTypes(&btype, nil); err == nil {
				ac.TpLogDebug("UBF buffer - requesting logfile from %s", svc.Reqlogsvc)

				if err := ac.TpLogSetReqFile(buf.GetBuf(), "", svc.Reqlogsvc); err == nil {
					reqlogOpen = true
				}
			}
		}

		//Do not send service, just echo buffer back
		if svc.Echo {
			genRsp(ac, buf, svc, w, err, reqlogOpen)
		} else if svc.Asynccall {
			_, err := ac.TpACall(svc.Svc, buf.GetBuf(), flags|atmi.TPNOREPLY)
			genRsp(ac, buf, svc, w, err, reqlogOpen)
		} else {
			_, err := ac.TpCall(svc.Svc, buf.GetBuf(), flags)
			genRsp(ac, buf, svc, w, err, reqlogOpen)
		}
	}

	if reqlogOpen {
		ac.TpLogCloseReqFile()
	}

	return atmi.SUCCEED
}

//Initialise channels and work pools
func initPool(ac *atmi.ATMICtx) error {

	M_freechan = make(chan int, M_workers)

	for i := 0; i < M_workers; i++ {

		ctx, err := atmi.NewATMICtx()

		if err != nil {
			ac.TpLogError("Failed to create context: %s", err.Message())
			return err
		}

		M_ctxs = append(M_ctxs, ctx)

		//Submit the free ATMI context
		M_freechan <- i
	}
	return nil
}
