/*
** Enduro/X Incoming http REST handler (HTTP server, XATMI client)
**
** @file restincl.go
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

// Request types supported:
// - json (TypedJSON, TypedUBF)
// - plain text (TypedString)
// - binary (TypedCarray)

//Hmm we might need to put in channels a free ATMI contexts..
import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

/*
#include <signal.h>
*/
import "C"

const (
	progsection = "RESTIN"
)

const (
	UNSET = -1
	FALSE = 0
	TRUE  = 1
)

//Error handling type
const (
	ERRORS_HTTP = 1 //Return error code in http
	ERRORS_TEXT = 2 //Return error as formatted text (from config)
	ERRORS_RAW  = 3 //Use the raw formatting (just another kind for text)
	ERRORS_JSON = 4 //Contact the json fields to main respons block.
	//Return the error code as UBF response (usable only in case if CONV_JSON2UBF used)
	ERRORS_JSON2UBF = 5
)

//Conversion types resolved
const (
	CONV_JSON2UBF = 1
	CONV_TEXT     = 2
	CONV_JSON     = 3
	CONV_RAW      = 4
)

//Defaults
const (
	ERRORS_DEFAULT             = ERRORS_JSON
	NOTIMEOUT_DEFAULT          = false /* we will use default timeout */
	CONV_DEFAULT               = "json2ubf"
	CONV_INT_DEFAULT           = CONV_JSON2UBF
	ERRFMT_JSON_MSG_DEFAULT    = "\"error_message\":\"%s\""
	ERRFMT_JSON_CODE_DEFAULT   = "\"error_code\":%d"
	ERRFMT_JSON_ONSUCC_DEFAULT = true /* generate success message in JSON */
	ERRFMT_TEXT_DEFAULT        = "%d: %s"
	ERRFMT_RAW_DEFAULT         = "%d: %s"
	ASYNCCALL_DEFAULT          = false
	WORKERS                    = 10 /* Number of worker processes */
)

//We will have most of the settings as defaults
//And then these settings we can override with
type ServiceMap struct {
	Svc    string `json:"svc"`
	Url    string
	Errors string `json:"errors"`
	//Above converted to consntant
	Errors_int       int
	Trantout         int64  `json:"trantout"` //If set, then using global transactions
	Notime           bool   `json:"notime"`
	Errfmt_text      string `json:"errfmt_text"`
	Errfmt_json_msg  string `json:"errfmt_json_msg"`
	Errfmt_json_code string `json:"errfmt_json_code"`
	//If set, then generate code/message for success too
	Errfmt_json_onsucc bool        `json:"errfmt_json_onsucc"`
	Errmap_http        string      `json:"errmap_http"`
	Errmap_http_hash   map[int]int //Lookup map for tp->http codes
	Asynccall          bool        `json:"async"` //use tpacall()
	Asyncecho	   bool		`json:"asyncecho"`//echo message in async mode
	Conv               string      `json:"conv"`  //Conv mode
	Conv_int           int         //Resolve conversion type
	//Request logging classify service
	Reqlogsvc string `json:"reqlogsvc"`
	//Error mapping Enduro/X error code (including * for all):http error code
	Errors_fmt_http_map_str string `json:"errors_fmt_http_map"`
	Errors_fmt_http_map     map[string]int
	Noreqfilersp            bool `json:noreqfilersp` //Do not sent request file in respones
	Echo                    bool `json:echo`         //Echo request buffer back
}

var M_port int = atmi.FAIL
var M_ip string
var M_url_map map[string]ServiceMap

//map the atmi error code (numbers + *) to some http error
//We shall provide default mappings.

var M_defaults ServiceMap

/* TLS Settings: */
var M_tls_enable int16 = FALSE
var M_tls_cert_file string
var M_tls_key_file string

//Conversion types
var M_convs = map[string]int{

	"json2ubf": CONV_JSON2UBF,
	"text":     CONV_TEXT,
	"json":     CONV_JSON,
	"raw":      CONV_RAW,
}

var M_workers int
var M_ac *atmi.ATMICtx //Mainly shared for logging....

//Remap the error from string to int constant
//for better performance...
func remapErrors(svc *ServiceMap) error {

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
	default:
		return fmt.Errorf("Unsupported error type [%s]", svc.Errors)
	}

	return nil
}

//Run the listener
func apprun(ac *atmi.ATMICtx) error {

	var err error
	//TODO: Some works needed for TLS...
	listenOn := fmt.Sprintf("%s:%d", M_ip, M_port)
	ac.TpLog(atmi.LOG_INFO, "About to listen on: (ip: %s, port: %d) %s",
		M_ip, M_port, listenOn)
	if TRUE == M_tls_enable {

		/* To prepare cert (self-signed) do following steps:
		 * - TODO
		 */
		err = http.ListenAndServeTLS(listenOn, M_tls_cert_file, M_tls_key_file, nil)
		ac.TpLog(atmi.LOG_ERROR, "ListenAndServeTLS() failed: %s", err)
	} else {
		err = http.ListenAndServe(listenOn, nil)
		ac.TpLog(atmi.LOG_ERROR, "ListenAndServe() failed: %s", err)
	}

	return err
}

//Init function, read config (with CCTAG)

func dispatchRequest(w http.ResponseWriter, req *http.Request) {
	M_ac.TpLog(atmi.LOG_DEBUG, "URL [%s] getting free goroutine", req.URL)

	nr := <-M_freechan

	svc := M_url_map[req.URL.String()]

	M_ac.TpLogInfo("Got free goroutine, nr %d", nr)

	handleMessage(M_ctxs[nr], &svc, w, req)

	M_ac.TpLogInfo("Request processing done %d... releasing the context", nr)

	M_freechan <- nr

}

//Map the ATMI Errors to Http errors
//Format: <atmi_err>:<http_err>,<*>:<http_err>
//* - means any other unmapped ATMI error
//@param svc	Service map
func parseHTTPErrorMap(ac *atmi.ATMICtx, svc *ServiceMap) error {

	svc.Errors_fmt_http_map = make(map[string]int)
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
		svc.Errors_fmt_http_map[pair[0]] = int(number)
	}

	return nil
}

//Print the summary of the service after init
func printSvcSummary(ac *atmi.ATMICtx, svc *ServiceMap) {
	ac.TpLogWarn("Service: %s, Url: %s, Async mode: %t, Log request svc: [%s], Errors:%d (%s), Async echo %t",
		svc.Svc,
		svc.Url,
		svc.Asynccall,
		svc.Reqlogsvc,
		svc.Errors_int,
		svc.Errors,
		svc.Asyncecho)
}

//Un-init function
func appinit(ac *atmi.ATMICtx) error {
	//runtime.LockOSThread()

	M_url_map = make(map[string]ServiceMap)

	//Setup default configuration
	M_defaults.Errors_int = ERRORS_DEFAULT
	M_defaults.Notime = NOTIMEOUT_DEFAULT
	M_defaults.Conv = CONV_DEFAULT
	M_defaults.Conv_int = CONV_INT_DEFAULT
	M_defaults.Errfmt_json_msg = ERRFMT_JSON_MSG_DEFAULT
	M_defaults.Errfmt_json_code = ERRFMT_JSON_CODE_DEFAULT
	M_defaults.Errfmt_json_onsucc = ERRFMT_JSON_ONSUCC_DEFAULT
	M_defaults.Errfmt_text = ERRFMT_TEXT_DEFAULT
	M_defaults.Asynccall = ASYNCCALL_DEFAULT

	M_workers = WORKERS

	if err := ac.TpInit(); err != nil {
		return errors.New(err.Error())
	}

	//Get the configuration

	buf, err := ac.NewUBF(16 * 1024)
	if nil != err {
		ac.TpLog(atmi.LOG_ERROR, "Failed to allocate buffer: [%s]", err.Error())
		return errors.New(err.Error())
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", progsection, os.Getenv("NDRX_CCTAG")))

	if _, err := ac.TpCall("@CCONF", buf, 0); nil != err {
		ac.TpLog(atmi.LOG_ERROR, "ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return errors.New(err.Error())
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	//Set the parameters (ip/port/services)

	occs, _ := buf.BOccur(u.EX_CC_KEY)
	// Load in the config...
	for occ := 0; occ < occs; occ++ {
		ac.TpLog(atmi.LOG_DEBUG, "occ %d", occ)
		fldName, err := buf.BGetString(u.EX_CC_KEY, occ)

		if nil != err {
			ac.TpLog(atmi.LOG_ERROR, "Failed to get field "+
				"%d occ %d", u.EX_CC_KEY, occ)
			return errors.New(err.Error())
		}

		ac.TpLog(atmi.LOG_DEBUG, "Got config field [%s]", fldName)

		switch fldName {

		case "workers":
			M_workers, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			break
		case "gencore":
			gencore, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if TRUE == gencore {
				//Process signals by default handlers
				ac.TpLogInfo("gencore=1 - SIGSEG signal will be " +
					"processed by default OS handler")
				// Have some core dumps...
				C.signal(11, nil)
			}
			break
		case "port":
			M_port, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			break
		case "ip":
			M_ip, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "tls_enable":
			M_tls_enable, _ = buf.BGetInt16(u.EX_CC_VALUE, occ)
			break
		case "tls_cert_file":
			M_tls_cert_file, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "tls_key_file":
			M_tls_key_file, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "defaults":
			//Override the defaults
			jsonDefault, _ := buf.BGetByteArr(u.EX_CC_VALUE, occ)

			jerr := json.Unmarshal(jsonDefault, &M_defaults)
			if jerr != nil {
				ac.TpLog(atmi.LOG_ERROR,
					fmt.Sprintf("Failed to parse defaults: %s", jerr))
				return jerr
			}

			if M_defaults.Errors_fmt_http_map_str != "" {
				if jerr := parseHTTPErrorMap(ac, &M_defaults); err != nil {
					return jerr
				}
			}

			remapErrors(&M_defaults)

			M_defaults.Conv_int = M_convs[M_defaults.Conv]
			if M_defaults.Conv_int == 0 {
				return fmt.Errorf("Invalid conv: %s", M_defaults.Conv)
			}

			printSvcSummary(ac, &M_defaults)

			break
		default:
			//Assign the defaults

			//Load routes...
			if strings.HasPrefix(fldName, "/") {
				cfgVal, _ := buf.BGetString(u.EX_CC_VALUE, occ)

				ac.TpLogInfo("Got route config [%s]", cfgVal)

				tmp := M_defaults

				//Override the stuff from current config

				//err := json.Unmarshal(cfgVal, &tmp)
				decoder := json.NewDecoder(strings.NewReader(cfgVal))
				//conf := Config{}
				err := decoder.Decode(&tmp)

				if err != nil {
					ac.TpLog(atmi.LOG_ERROR,
						fmt.Sprintf("Failed to parse config key %s: %s",
							fldName, err))
					return err
				}

				ac.TpLogDebug("Got route: URL [%s] -> Service [%s]",
					fldName, tmp.Svc)
				tmp.Url = fldName

				//Parse http errors for
				if tmp.Errors_fmt_http_map_str != "" {
					if jerr := parseHTTPErrorMap(ac, &tmp); err != nil {
						return jerr
					}
				}

				remapErrors(&tmp)
				//Map the conv
				tmp.Conv_int = M_convs[tmp.Conv]

				if tmp.Conv_int == 0 {
					return fmt.Errorf("Invalid conv: %s", tmp.Conv)
				}

				printSvcSummary(ac, &tmp)

				M_url_map[fldName] = tmp

				//Add to HTTP listener
				http.HandleFunc(fldName, dispatchRequest)

			}
			break
		}

	}

	if atmi.FAIL == M_port || "" == M_ip {
		ac.TpLog(atmi.LOG_ERROR, "Invalid config: missing ip (%s) or port (%d)",
			M_ip, M_port)
		return errors.New("Invalid config: missing ip or port")
	}

	//Check the TLS settings
	if TRUE == M_tls_enable && (M_tls_cert_file == "" || M_tls_key_file == "") {

		ac.TpLog(atmi.LOG_ERROR, "Invalid TLS settigns missing cert "+
			"(%s) or keyfile (%s) ", M_tls_cert_file, M_tls_key_file)

		return errors.New("Invalid config: missing ip or port")
	}

	//Add the default erorr mappings
	if M_defaults.Errors_fmt_http_map_str == "" {

		//https://golang.org/src/net/http/status.go
		M_defaults.Errors_fmt_http_map = make(map[string]int)
		//Accepted
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPMINVAL)] =
			http.StatusOK
		//Errors:
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEABORT)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEBADDESC)] =
			http.StatusBadRequest
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEBLOCK)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEINVAL)] =
			http.StatusBadRequest
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPELIMIT)] =
			http.StatusRequestEntityTooLarge
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPENOENT)] =
			http.StatusNotFound
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEOS)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEPERM)] =
			http.StatusUnauthorized
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEPROTO)] =
			http.StatusBadRequest
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPESVCERR)] =
			http.StatusBadGateway
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPESVCFAIL)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPESYSTEM)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPETIME)] =
			http.StatusGatewayTimeout
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPETRAN)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPERMERR)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEITYPE)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEOTYPE)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPERELEASE)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEHAZARD)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEHEURISTIC)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEEVENT)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEMATCH)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEDIAGNOSTIC)] =
			http.StatusInternalServerError
		M_defaults.Errors_fmt_http_map[strconv.Itoa(atmi.TPEMIB)] =
			http.StatusInternalServerError
		//Anything other goes to server error.
		M_defaults.Errors_fmt_http_map["*"] = http.StatusInternalServerError

	}

	ac.TpLogInfo("About to init woker pool, number of workers: %d", M_workers)

	initPool(ac)

	return nil
}

//Un-init & Terminate the application
func unInit(ac *atmi.ATMICtx, retCode int) {

	for i := 0; i < M_workers; i++ {
		nr := <-M_freechan

		ac.TpLogWarn("Terminating %d context", nr)
		M_ctxs[nr].TpTerm()
		M_ctxs[nr].FreeATMICtx()
	}

	ac.TpTerm()
	ac.FreeATMICtx()
	os.Exit(retCode)
}

//Handle the shutdown
func handleShutdown(ac *atmi.ATMICtx) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-signalChannel
		//Shutdown all contexts...
		ac.TpLogWarn("Got signal %d - shutting down all XATMI client contexts",
			sig)
		unInit(ac, atmi.SUCCEED)
	}()
}

//Service Main

func main() {

	var err atmi.ATMIError
	M_ac, err = atmi.NewATMICtx()

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to allocate cotnext %s!\n", err)
		os.Exit(atmi.FAIL)
	}

	if err := appinit(M_ac); nil != err {
                M_ac.TpLogError("Failed to init: %s", err)
		os.Exit(atmi.FAIL)
	}

	handleShutdown(M_ac)

	M_ac.TpLogWarn("REST Incoming init ok - serving...")

	if err := apprun(M_ac); nil != err {
		unInit(M_ac, atmi.FAIL)
	}

	unInit(M_ac, atmi.SUCCEED)
}
