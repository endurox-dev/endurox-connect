package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

/*
#include <signal.h>
*/
import "C"

const (
	ProgSection = "testcl"
)

var MSomeConfigFlag string = ""
var MSomeOtherConfigFlag int = 0
var MErrorCode int = atmi.TPMINVAL

type TestJSONMsg struct {
	StringField  string `json:"StringField"`
	StringField2 string `json:"StringField2"`
	NumField     int    `json:"NumField"`
	NumField2    int    `json:"NumField2"`
	BoolField    bool   `json:"BoolField"`
	BoolField2   bool   `json:"BoolField2"`
}

//Do the service call with UBF buffer
func UBFCall(ac *atmi.ATMICtx, cmd string, svc string, times string) error {

	nrTimes, _ := strconv.Atoi(times)

	for i := 0; i < nrTimes; i++ {

		buf, err := ac.NewUBF(1024)

		if err != nil {
			return errors.New(err.Error())
		}

		//Set some field for call
		buf.BChg(u.T_CHAR_FLD, 0, "A")
		buf.BChg(u.T_SHORT_FLD, 0, "123")
		buf.BChg(u.T_LONG_FLD, 0, i)
		buf.BChg(u.T_FLOAT_FLD, 0, "1.33")
		buf.BChg(u.T_DOUBLE_FLD, 0, "4444.3333")
		buf.BChg(u.T_STRING_FLD, 0, "HELLO")
		buf.BChg(u.T_CARRAY_FLD, 0, "WORLD")

		//Call the server
		if _, err := ac.TpCall(svc, buf, 0); nil != err {
			MErrorCode = err.Code()
			return errors.New(err.Error())
		}

		//Test the response...
		stmt := fmt.Sprintf("T_SHORT_2_FLD==123 "+
			"&& T_LONG_2_FLD==%d"+
			"&& T_CHAR_2_FLD=='A'"+
			"&& T_FLOAT_2_FLD==1.33"+
			"&& T_DOUBLE_2_FLD==4444.3333"+
			"&& T_STRING_2_FLD=='HELLO'"+
			"&& T_CARRAY_2_FLD=='WORLD'", i)

		if res, err := buf.BQBoolEv(stmt); !res || nil != err {
			if nil != err {
				return errors.New(fmt.Sprintf("juerrors: Expression "+
					"failed: %s", err.Error()))
			} else {
				return errors.New("juerrors: Expression is FALSE!: %s")
			}
		}
	}

	return nil
}

//Call the service with string buffer
func STRINGCall(ac *atmi.ATMICtx, cmd string, svc string, times string) error {

	nrTimes, _ := strconv.Atoi(times)

	for i := 0; i < nrTimes; i++ {

		buf, err := ac.NewString("Hi there!")

		if err != nil {
			return errors.New(err.Error())
		}

		//Call the server
		if _, err := ac.TpCall(svc, buf, 0); nil != err {
			MErrorCode = err.Code()
			return errors.New(err.Error())
		}

		//Test the response...
		s := buf.GetString()
		exp := "Hello from EnduroX"

		if s != exp {
			ac.TpLogError("Expected: [%s] got [%s]", exp, s)
			return errors.New(fmt.Sprintf("Expected: [%s] got [%s]",
				exp, s))
		}

	}

	return nil
}

//Call the sever with JSON buffer
func JSONCall(ac *atmi.ATMICtx, cmd string, svc string, times string) error {

	nrTimes, _ := strconv.Atoi(times)

	call := "{\"StringField\":\"Hello\", \"NumField\":12345, \"BoolField\":true}"

	for i := 0; i < nrTimes; i++ {

		var msg TestJSONMsg
		buf, err := ac.NewJSON([]byte(call))

		if err != nil {
			return errors.New(err.Error())
		}

		//Call the server
		if _, err := ac.TpCall(svc, buf, 0); nil != err {
			MErrorCode = err.Code()
			return errors.New(err.Error())
		}

		jerr := json.Unmarshal(buf.GetJSON(), &msg)
		if jerr != nil {
			return fmt.Errorf("Unmarshal: %s", jerr.Error())
		}

		//Test the response...
		if svc == "JSONJE_OK" || svc == "JSONJE_OKNS" {
			//Check the normal rsp...

			if msg.StringField2 != "Hello" {
				return fmt.Errorf("StringField2 expted [Hello],"+
					" got [%s]", msg.StringField2)
			}

			if !msg.BoolField2 {
				return errors.New("BoolField2 = true")
			}

			if msg.NumField2 != 12345 {

				return fmt.Errorf("NumField2 expected [12345],"+
					" got [%d]", msg.NumField)
			}
		}
	}

	return nil
}

func CmpArrays(a []byte, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

//Carray/RAW/BINARY tests...
//Call the sever with JSON buffer
func CARRAYCall(ac *atmi.ATMICtx, cmd string, svc string, times string) error {

	nrTimes, _ := strconv.Atoi(times)

	for i := 0; i < nrTimes; i++ {

		buf, err := ac.NewCarray([]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0})

		if err != nil {
			return errors.New(err.Error())
		}

		//Call the server
		if _, err := ac.TpCall(svc, buf, 0); nil != err {
			MErrorCode = err.Code()
			return errors.New(err.Error())
		}
		//Test the response...
		if !CmpArrays(buf.GetBytes(), []byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}) {
			return fmt.Errorf("Carray fail: expected [[]byte{9, 8, 7, 6, 5, 4, 3, 2, 1, 0}],"+
				" got [%x]", buf.GetBytes())
		}
	}

	return nil
}

//Call the view with REQUEST1
func VIEWCallREQUEST1(ac *atmi.ATMICtx, cmd string, svc string, times string) error {

	nrTimes, _ := strconv.Atoi(times)

	for i := 0; i < nrTimes; i++ {

		buf, err := ac.NewVIEW("REQUEST1", 0)

		if err != nil {
			return errors.New(err.Error())
		}

		//Set some values in buffer...
		ac.TpAssertEqualPanic(buf.BVChg("tshort1", 0, 5), nil, "Failed to set tshort1")
		ac.TpAssertEqualPanic(buf.BVChg("tlong1", 0, 77777), nil, "Failed to set tlong1")
		ac.TpAssertEqualPanic(buf.BVChg("tstring1", 1, "INCOMING TEST"), nil, "Failed to set tstring1")

		//Call the server
		if _, err := ac.TpCall(svc, buf, 0); nil != err {
			MErrorCode = err.Code()
			//In this case return error directly...
			if cmd == "view_request1_tout" {
				return errors.New(err.Error())
			}
		}

		//The data still must be parsed in!

		//Test the response...
		tstring1, errU := buf.BVGetString("tstring1", 0, 0)
		ac.TpAssertEqualPanic(errU, nil, "tstring1 must be present")
		ac.TpLogInfo("Got string: [%s]", tstring1)
		ac.TpAssertEqualPanic(tstring1, "HELLO RESPONSE", "tstring1 must be set "+
			"to \"HELLO RESPONSE\"")

		tstring1, errU = buf.BVGetString("tstring1", 1, 0)
		ac.TpAssertEqualPanic(errU, nil, "tstring1[1] must be present")
		ac.TpAssertEqualPanic(tstring1, "INCOMING TEST", "tstring1[1] must be set"+
			" to \"HELLO TESTCL\"")

		tlong1, errU := buf.BVGetInt64("tlong1", 0, 0)
		ac.TpAssertEqualPanic(errU, nil, "tlong1[0] must be present")
		ac.TpAssertEqualPanic(tlong1, 11111, "tlong1 must be 11111")
	}

	if 0 != MErrorCode {
		return errors.New("VIEWCallREQUEST1 failed")
	} else {
		return nil
	}

}

//Run the listener
func apprun(ac *atmi.ATMICtx) error {

	//Do some work here

	if len(os.Args) != 4 {
		return errors.New(fmt.Sprintf("usage: %s <command> <service> <times>",
			os.Args[0]))
	}

	cmd := os.Args[1]
	svc := os.Args[2]
	times := os.Args[3]

	ac.TpLogInfo("Got command: [%s], service: [%s], times: [%s]", cmd, svc, times)

	//These are projection on 01_restin/runtime/conf/restin.ini cases
	switch cmd {
	case "ubfcall":
		return UBFCall(ac, cmd, svc, times)
	case "stringcall":
		return STRINGCall(ac, cmd, svc, times)
	case "jsoncall":
		return JSONCall(ac, cmd, svc, times)
	case "carraycall":
		return CARRAYCall(ac, cmd, svc, times)
	case "view_request1", "view_request1_tout":

		return VIEWCallREQUEST1(ac, cmd, svc, times)
	default:
		return errors.New(fmt.Sprintf("Invalid test case: [%s]", cmd))
	}

}

//Init function
//@param ac	ATMI context
//@return error (if erro) or nil
func appinit(ac *atmi.ATMICtx) error {

	if err := ac.TpInit(); err != nil {
		return errors.New(err.Error())
	}

	return nil
}

//Un-init & Terminate the application
//@param ac	ATMI Context
//@param restCode	Return code. atmi.FAIL (-1) or atmi.SUCCEED(0)
func unInit(ac *atmi.ATMICtx, retCode int) {

	ac.TpTerm()
	ac.FreeATMICtx()
	os.Exit(retCode)
}

//Cliet process main entry
func main() {

	ac, errA := atmi.NewATMICtx()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate cotnext %d:%s!\n",
			errA.Code(), errA.Message())
		os.Exit(atmi.FAIL)
	}

	if err := appinit(ac); nil != err {
		ac.TpLogError("Failed to init: %s", err)
		os.Exit(atmi.FAIL)
	}

	ac.TpLogWarn("Init complete, processing...")

	if err := apprun(ac); nil != err {
		ac.TpLogError("Got error: [%s]", err.Error())
		if 0 == MErrorCode {
			MErrorCode = atmi.TPESYSTEM
		}
		unInit(ac, MErrorCode)
	}

	unInit(ac, atmi.SUCCEED)
}
