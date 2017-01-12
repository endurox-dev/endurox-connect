package main

import (
	"errors"
	"fmt"
	"os"
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

//Run the listener
func apprun(ac *atmi.ATMICtx) error {

	//Do some work here

	return nil
}

//Init function
//@param ac	ATMI context
//@return error (if erro) or nil
func appinit(ac *atmi.ATMICtx) error {
	//runtime.LockOSThread()

	if err := ac.TpInit(); err != nil {
		return errors.New(err.Error())
	}

	//Get the configuration
	buf, err := ac.NewUBF(16 * 1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s]", err.Error())
		return errors.New(err.Error())
	}

	//If we have a command line flag, then use it
	//else use CCTAG from env
	buf.BChg(u.EX_CC_CMD, 0, "g")

	subSection := ""

	if len(os.Args) > 1 {
		subSection = os.Args[1]
		ac.TpLogInfo("Using subsection from command line: [%s]", subSection)
	} else {
		subSection = os.Getenv("NDRX_CCTAG")
		ac.TpLogInfo("Using subsection from environment NDRX_CCTAG: [%s]",
			subSection)
	}

	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", ProgSection, subSection))

	if _, err := ac.TpCall("@CCONF", buf, 0); nil != err {
		ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return errors.New(err.Error())
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	//Set the parameters
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

		case "some_config_flag":
			MSomeConfigFlag, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogInfo("Got [%s] = [%s]", fldName, MSomeConfigFlag)
			break
		case "some_other_flag":
			MSomeOtherConfigFlag, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogInfo("Got [%s] = [%d]", fldName, MSomeOtherConfigFlag)
			break
		case "gencore":
			gencore, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if 1 == gencore {
				//Process signals by default handlers
				ac.TpLogInfo("gencore=1 - SIGSEG signal will be " +
					"processed by default OS handler")
				// Have some core dumps...
				C.signal(11, nil)
			}
			break
		default:
			ac.TpLogInfo("Unknown flag [%s] - ignoring...", fldName)
			break
		}

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
		unInit(ac, atmi.FAIL)
	}

	unInit(ac, atmi.SUCCEED)
}
