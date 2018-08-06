/**
 * @brief Enduro/X Outgoing http REST handler (HTTP client, XATMI server)
 *
 * @file restoutsv.go
 */
/* -----------------------------------------------------------------------------
 * Enduro/X Middleware Platform for Distributed Transaction Processing
 * Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
 * Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
 * This software is released under one of the following licenses:
 * GPL or Mavimax's license for commercial use.
 * -----------------------------------------------------------------------------
 * GPL license:
 * 
 * This program is free software; you can redistribute it and/or modify it under
 * the terms of the GNU General Public License as published by the Free Software
 * Foundation; either version 3 of the License, or (at your option) any later
 * version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with
 * this program; if not, write to the Free Software Foundation, Inc., 59 Temple
 * Place, Suite 330, Boston, MA 02111-1307 USA
 *
 * -----------------------------------------------------------------------------
 * A commercial use license is available from Mavimax, Ltd
 * contact@mavimax.com
 * -----------------------------------------------------------------------------
 */
package main

// Request types supported:
// - json (TypedJSON, TypedUBF)
// - plain text (TypedString)
// - binary (TypedCarray)

//Hmm we might need to put in channels a free ATMI contexts..
import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

/*
#include <signal.h>
*/
import "C"

const (
	progsection = "@restout"
)

const (
	UNSET   = -1
	FALSE   = 0
	TRUE    = 1
	SUCCEED = atmi.SUCCEED
	FAIL    = atmi.FAIL
)

//Error handling type
const (
	ERRORS_HTTP = 1 //Return error code in http
	ERRORS_TEXT = 2 //Return error as formatted text (from config)
	ERRORS_JSON = 3 //Contact the json fields to main respons block.
	//Return the error code as UBF response (usable only in case if CONV_JSON2UBF used)
	ERRORS_JSON2UBF  = 4
	ERRORS_JSON2VIEW = 5
)

//Conversion types resolved
const (
	CONV_JSON2UBF  = 1
	CONV_TEXT      = 2
	CONV_JSON      = 3
	CONV_RAW       = 4
	CONV_JSON2VIEW = 5
)

//Defaults
const (
	ECHO_DEFAULT               = false
	ECHO_CONV_DEFAULT          = "json2ubf"
	ECHO_DATA_DEFAULT          = "{\"EX_DATA_STR\":\"Echo test\"}"
	ECHO_TIME_DEFAULT          = 5
	ECHO_MAX_FAIL_DEFAULT      = 2
	ECHO_MIN_OK_DEFAULT        = 3
	ERRORS_DEFAULT             = ERRORS_JSON2UBF
	TIMEOUT_DEFAULT            = 60
	ERRFMT_JSON_MSG_DEFAULT    = "error_message"
	ERRFMT_JSON_CODE_DEFAULT   = "error_code"
	ERRFMT_JSON_ONSUCC_DEFAULT = true /* generate success message in JSON */
	ERRFMT_TEXT_DEFAULT        = "^([0-9]+):(.*)$"
	WORKERS_DEFAULT            = 10 /* Number of worker processes */
	NOREQFILE_DEFAULT          = true
)

//We will have most of the settings as defaults
//And then these settings we can override with
type ServiceMap struct {
	Svc         string
	UrlBase     string `json:"urlbase"`
	Url         string `json:"url"`
	SSLInsecure bool   `json:"sslinsecure"`

	Timeout int `json:"timeout"`

	Errors string `json:"errors"`
	//Above converted to consntant
	Errors_int int

	//Format for error to parse
	//for 'text'
	Errfmt_text string `json:"errfmt_text"`
	//Have a hanlder too for compiled regex
	Errfmt_text_Regexp *regexp.Regexp

	//JSON fields
	//for 'json'
	Errfmt_json_msg  string `json:"errfmt_json_msg"`
	Errfmt_json_code string `json:"errfmt_json_code"`
	//Should fields be present on success
	//If missing, then assume response is ok
	Errfmt_json_onsucc bool `json:"errfmt_json_onsucc"`

	//VIEW fields
	Errfmt_view_msg  string `json:"errfmt_view_msg"`
	Errfmt_view_code string `json:"errfmt_view_code"`

	//Error mapping between <http><Enduro/X, currently 0 or 11)
	Errors_fmt_http_map_str string `json:"errors_fmt_http_map"`
	Errors_fmt_http_map     map[string]*int

	//Should we parse the response (and fill the reply buffer)
	//in case if we got the error
	ParseOnError bool `json:"parseonerror"`

	//This is echo tester service
	Echo        bool   `json:"echo"`
	EchoTime    int    `json:"echo_time"`
	EchoMaxFail int    `json:"echo_max_fail"`
	EchoMinOK   int    `json:"echo_min_ok"`
	EchoConv    string `json:"echo_conv"`
	echoConvInt int
	EchoData    string `json:"echo_data"`

	//Install in response non null fields only
	View_notnull bool  `json:"view_notnull"`
	View_flags   int64 //Flags used for VIEW2JSON

	//Counters:
	echoFails      int  //Number failed echos
	echoSchedUnAdv bool //Should we schedule advertise

	echoSucceeds int  //Number of ok echos
	echoSchedAdv bool //Should we schedule advertise

	echoIsAdvertised bool //Are we advertised?

	DependsOn string `json:"depends_on"`

	//Wait for shutdown message
	shutdown chan bool //This is if we get shutdown messages

	//Preparsed buffers
	echoUBF    *atmi.TypedUBF
	echoVIEW   *atmi.TypedVIEW //View support, instantiated echo buffer
	echoCARRAY *atmi.TypedCarray

	//Dependies...
	Dependies []*ServiceMap
}

var Mservices map[string]*ServiceMap

//map the atmi error code (numbers + *) to some http error
//We shall provide default mappings.

var Mdefaults ServiceMap
var Mworkers int
var Mac *atmi.ATMICtx //Mainly shared for logging....

var Mmonitors int //Number of monitoring threads, to wait for shutdown.

var MmonitorsShut chan bool //Channel to wait for shutdown reply msgs

//Lock the advertise operations
//So that we do not get advertise twice
//and so on...
var MadvertiseLock = &sync.Mutex{}

var MScanTime = 1 //In seconds

//Conversion types
var Mconvs = map[string]int{

	"json2ubf":  CONV_JSON2UBF,
	"text":      CONV_TEXT,
	"json":      CONV_JSON,
	"raw":       CONV_RAW,
	"json2view": CONV_JSON2VIEW,
}

//Remap the error from string to int constant
//for better performance...
func remapErrors(ac *atmi.ATMICtx, svc *ServiceMap) error {

	switch svc.Errors {
	case "http":
		svc.Errors_int = ERRORS_HTTP
		break
	case "json":
		svc.Errors_int = ERRORS_JSON
		break
	case "json2ubf":
		svc.Errors_int = ERRORS_JSON2UBF
		break
	case "text":
		svc.Errors_int = ERRORS_TEXT
		break
	case "json2view":
		svc.Errors_int = ERRORS_JSON2VIEW
		break
	default:
		return fmt.Errorf("Unsupported error type [%s]", svc.Errors)
	}

	//Try to compile the text errors
	var err error
	ac.TpLogInfo("Compiling text error parser: [%s]", svc.Errfmt_text)
	if svc.Errfmt_text_Regexp, err = regexp.Compile(svc.Errfmt_text); err != nil {

		ac.TpLogError("Failed to comiple errfmt_text [%s] for svc [%s]: %s",
			svc.Errfmt_text, svc.Svc, err.Error())

		return err

	}

	return nil
}

//Map the Http errors to ATMI errors
//Format: <http_err_1>:<atmi_err_1>,<http_err_N>:<atmi_err_N>,<*>:<atmi_err_N>
//* - means any other unmapped HTTP error
//@param svc	Service map
func parseHTTPErrorMap(ac *atmi.ATMICtx, svc *ServiceMap) error {

	svc.Errors_fmt_http_map = make(map[string]*int)
	ac.TpLogDebug("Splitting error mapping string [%s]",
		svc.Errors_fmt_http_map_str)

	parsed := regexp.MustCompile(", *").Split(svc.Errors_fmt_http_map_str, -1)

	for index, element := range parsed {
		ac.TpLogDebug("Got pair [%s] at %d", element, index)

		pair := regexp.MustCompile(": *").Split(element, -1)

		pairLen := len(pair)

		if pairLen < 2 || pairLen > 2 {
			ac.TpLogError("Invalid http error pair: [%s] "+
				"parsed into %d elms", element, pairLen)

			return fmt.Errorf("Invalid http error pair: [%s] "+
				"parsed into %d elms", element, pairLen)
		}

		number, err := strconv.ParseInt(pair[1], 10, 0)

		if err != nil {
			ac.TpLogError("Failed to parse http error code %s (%s)",
				pair[1], err)
			return fmt.Errorf("Failed to parse http error code %s (%s)",
				pair[1], err)
		}

		//Add to hash
		n := int(number)
		svc.Errors_fmt_http_map[pair[0]] = &n
	}

	if nil == svc.Errors_fmt_http_map["*"] {
		return fmt.Errorf("Missing wildcard \"*\" in error config string!")
	}

	return nil
}

//Print the summary of the service after init
func printSvcSummary(ac *atmi.ATMICtx, svc *ServiceMap) {
	ac.TpLogWarn("Service: [%s], Url: [%s], Errors:%d (%s), Echo %t",
		svc.Svc,
		svc.Url,
		svc.Errors_int,
		svc.Errors,
		svc.Echo)
}

//Un-init function
func appinit(ctx *atmi.ATMICtx) int {
	//runtime.LockOSThread()

	Mservices = make(map[string]*ServiceMap)

	//Setup default configuration
	Mdefaults.Errors_int = ERRORS_DEFAULT
	Mdefaults.Echo = ECHO_DEFAULT
	Mdefaults.EchoConv = ECHO_CONV_DEFAULT
	Mdefaults.EchoData = ECHO_DATA_DEFAULT
	Mdefaults.EchoTime = ECHO_TIME_DEFAULT
	Mdefaults.EchoMaxFail = ECHO_MAX_FAIL_DEFAULT
	Mdefaults.EchoMinOK = ECHO_MIN_OK_DEFAULT
	Mdefaults.Errfmt_json_msg = ERRFMT_JSON_MSG_DEFAULT
	Mdefaults.Errfmt_json_code = ERRFMT_JSON_CODE_DEFAULT
	Mdefaults.Errfmt_json_onsucc = ERRFMT_JSON_ONSUCC_DEFAULT
	Mdefaults.Errfmt_text = ERRFMT_TEXT_DEFAULT

	Mworkers = WORKERS_DEFAULT

	//Get the configuration

	buf, err := ctx.NewUBF(16 * 1024)
	if nil != err {
		ctx.TpLog(atmi.LOG_ERROR, "Failed to allocate buffer: [%s]", err.Error())
		return FAIL
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", progsection,
		os.Getenv("NDRX_CCTAG")))

	if _, err := ctx.TpCall("@CCONF", buf, 0); nil != err {
		ctx.TpLog(atmi.LOG_ERROR, "ATMI Error %d:[%s]\n", err.Code(),
			err.Message())
		return FAIL
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	//Set the parameters (ip/port/services)

	occs, _ := buf.BOccur(u.EX_CC_KEY)
	// Load in the config...
	for occ := 0; occ < occs; occ++ {
		ctx.TpLog(atmi.LOG_DEBUG, "occ %d", occ)
		fldName, err := buf.BGetString(u.EX_CC_KEY, occ)

		if nil != err {
			ctx.TpLog(atmi.LOG_ERROR, "Failed to get field "+
				"%d occ %d", u.EX_CC_KEY, occ)
			return FAIL
		}

		ctx.TpLog(atmi.LOG_DEBUG, "Got config field [%s]", fldName)

		switch fldName {
		case "debug":
			//Set debug configuration string
			debug, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ctx.TpLogDebug("Got [%s] = [%s] ", fldName, debug)
			if err := ctx.TpLogConfig((atmi.LOG_FACILITY_NDRX | atmi.LOG_FACILITY_UBF | atmi.LOG_FACILITY_TP),
				-1, debug, "ROUT", ""); nil != err {
				ctx.TpLogError("Invalid debug config [%s] %d:[%s]",
					debug, err.Code(), err.Message())
				return FAIL
			}

			break
		case "workers":
			Mworkers, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			break
		case "gencore":
			gencore, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if TRUE == gencore {
				//Process signals by default handlers
				ctx.TpLogInfo("gencore=1 - SIGSEG signal will be " +
					"processed by default OS handler")
				// Have some core dumps...
				C.signal(11, nil)
			}
			break
		case "scan_time":
			MScanTime, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ctx.TpLogDebug("Got [%s] = [%d] ", fldName, MScanTime)
			break
		case "defaults":
			//Override the defaults
			jsonDefault, _ := buf.BGetByteArr(u.EX_CC_VALUE, occ)

			jerr := json.Unmarshal(jsonDefault, &Mdefaults)
			if jerr != nil {
				ctx.TpLog(atmi.LOG_ERROR,
					"Failed to parse defaults: %s", jerr.Error())
				return FAIL
			}

			if Mdefaults.Errors_fmt_http_map_str != "" {
				if jerr := parseHTTPErrorMap(ctx, &Mdefaults); jerr != nil {
					return FAIL
				}
			}

			if Mdefaults.Echo {
				Mdefaults.echoConvInt = Mconvs[Mdefaults.EchoConv]
				if Mdefaults.echoConvInt == 0 {
					ctx.TpLogError("Invalid conv: %s",
						Mdefaults.EchoConv)
					return FAIL
				}
			}

			//printSvcSummary(ctx, &Mdefaults)

			break
		default:
			//Assign the defaults

			//Load services...

			match, _ := regexp.MatchString("^service\\s*.*$", fldName)

			if match {

				re := regexp.MustCompile("^service\\s*(.*)$")
				matchSvc := re.FindStringSubmatch(fldName)

				cfgVal, _ := buf.BGetString(u.EX_CC_VALUE, occ)

				ctx.TpLogInfo("Got service route config [%s]=[%s]",
					matchSvc[1], cfgVal)

				tmp := Mdefaults

				//Override the stuff from current config
				tmp.Svc = matchSvc[1]

				//err := json.Unmarshal(cfgVal, &tmp)
				decoder := json.NewDecoder(strings.NewReader(cfgVal))
				//conf := Config{}
				err := decoder.Decode(&tmp)

				if err != nil {
					ctx.TpLog(atmi.LOG_ERROR,
						"Failed to parse config key %s: %s",
						fldName, err)
					return FAIL
				}

				ctx.TpLogDebug("Got config block [%s] -> Service [%s]",
					fldName, tmp.Svc)

				//Parse http errors for
				if tmp.Errors_fmt_http_map_str != "" {
					if jerr := parseHTTPErrorMap(ctx, &tmp); jerr != nil {
						return FAIL
					}
				}

				//Remap the error codes and regexps...
				if err := remapErrors(ctx, &tmp); err != nil {
					ctx.TpLogError("remapErrors failed: %s",
						err.Error())
					return FAIL
				}

				ctx.TpLogInfo("Errors mapped to: %d", tmp.Errors_int)

				//Add to HTTP listener
				//We should add service to advertise list...
				//And list if echo is enabled & succeeed
				//or if echo not set, then auto advertise all
				//http.HandleFunc(fldName, dispatchRequest)

				if strings.HasPrefix(tmp.Url, "/") {
					//This is partial URL, so use base
					tmp.Url = tmp.UrlBase + tmp.Url

					ctx.TpLogInfo("Have / prefix => building"+
						" URL, got: [%s]",
						tmp.Url)
				} else {
					ctx.TpLogInfo("No / prefix => assuming"+
						"full url [%s]",
						tmp.Url)
				}

				if tmp.Echo {
					tmp.echoConvInt = Mconvs[tmp.EchoConv]
					if tmp.echoConvInt == 0 {
						ctx.TpLogError("Invalid conv: %s",
							tmp.EchoConv)
						return FAIL
					}

					if tmp.EchoMaxFail < 1 {
						ctx.TpLogError("Invalid 'echo_max_fail' "+
							"setting: %d, must be >=1",
							tmp.EchoMaxFail)
					}

					if tmp.EchoMinOK < 1 {
						ctx.TpLogError("Invalid 'echo_min_ok' "+
							"setting: %d, must be >=1",
							tmp.EchoMaxFail)
					}

					if errA := tmp.PreparseEchoBuffers(ctx); nil != errA {
						ctx.TpLogError("Failed to parse "+
							"echo buffers: %s",
							errA.Error())
						return FAIL
					}

					//Make async chan
					tmp.shutdown = make(chan bool, 2)
					Mmonitors++
				}

				//Test service if it is view

				if err := VIEWValidateService(ctx, &tmp); nil != err {
					ctx.TpLogError("Failed to validate view settings: %s",
						err.Error())
					return FAIL
				}

				Mservices[matchSvc[1]] = &tmp

				printSvcSummary(ctx, &tmp)
			}
			break
		}
	}

	ctx.TpLogInfo("Number of monitor services: %d", Mmonitors)
	MmonitorsShut = make(chan bool, Mmonitors)

	//Add the default erorr mappings
	if Mdefaults.Errors_fmt_http_map_str == "" {

		//https://golang.org/src/net/http/status.go
		Mdefaults.Errors_fmt_http_map = make(map[string]*int)
		//Accepted
		tpeminval := atmi.TPMINVAL
		Mdefaults.Errors_fmt_http_map[strconv.Itoa(http.StatusOK)] = &tpeminval

		tpetime := atmi.TPETIME
		Mdefaults.Errors_fmt_http_map[strconv.Itoa(http.StatusGatewayTimeout)] = &tpetime

		tpnoent := atmi.TPENOENT
		Mdefaults.Errors_fmt_http_map[strconv.Itoa(http.StatusNotFound)] = &tpnoent

		//Anything other goes to server error.
		genfail := atmi.TPESVCFAIL
		Mdefaults.Errors_fmt_http_map["*"] = &genfail

	}

	ctx.TpLogInfo("About to init woker pool, number of workers: %d", Mworkers)

	MoutXPool.nrWorkers = Mworkers
	if err := initPool(ctx, &MoutXPool); nil != err {
		return FAIL
	}

	haveEcho := false
	//Advertise services which are not dependent
	for _, v := range Mservices {

		if v.Echo {
			haveEcho = true
		}

		if v.DependsOn == "" || v.Echo {
			//Advertize service
			if errA := v.Advertise(ctx); nil != errA {
				return FAIL
			}
			v.echoIsAdvertised = true
		} else if v.DependsOn != "" {

			echoSvc := Mservices[v.DependsOn]

			if nil != echoSvc {
				ctx.TpLogInfo("Adding [%s] to [%s] as dependie",
					v.Svc, echoSvc.Svc)
				echoSvc.Dependies = append(echoSvc.Dependies, v)
			} else {
				ctx.TpLogError("Invalid echo service "+
					"('depends_on') [%s] for [%s]",
					v.DependsOn, echoSvc.Svc)
				return FAIL
			}
		}
	}

	if haveEcho {
		ctx.TpLogWarn("Echo services present - installing periodic callback")
		if err := ctx.TpExtAddPeriodCB(MScanTime, Periodic); err != nil {
			ctx.TpLogError("Advertise failed %d: %s",
				err.Code(), err.Message())
			return FAIL
		}

		//Boot the monitor threads..
		for _, v := range Mservices {
			if v.Echo {
				go v.Monitor()
			}
		}
	}

	return SUCCEED
}

//RESTOUT service - generic entry point
//@param ac ATMI Context
//@param svc Service call information
func RESTOUT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			/* ac.TpContinue() - No need for this
			 * Or it have nothing todo.
			 * as operation  must be last.
			 */
			ac.TpContinue()
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()
	ac.TpLogInfo("RESTOUT got request...")

	//Pack the request data to pass to thread
	ctxData, err := ac.TpSrvGetCtxData()
	if nil != err {
		ac.TpLogError("Failed to get context data - dropping request",
			err.Code(), err.Message())
		ret = FAIL
		return
	}

	ac.TpLogInfo("Waiting for free XATMI out object")
	nr := getFreeXChan(ac, &MoutXPool)
	ac.TpLogInfo("Got XATMI out object")

	go XATMIDispatchCall(&MoutXPool, nr, ctxData, &svc.Data, svc.Cd, svc.Name)

	//runtime.GC()

	return
}

//Un-init & Terminate the application
func unInit(ac *atmi.ATMICtx) {

	//dispatch to monitors & wait for them to complete the shutdown
	for _, v := range Mservices {

		//Send shutdown to svc
		if v.Echo {
			ac.TpLogInfo("Shutting down monitor: [%s]", v.Svc)
			v.shutdown <- true
		}
	}

	for i := 0; i < Mmonitors; i++ {

		ac.TpLogInfo("Waiting monitor %d to complete", i)
		_ = <-MmonitorsShut
	}

	deInitPoll(ac, &MoutXPool)

	ac.TpLogInfo("Shutdown ok")

}

//Executable main entry point
func main() {
	//Have some context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s", err)
		os.Exit(atmi.FAIL)
	} else {
		//Run as server
		if err = ac.TpRun(appinit, unInit); nil != err {
			ac.TpLogError("Exit with failure")
			os.Exit(atmi.FAIL)
		} else {
			ac.TpLogInfo("Exit with success")
			os.Exit(atmi.SUCCEED)
		}
	}
}
