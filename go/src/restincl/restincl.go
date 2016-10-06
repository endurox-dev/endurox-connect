package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"
	"strings"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	progsection = "RESTIN"
)

const (
	UNSET = -1
	FALSE = 0
	TRUE  = 1
)

var M_port int16 = atmi.FAIL
var M_ip string
var M_url_map map[string]string

/* TLS Settings: */
var M_tls_enable int16 = FALSE
var M_tls_cert_file string
var M_tls_key_file string

// Request handler
func handler(w http.ResponseWriter, req *http.Request) {
	runtime.LockOSThread()
	atmi.TpLog(atmi.LOG_DEBUG, "Got URL [%s]", req.URL)

	/* Send json to service */
	svc := M_url_map[req.URL.String()]
	if "" != svc {

		body, _ := ioutil.ReadAll(req.Body)

		atmi.TpLog(atmi.LOG_DEBUG, "Requesting service [%s] buffer [%s]", svc, body)

		buf, err := atmi.NewJSON(body)

		if err != nil {
			atmi.TpLog(atmi.LOG_ERROR, "ATMI Error %d:[%s]\n", err.Code(), err.Message())
			return
		}

		if _, err := atmi.TpCall(svc, buf.GetBuf(), 0); err != nil {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(err.Message()))
		} else {
			w.Header().Set("Content-Type", "text/json")
			w.Write([]byte(buf.GetJSONText()))
		}
	}

	/*
		w.Write([]byte("{status:\"ok\"}\n"))
	*/

}

//Run the listener
func apprun() error {

	var err error
	//TODO: Some works needed for TLS...
	listen_on := fmt.Sprintf("%s:%d", M_ip, M_port)
	atmi.TpLog(atmi.LOG_INFO, "About to listen on: (ip: %s, port: %d) %s",
		M_ip, M_port, listen_on)
	if TRUE == M_tls_enable {

		/* To prepare cert (self-signed) do following steps:
		 * - TODO
		 */
		err := http.ListenAndServeTLS(listen_on, M_tls_cert_file, M_tls_key_file, nil)
		atmi.TpLog(atmi.LOG_ERROR, "ListenAndServeTLS() failed: %s", err)
	} else {
		err := http.ListenAndServe(listen_on, nil)
		atmi.TpLog(atmi.LOG_ERROR, "ListenAndServe() failed: %s", err)
	}

	return err
}

//Init function, read config (with CCTAG)

//Un-init function
func appinit() error {
	//runtime.LockOSThread()

	M_url_map = make(map[string]string)

	if err := atmi.TpInit(); err != nil {
		return errors.New(err.Error())
	}

	//Get the configuration

	buf, err := atmi.NewUBF(16 * 1024)
	if nil != err {
		atmi.TpLog(atmi.LOG_ERROR, "Failed to allocate buffer: [%s]", err.Error())
		return errors.New(err.Error())
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", progsection, os.Getenv("NDRX_CCTAG")))

	if _, err := atmi.TpCall("@CCONF", buf, 0); nil != err {
		atmi.TpLog(atmi.LOG_ERROR, "ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return errors.New(err.Error())
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	//Set the parameters (ip/port/services)
	first := true
	// Load in the config...
	for {
		if fldid, occ, err := buf.BNext(first); nil == err {
			first = false
			atmi.TpLog(atmi.LOG_DEBUG, "BNext %d, %d", fldid, occ)
			fld_name, err := buf.BGetString(fldid, occ)

			if nil != err {
				atmi.TpLog(atmi.LOG_ERROR, "Failed to get field "+
					"%d occ %d", fldid, occ)
				return errors.New(err.Error())
			}

			atmi.TpLog(atmi.LOG_DEBUG, "Got config field [%s]", fld_name)

			switch fld_name {

			case "port":
				M_port, _ = buf.BGetInt16(u.EX_CC_VALUE, occ)
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
			default:

				if strings.HasPrefix(fld_name, "/") {
					cfg_val, _ := buf.BGetString(u.EX_CC_VALUE, occ)

					atmi.TpLog(atmi.LOG_DEBUG,
						"Got route: URL [%s] -> Service [%s]",
						fld_name, cfg_val)
					//Add route to hash list (or open listener on this..?)
					M_url_map[fld_name] = cfg_val
					//Add to HTTP listener
					http.HandleFunc(fld_name, handler)
				}
				break
			}

		} else {
			/* Done... */
			break
		}

	}

	if atmi.FAIL == M_port || "" == M_ip {
		atmi.TpLog(atmi.LOG_ERROR, "Invalid config: missing ip (%s) or port (%d)",
			M_ip, M_port)
		return errors.New("Invalid config: missing ip or port")
	}

	//Check the TLS settings
	if TRUE == M_tls_enable && (M_tls_cert_file == "" || M_tls_key_file == "") {

		atmi.TpLog(atmi.LOG_ERROR, "Invalid TLS settigns missing cert "+
			"(%s) or keyfile (%s) ", M_tls_cert_file, M_tls_key_file)

		return errors.New("Invalid config: missing ip or port")
	}

	return nil
}

//Service Main

func main() {

	if err := appinit(); nil != err {
		os.Exit(atmi.FAIL)
	}
	atmi.TpLog(atmi.LOG_DEBUG, "REST Incoming init ok - serving...")

	if err := apprun(); nil != err {
		os.Exit(atmi.FAIL)
	}

	os.Exit(atmi.SUCCEED)
}
