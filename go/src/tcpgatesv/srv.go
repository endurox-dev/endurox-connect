package main

import (
	"fmt"
	"os"
	"strings"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = 0
	FAIL        = -1
	PROGSECTION = "TPCGATE"
)

func serve(ub *atmi.TypedUBF) int {

	first := true
	outmsg := ""

	cid, err := ub.BGetInt64(ubftab.L_CONID, 0)
	if nil != err {
		fmt.Printf("Missing L_CONID - outgoing connection id)")
		return FAIL
	}

	for {
		fid, occ, err := ub.BNext(first)
		if err != nil {
			break
		}
		first = false

		f, _ := ub.BGetString(fid, occ)

		//Get the field name
		fn, err := atmi.BFname(fid)

		if nil != err {
			fmt.Printf("Cannot translated field %ld - Got error: %d:[%s]\n",
				fid, err.Code(), err.Message())
			return FAIL
		}
		fmt.Printf("field [%s] = [%s]\n", fn, f)

		if outmsg == "" {
			outmsg = strings.Join([]string{fn, "=", f}, "")
		} else {
			outmsg = strings.Join([]string{outmsg, "|", fn, "=", f}, "")
		}
	}
	//Put the final newline...
	outmsg = strings.Join([]string{outmsg, "\n"}, "")

	fmt.Printf("Outgoing message [%s]\n", outmsg)
	if nil != M_leSrv.clients[cid] {
		M_leSrv.clients[cid].outgoing <- outmsg
		fmt.Printf("Message delivered to channel")
	} else {
		fmt.Printf("Connection not active!")
		return FAIL
	}

	return SUCCEED
}

func TCPSRV(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	ub, _ := atmi.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	fmt.Println("Incoming request:")
	ub.BPrint()

	ret = serve(&ub)

	//Return to the caller
	if SUCCEED == ret {
		atmi.TpReturn(atmi.TPSUCCESS, 0, &ub, 0)
	} else {
		atmi.TpReturn(atmi.TPFAIL, 0, &ub, 0)
	}
	return
}

//Server init
func Init(ac *atmi.ATMICtx) int {
	ac.TpLogWarn("Doing server init...")
	if err := ac.TpInit(); err != nil {
		return FAIL
	}

	//Get the configuration

	//Allocate configuration buffer
	buf, err := ac.NewUBF(16 * 1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s]", err.Error())
		return FAIL
	}

	buf.BChg(u.EX_CC_CMD, 0, "g")
	buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", PROGSECTION, os.Getenv("NDRX_CCTAG")))

	if _, err := ac.TpCall("@CCONF", buf, 0); nil != err {
		ac.TpLogError("ATMI Error %d:[%s]\n", err.Code(), err.Message())
		return FAIL
	}

	//Dump to log the config read
	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Got configuration.")

	occs, _ := buf.BOccur(u.EX_CC_KEY)

	// Load in the config...
	for occ := 0; occ < occs; occ++ {
		ac.TpLogDebug("occ %d", occ)
		fldName, err := buf.BGetString(u.EX_CC_KEY, occ)

		if nil != err {
			ac.TpLogError("Failed to get field "+
				"%d occ %d", u.EX_CC_KEY, occ)
			return FAIL
		}

		ac.TpLogDebug("Got config field [%s]", fldName)

		switch fldName {

		case "mykey1":
			myval, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, myval)
			break

		default:

			break
		}
	}
	//Advertize TESTSVC
	if err := ac.TpAdvertise("TESTSVC", "TCPSRV", TCPSRV); err != nil {
		fmt.Println(err)
		return FAIL
	}

	return atmi.SUCCEED
}

//Server shutdown
func Uninit(ac *atmi.ATMICtx) {
	fmt.Println("Server shutting down...")
}

//Executable main entry point
func main() {
	//Have some context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Errorf("Failed to allocate cotnext!", err)
		os.Exit(atmi.FAIL)
	} else {
		//Run as server
		ac.TpRun(Init, Uninit)
		ac.FreeATMICtx()
	}
}
