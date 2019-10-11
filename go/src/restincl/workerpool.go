/**
 * @brief Enduro/X XATMI session pool & call handlers
 *
 * @file workerpool.go
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
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
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

//Generate the headers for UBF mode and for EXT mode
//Return content type if available
func genRspHeaders(ac *atmi.ATMICtx, bufu *atmi.TypedUBF, w http.ResponseWriter,
	svc *ServiceMap) string {
	ret := ""
	// Parse and set response header Name/Value pairs
	if svc.Parseheaders {
		ac.TpLogInfo("Setting Response Headers")
		occs, _ := bufu.BOccur(ubftab.EX_IF_RSPHN)
		for occ := 0; occ < occs; occ++ {
			HdrName, err1 := bufu.BGetString(ubftab.EX_IF_RSPHN, occ)
			if nil != err1 {

				ac.TpLogError("Failed to get EX_IF_RSPHN[%d]", occ)
				continue
			}
			HdrValue, err2 := bufu.BGetString(ubftab.EX_IF_RSPHV, occ)
			if nil != err2 {

				ac.TpLogError("Failed to get EX_IF_RSPHV[%d]", occ)
			}

			if HdrName == "Content-Type" {
				ret = HdrValue
			} else {
				w.Header().Set(HdrName, HdrValue)
			}

		}

		// Parse and set response cookies
		if svc.Parsecookies {
			ck := http.Cookie{}
			occ := 0
			var e error
			//Print the buffer to stdout
			bufu.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")
			ac.TpLogInfo("Setting Response Cookies")
			if bufu.BPres(ubftab.EX_IF_RSPCN, occ) {
				CookieName, retName := bufu.BGetString(ubftab.EX_IF_RSPCN, occ)
				if nil != retName {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					ck.Name = CookieName
				}
				CookieValue, retValue := bufu.BGetString(ubftab.EX_IF_RSPCV, occ)
				if nil != retValue {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {

					if CookieValue != "" {
						ck.Value = CookieValue
					}
				}
				CookiePath, retPath := bufu.BGetString(ubftab.EX_IF_RSPCPATH, occ)
				if nil != retPath {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {

					if CookiePath != "" {
						ck.Path = CookiePath
					}
				}
				CookieDomain, retDomain := bufu.BGetString(ubftab.EX_IF_RSPCDOMAIN, occ)
				if nil != retDomain {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					if CookieDomain != "" {
						ck.Domain = CookieDomain
					}
				}
				CookieExpires, retExpires := bufu.BGetString(ubftab.EX_IF_RSPCEXPIRES, occ)
				if nil != retExpires {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					if CookieExpires != "" {
						ck.Expires, e = time.Parse(time.RFC1123, CookieExpires)
						if nil != e {
							ac.TpLog(atmi.LOG_ERROR,
								"Failed to parse Cookie Expire Time [%s]", CookieExpires)
						}
					}
				}
				CookieMaxAge, retMaxAge := bufu.BGetString(ubftab.EX_IF_RSPCMAXAGE, occ)
				if nil != retMaxAge {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					if CookieMaxAge != "" {
						ck.MaxAge, e = strconv.Atoi(CookieMaxAge)
						if nil != e {
							ac.TpLog(atmi.LOG_ERROR,
								"Failed to convert Cookie MaxAge [%s]", CookieMaxAge)
						}
					}
				}
				CookieSecure, retSecure := bufu.BGetString(ubftab.EX_IF_RSPCSECURE, occ)
				if nil != retSecure {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					if CookieSecure != "" {
						ck.Secure, e = strconv.ParseBool(CookieSecure)
						if nil != e {
							ac.TpLog(atmi.LOG_ERROR,
								"Failed to parse Cookie Secure [%s]", CookieSecure)
						}
					}
				}
				CookieHttpOnly, retHttpOnly := bufu.BGetString(ubftab.EX_IF_RSPCHTTPONLY, occ)
				if nil != retHttpOnly {
					ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
						"%d occ %d", ubftab.EX_IF_RSPCN, occ)
				} else {
					if CookieHttpOnly != "" {
						ck.HttpOnly, e = strconv.ParseBool(CookieHttpOnly)
						if nil != e {
							ac.TpLog(atmi.LOG_ERROR,
								"Failed to parse Cookie HttpOnly [%s]", CookieHttpOnly)
						}
					}
				}
			}
			http.SetCookie(w, &ck)
		}
	}

	return ret
}

//Generate response in the service configured way...
//@w	handler for writting response to
//if postSvc is set to true, that indicates that service was called and now
//response is being generated (i.e. messager convert on incoming was ok)
func genRsp(ac *atmi.ATMICtx, buf atmi.TypedBuffer, svc *ServiceMap,
	w http.ResponseWriter, atmiErr atmi.ATMIError, reqlogOpen bool, postSvc bool) {

	var rsp []byte
	var err atmi.ATMIError
	var netCode int = 200
	/*	application/json */
	rspType := "text/plain"
	// Header and Cookies fields to delete from buffer
	delFldList := []int{ubftab.EX_IF_REQHN,
		ubftab.EX_IF_REQHV,
		ubftab.EX_IF_REQCN,
		ubftab.EX_IF_REQCV,
		// Response filed for Headers/Cookies
		ubftab.EX_IF_RSPHN,
		ubftab.EX_IF_RSPHV,
		ubftab.EX_IF_RSPCN,
		ubftab.EX_IF_RSPCV,
		ubftab.EX_IF_RSPCPATH,
		ubftab.EX_IF_RSPCDOMAIN,
		ubftab.EX_IF_RSPCEXPIRES,
		ubftab.EX_IF_RSPCMAXAGE,
		ubftab.EX_IF_RSPCSECURE,
		ubftab.EX_IF_RSPCHTTPONLY}

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
	case CONV_EXT:

		bufu, ok := buf.(*atmi.TypedUBF)

		if !ok {
			ac.TpLogError("Invalid response buffer, not UBF!")
			w.WriteHeader(500)
			break
		}

		out_err := false
		was_error := false

		ac.TpLogInfo("err=%v", err)

		//OK we are at ext, execute the error filters, if any
		if !postSvc {
			//This is incoming error, run the incoming error handler
			runChain(ac, svc, buf, false, svc.Finerr_arr,
				"filter-incoming-error-opt(finerr)")
			was_error = true
		} else if nil == err || 0 == err.Code() {
			//Execute the outgoing chains...
			if errA := runChain(ac, svc, buf, true, svc.Foutman_arr,
				"filter-outgoing-mandatory(foutman)"); nil != errA {
				out_err = true
				was_error = true
			}

			if !was_error {
				runChain(ac, svc, buf, false, svc.Foutopt_arr,
					"filter-outgoing-optional(foutopt)")
			}
		} else {
			out_err = true
			was_error = true
		}

		//If we got outgoing error, call the service correspondingly..
		if out_err {
			runChain(ac, svc, buf, false, svc.Fouterr_arr,
				"filter-outgoing-error-opt(fouterr)")
		}

		//OK, check the status code
		if was_error || bufu.BPres(ubftab.EX_NETRCODE, 0) {
			//Expect the return code from error service
			//If have return code, then we can use buffer for response
			//If there is no return code, then we cannot use the return buffer
			//and we will fail with server error

			netCode, _ = bufu.BGetInt(ubftab.EX_NETRCODE, 0)

			if 0 == netCode {
				ac.TpLogError("Invalid EX_NETRCODE or not set => return http 500")
				w.WriteHeader(500)
				break
			}

		}

		//Load the body if any and headers
		//Process headers
		rspTypeHdr := genRspHeaders(ac, bufu, w, svc)

		if rspTypeHdr != "" {
			rspType = rspTypeHdr
		}

		//Load the body (if any..)
		if bufu.BPres(ubftab.EX_IF_RSPDATA, 0) {
			var errU atmi.UBFError

			rsp, errU = bufu.BGetByteArr(ubftab.EX_IF_RSPDATA, 0)

			if nil != errU {
				ac.TpLogError("Failed to get body: %s", errU.Error())
				w.WriteHeader(500)
				break
			}

		}

		if netCode != 200 {

			//Send headers first..
			w.Header().Set("Content-Type", rspType)
			w.WriteHeader(netCode)
		}

		//That's it

		break

	case CONV_JSON2UBF:
		rspType = "application/json"
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
					ac.TpLogError("Failed to set EX_IF_ECODE: %d/%s ",
						e1.Code(), e1.Message())

				}

				if e2 := bufu.BChg(ubftab.EX_IF_EMSG, 0, err.Message()); nil != e2 {
					ac.TpLogError("Failed to set EX_IF_EMSG: %d/%s ",
						e2.Code(), e2.Message())
				}
			}

			//Process headers
			genRspHeaders(ac, bufu, w, svc)

			// Delete Header and Cookie data from buffer (req&rsp)
			bufu.BDelete(delFldList)

			ret, err1 := bufu.TpUBFToJSON()

			if nil == err1 {
				//Generate the resposne buffer...
				rsp = []byte(ret)
			} else {

				if err.Code() == atmi.TPMINVAL {
					err = err1
				}

				if svc.Errors_int == ERRORS_JSON2UBF {
					rsp = []byte(fmt.Sprintf("{\"EX_IF_ECODE\":%d,\"EX_IF_EMSG\":\"%s\"}",
						err1.Code(), err1.Message()))
				}

			}
		}

		break
	case CONV_JSON2VIEW:
		rspType = "application/json"
		//Convert buffer back to JSON & send it back..
		//But we could append the buffer with error here...

		bufv, ok := buf.(*atmi.TypedVIEW)

		if svc.Asynccall && !svc.Asyncecho {

			ac.TpLogInfo("Async mode, no echo")
			if svc.Errors_int == ERRORS_JSON2VIEW {

				ac.TpLogInfo("Generating configured rsp...")
				rsp = VIEWGenDefaultResponse(ac, svc, atmiErr)

				ac.TpLogInfo("Got response: [%v]", rsp)
			}
		} else if !ok || nil == buf { //Nil case goes here too
			ac.TpLogError("Failed to cast TypedBuffer to TypedVIEW!")
			//Create empty buffer for generating response...

			if err.Code() == atmi.TPMINVAL {
				err = atmi.NewCustomATMIError(atmi.TPESYSTEM, "Invalid buffer")
			}

			if svc.Errors_int == ERRORS_JSON2VIEW {
				rsp = VIEWGenDefaultResponse(ac, svc, atmiErr)
			}
		} else {

			rsp = nil
			errorFallback := false
			//If error is set, then try to install error
			if svc.Errors_int == ERRORS_JSON2VIEW && err.Code() != atmi.TPMINVAL {

				ac.TpLogInfo("Setting JSON2VIEW buffer error codes to: %d/%s",
					err.Code(), err.Message())
				//In case of success try to install only if
				//ret on success is set..

				if svc.Errfmt_view_rsp_first {
					//Response directly with response object

					rsp = VIEWGenDefaultResponse(ac, svc, atmiErr)

				} else {
					//Install response in view object

					itype := ""
					subtype := ""

					if _, errA := ac.TpTypes(bufv.Buf, &itype, &subtype); nil != errA {
						ac.TpLogError("Failed to get buffer infos: %s", errA.Error())
						errorFallback = true
					}

					ac.TpLogInfo("Got buffer infos: %s/%s [%s]",
						itype, subtype, bufv.BVName())

					if errA := VIEWInstallError(bufv, subtype, svc.Errfmt_view_code,
						err.Code(), svc.Errfmt_view_msg, err.Message()); nil != errA {
						ac.TpLogWarn("Failed to set view resposne fields, "+
							"falling back to view rsp object: %s", errA.Error())
						errorFallback = true
					}

				}

				//Fallback to view object
				if errorFallback {
					if "" != svc.Errfmt_view_rsp {
						ac.TpLogInfo("Error fallback enabled -Respond with rsp view")

						rsp = VIEWGenDefaultResponse(ac, svc, atmiErr)

					} else {
						ac.TpLogInfo("svc: %s: Error fallback, but no rsp view "+
							"defined-> drop rsp: %d/%s",
							svc.Svc, err.Code(), err.Message())
						ac.UserLog("svc: %s: Error fallback, but no rsp view "+
							"defined-> drop rsp: %d/%s",
							svc.Svc, err.Code(), err.Message())
						rsp = []byte("{}")
					}
				}
			} else if svc.Errors_int == ERRORS_JSON2VIEW &&
				err.Code() == atmi.TPMINVAL && svc.Errfmt_view_onsucc {
				//Try to install error code on success
				//If no response fields found, ignore error and return the response

				itype := ""
				subtype := ""

				if _, errA := ac.TpTypes(bufv.Buf, &itype, &subtype); nil != errA {
					ac.TpLogError("Failed to get buffer infos: %s", errA.Error())
					errorFallback = true
				}

				ac.TpLogInfo("Got buffer infos: %s/%s", itype, subtype)

				if errA := VIEWInstallError(bufv, subtype, svc.Errfmt_view_code,
					atmi.TPMINVAL, svc.Errfmt_view_msg, "SUCCEED"); nil != errA {
					ac.TpLogWarn("Failed to set view resposne fields, "+
						"falling back to view rsp object: %s", errA.Error())
				}
			}

			//Generate response if one is not set already
			if nil == rsp {
				ret, err1 := bufv.TpVIEWToJSON(svc.View_flags)

				if nil == err1 {
					//Generate the resposne buffer...
					rsp = []byte(ret)
				} else {

					if err.Code() == atmi.TPMINVAL {
						err = err1
					}
					//In case of failure, just return empty json (should be assumed
					//as timeout from on other end
					if svc.Errors_int == ERRORS_JSON2VIEW {
						rsp = []byte("{}")
					}
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
		rspType = "application/json"
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
	w.Header().Set("Content-Type", rspType)
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

		//TODO: We need to send headers first...!

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
	ac.TpLogDump(atmi.LOG_DEBUG, "Sending response back", rsp, len(rsp))
	w.Header().Set("Content-Length", strconv.Itoa(len(rsp)))

	w.Write(rsp)
}

//Common function parsing http request headers
func parseHeaders(ac *atmi.ATMICtx, svc *ServiceMap, req *http.Request,
	bufu *atmi.TypedUBF) atmi.UBFError {

	// Add header data to UBF fields
	if svc.Parseheaders {
		for k, v := range req.Header {
			ac.TpLogDebug("Header field %s, Value %+v", k, v)
			hv := fmt.Sprintf("%s", v)
			if errU := bufu.BAdd(ubftab.EX_IF_REQHN, k); nil != errU {
				return errU
			}

			if errU := bufu.BAdd(ubftab.EX_IF_REQHV, hv); nil != errU {
				return errU
			}
			// Add Cookies data to UBF
		}

		if svc.Parsecookies {
			for _, cookie := range req.Cookies() {
				// Incoming request have Name and Value
				ac.TpLogDebug("cookie.Name=[%s]", cookie.Name)
				ac.TpLogDebug("cookie.Value=[%s]", cookie.Value)
				if errU := bufu.BAdd(ubftab.EX_IF_REQCN, cookie.Name); nil != errU {
					return errU
				}
				if errU := bufu.BAdd(ubftab.EX_IF_REQCV, cookie.Value); nil != errU {
					return errU
				}
			}
		}
	}

	return nil
}

//Common func for parsing query parameters and loading them into UBF buffer
func parseQuery(ac *atmi.ATMICtx, svc *ServiceMap, req *http.Request,
	bufu *atmi.TypedUBF) atmi.UBFError {

	m := req.URL.Query()

	for k, v := range m {

		//Query args are arrays by it self...
		for _, vv := range v {

			if errU := bufu.BAdd(ubftab.EX_IF_REQQUERYN, k); nil != errU {
				ac.TpLogError("Failed to add EX_IF_REQQUERYN: %s", errU.Error())
				return errU
			}

			if errU := bufu.BAdd(ubftab.EX_IF_REQQUERYV, vv); nil != errU {
				ac.TpLogError("Failed to add EX_IF_REQQUERYV: %s", errU.Error())
				return errU
			}
		}
	}

	return nil
}

//Run the list of services (if any). If opt set to false, then each service call is
//mandatory. Failed, error will returned immediately
//ac is Atmi Context, svc is currently mapped service definition, buf is associated
//converted buffer, svclist is comma seperated service name list.
//Listdbg is debug string for the invocation
func runChain(ac *atmi.ATMICtx, svc *ServiceMap, buf atmi.TypedBuffer, mand bool,
	svclist []string, listdbg string) atmi.ATMIError {

	if len(svclist) == 0 {
		return nil
	}

	for _, svc := range svclist {

		ac.TpLogInfo("%s: About to invoke: [%s]", listdbg, svc)

		_, err := ac.TpCall(svc, buf, 0)

		if nil != err {

			if !mand {
				ac.TpLogWarn("%s: Failed to call [%s] service: %s - optional, continue",
					listdbg, svc, err.Message())
			} else {
				ac.TpLogError("%s: Failed to call [%s] service: %s - fail",
					listdbg, svc, err.Message())
				return err
			}
		}

	}

	return nil
}

//Request handler
//@param ac	ATMI Context
//@param w	Response writer (as usual)
//@param req	Request message (as usual)
func handleMessage(ac *atmi.ATMICtx, svc *ServiceMap, w http.ResponseWriter,
	req *http.Request) int {

	var flags int64 = 0
	var buf atmi.TypedBuffer
	var err atmi.ATMIError
	reqlogOpen := false
	ac.TpLog(atmi.LOG_DEBUG, "Got URL [%s], caller: %s", req.URL, req.RemoteAddr)

	if "" != svc.Svc || svc.Echo {

		var body []byte
		if !svc.Parseform {

			body, _ = ioutil.ReadAll(req.Body)
			ac.TpLogDebug("Requesting service [%s] buffer [%s]",
				svc.Svc, string(body))
		}

		//Prepare outgoing buffer...
		switch svc.Conv_int {
		case CONV_EXT:
			//Convert JSON 2 UBF...
			//Bug #200, use max buffer size
			bufu, err1 := ac.NewUBF(atmi.ATMIMsgSizeMax())

			if nil != err1 {
				ac.TpLogError("failed to alloca ubf buffer %d:[%s]",
					err1.Code(), err1.Message())

				genRsp(ac, nil, svc, w, err1, false, false)
				return atmi.FAIL
			}

			//Load the body
			if !svc.Parseform {

				if errU := bufu.BChg(ubftab.EX_IF_REQDATA, 0, body); errU != nil {

					ac.TpLogError("Failed to set body data in EX_IF_REQDATA %d:[%s]",
						errU.Code(), errU.Message())

					errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
						fmt.Sprintf("Failed to set body data in EX_IF_REQDATA %d:[%s]",
							errU.Code(), errU.Message()))

					genRsp(ac, nil, svc, w, errA, false, false)
					return atmi.FAIL
				}
			}

			//Load the headers if requested...
			if errU := parseHeaders(ac, svc, req, bufu); nil != errU {
				ac.TpLogError("Failed to parse/load headers")

				errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
					fmt.Sprintf("Failed to parse headers %d:[%s]",
						errU.Code(), errU.Message()))

				genRsp(ac, nil, svc, w, errA, false, false)
				return atmi.FAIL
			}

			//Load the request URL
			if errU := bufu.BAdd(ubftab.EX_IF_URL, req.URL.Path); nil != errU {

				errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
					fmt.Sprintf("Failed to set EX_IF_URL %d:[%s]",
						errU.Code(), errU.Message()))

				ac.TpLogError("Failed to set request URL")
				genRsp(ac, nil, svc, w, errA, false, false)
				return atmi.FAIL
			}

			//Load request paramters
			if errU := parseQuery(ac, svc, req, bufu); nil != errU {
				ac.TpLogError("Failed to parse/load URL Query params")

				errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
					fmt.Sprintf("Failed to parse Query params %d:[%s]",
						errU.Code(), errU.Message()))

				genRsp(ac, nil, svc, w, errA, false, false)
				return atmi.FAIL
			}

			//Parse for in requested..
			if svc.Parseform {
				if errF := req.ParseForm(); errF != nil {
					ac.TpLogError("Failed to parse form: [%s]", errF.Error())
				} else {
					ac.TpLogInfo("Form parsed OK")
					//Load the arguments in the buffer..
					for k, v := range req.Form {

						ac.TpLogDebug("form field name=[%s]", k)
						str := strings.Join(v, ";")
						ac.TpLogDebug("form field value=[%s]", str)

						if errU := bufu.BAdd(ubftab.EX_IF_REQFORMN, k); nil != errU {

							ac.TpLogError("Failed to add form field: [%s]: %s",
								k, errU.Error())

							errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
								fmt.Sprintf("Failed to add EX_IF_REQFORMN %d:[%s]",
									errU.Code(), errU.Message()))

							genRsp(ac, nil, svc, w, errA, false, false)
							return atmi.FAIL
						}

						if errU := bufu.BAdd(ubftab.EX_IF_REQFORMV, str); nil != errU {

							ac.TpLogError("Failed to add form field value: [%s]: %s",
								k, errU.Error())

							errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
								fmt.Sprintf("Failed to add EX_IF_REQFORMV %d:[%s]",
									errU.Code(), errU.Message()))

							genRsp(ac, nil, svc, w, errA, false, false)
							return atmi.FAIL
						}
					} //for form value
				} //If form parse ok
			}

			buf = bufu
			break
		case CONV_JSON2UBF:
			//Convert JSON 2 UBF...
			//Bug #200, use max buffer size
			bufu, err1 := ac.NewUBF(atmi.ATMIMsgSizeMax())

			if nil != err1 {
				ac.TpLogError("failed to alloca ubf buffer %d:[%s]\n",
					err1.Code(), err1.Message())

				genRsp(ac, nil, svc, w, err1, false, false)
				return atmi.FAIL
			}

			ac.TpLogDebug("Converting to UBF: [%s]", body)

			if errU := parseHeaders(ac, svc, req, bufu); nil != errU {
				ac.TpLogError("Failed to parse/load headers")

				errA := atmi.NewCustomATMIError(atmi.TPESYSTEM,
					fmt.Sprintf("Failed to parse headers %d:[%s]",
						errU.Code(), errU.Message()))

				genRsp(ac, nil, svc, w, errA, false, false)
				return atmi.FAIL
			}

			if err1 := bufu.TpJSONToUBF(string(body)); err1 != nil {
				ac.TpLogError("Failed to conver from JSON to UBF %d:[%s]\n",
					err1.Code(), err1.Message())

				ac.TpLogError("Failed req: [%s]", string(body))

				genRsp(ac, nil, svc, w, err1, false, false)
				return atmi.FAIL
			}
			if svc.Format == "r" || svc.Format == "regexp" {
				if id, err := ac.BFldId(svc.UrlField); err == nil && id != 0 {
					ac.TpLogInfo("Setting field: [%d] with value [%s]", id, req.URL.Path)
					bufu.BAdd(id, req.URL.Path)
				} else {
					ac.TpLogInfo("Setting field: [EX_IF_URL] with value [%s]", req.URL.Path)
					bufu.BAdd(ubftab.EX_IF_URL, req.URL.Path)
				}
			}

			buf = bufu
			break
		case CONV_JSON2VIEW:
			//Conver JSON to View

			ac.TpLogDebug("Converting to VIEW: [%s]", body)

			bufv, err1 := ac.TpJSONToVIEW(string(body))

			if err1 != nil {
				ac.TpLogError("Failed to convert JSON to VIEW: %d:[%s]\n",
					err1.Code(), err1.Message())

				ac.TpLogError("Failed req: [%s]", string(body))

				genRsp(ac, nil, svc, w, err1, false, false)
				return atmi.FAIL
			}

			buf = bufv
			break
		case CONV_TEXT:
			//Use request buffer as string

			bufs, err1 := ac.NewString(string(body))

			if nil != err1 {
				ac.TpLogError("failed to alloc string/text buffer %d:[%s]\n",
					err1.Code(), err1.Message())

				genRsp(ac, nil, svc, w, err1, false, false)
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
				genRsp(ac, nil, svc, w, err1, false, false)
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
				genRsp(ac, nil, svc, w, err1, false, false)
				return atmi.FAIL
			}

			if svc.Format == "r" || svc.Format == "regexp" {
				var jsonObj interface{}
				if err := json.Unmarshal([]byte(bufj.GetJSON()), &jsonObj); err != nil {
					ac.TpLogError("Failed to unmarshal JSON: %v", err.Error())
					return atmi.FAIL
				}
				obj := jsonObj.(map[string]interface{})

				if svc.UrlField != "" {
					obj[svc.UrlField] = req.URL.Path
				} else {
					obj["EX_IF_URL"] = req.URL.Path
				}

				if barr, err2 := json.Marshal(obj); err2 == nil {
					if err = bufj.SetJSON(barr); err != nil {
						ac.TpLogError("Failed to set JSON: %v", err.Error())
						return atmi.FAIL
					}
				} else {
					ac.TpLogError("Failed to marshal JSON: %v", err2.Error())
					return atmi.FAIL
				}

			}

			buf = bufj

			break
		}

		if err != nil {
			ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())

			genRsp(ac, buf, svc, w, err, false, false)
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

		//Perform incoming filters...
		//If input filters fails, then generate response immediately...
		err = nil

		if len(svc.Finman_arr) > 0 {
			err = runChain(ac, svc, buf, true, svc.Finman_arr,
				"filter-incoming-mandatory(finman)")

			//Run optional chain, if any..
			if nil == err {

				runChain(ac, svc, buf, false, svc.Finopt_arr,
					"filter-incoming-optional(finopt)")
			}
		}

		if nil != err {
			genRsp(ac, buf, svc, w, err, reqlogOpen, false)
		} else if svc.Echo {
			//Do not send service, just echo buffer back
			genRsp(ac, buf, svc, w, err, reqlogOpen, true)
		} else if svc.Asynccall {
			_, err := ac.TpACall(svc.Svc, buf, flags|atmi.TPNOREPLY)
			genRsp(ac, buf, svc, w, err, reqlogOpen, true)
		} else {
			_, err := ac.TpCall(svc.Svc, buf, flags)

			genRsp(ac, buf, svc, w, err, reqlogOpen, true)
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

/* vim: set ts=4 sw=4 et smartindent: */
