package main

import (
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

//Run the listener
func apprun(ac *atmi.ATMICtx) error {

	//Do some work here
	command := os.Args[1]

	ac.TpLogInfo("Command: [%s]", command)
	switch command {
	case "async_call", "nocon":
		//case "nocon":
		ac.TpLogInfo("Command: [%s] 2", command)
		if len(os.Args) < 4 {
			return errors.New(fmt.Sprintf("Missing count: %s async_call <count> <gateway>",
				os.Args[0]))
		}

		nrOfTimes, err := strconv.Atoi(os.Args[2])

		gw := os.Args[3]

		if err != nil {
			return err
		}

		ba := make([]byte, 2048)

		//Test case 11
		ba[0] = 1
		ba[1] = 1

		for i := 2; i < len(ba); i++ {
			ba[i] = byte(i % 256)
		}

		//Setup the buffer

		ub, errA := ac.NewUBF(3000)

		if nil != errA {
			ac.TpLogError("Failed to allocate UBF buffer %d:%s",
				errA.Code(), errA.Message())
			return errors.New(errA.Message())
		}

		if errA := ub.BChg(u.EX_NETDATA, 0, ba); nil != errA {
			ac.TpLogError("Failed to set EX_NETDATA %d:%s",
				errA.Code(), errA.Message())
			return errors.New(errA.Message())
		}

		//Send the stuff out!!!
		//To async target
		for i := 0; i < nrOfTimes; i++ {

			//The reply here kills the buffer,
			//Thus we need a copy...
			ub.TpLogPrintUBF(atmi.LOG_INFO, "Calling server")
			if _, errA = ac.TpCall(gw, ub, 0); nil != errA {
				ac.TpLogError("Failed to call [%s] %d:%s",
					gw, errA.Code(), errA.Message())
			}

			//The response should succeed
			if rsp_code, err := ub.BGetInt(u.EX_NERROR_CODE, 0); nil != err {
				ac.TpLogError("TESTERROR: Failed to get EX_NERROR_CODE: %s",
					err.Message())
				return errors.New(err.Message())
			} else if rsp_code != 0 {
				if command == "nocon" && rsp_code == atmi.NENOCONN {
					ac.TpLogError("No connection test ok")
				} else {
					ac.TpLogError("TESTERROR: Response code must be 0 but got %d!",
						rsp_code)
					return errors.New("Invalid response code")
				}
			}

			//OK Realloc buffer back
			ub, errA = ac.NewUBF(3000)

			if nil != errA {
				ac.TpLogError("Failed to allocate UBF buffer %d:%s",
					errA.Code(), errA.Message())
				return errors.New(errA.Message())
			}

			if errA = ub.BChg(u.EX_NETDATA, 0, ba); nil != errA {
				ac.TpLogError("Failed to set EX_NETDATA %d:%s",
					errA.Code(), errA.Message())
				return errors.New(errA.Message())
			}
		}

		break
	case "corr":
		//case "nocon":
		ac.TpLogInfo("Command: [%s] 2", command)
		if len(os.Args) < 4 {
			return errors.New(fmt.Sprintf("Missing count: %s corr <count> <gateway>",
				os.Args[0]))
		}

		nrOfTimes, err := strconv.Atoi(os.Args[2])

		gw := os.Args[3]

		if err != nil {
			return err
		}

		ba := make([]byte, 2048)

		for i := 2; i < len(ba); i++ {
			ba[i] = byte(i % 256)
		}

		//Send the stuff out!!!
		//To async target
		for i := 0; i < nrOfTimes; i++ {

			//OK Realloc buffer back
			ub, errA := ac.NewUBF(3000)

			if nil != errA {
				ac.TpLogError("Failed to allocate UBF buffer %d:%s",
					errA.Code(), errA.Message())
				return errors.New(errA.Message())
			}

			//Test case with correlation
			ba[0] = 'A' //Test case A
			ba[1] = 'B' + byte(i%10)
			ba[2] = 'C' + byte(i%10)
			ba[3] = 'D' + byte(i%10)

			correl:=string(ba[:4])

			ac.TpLogInfo("Built correlator [%s]", correl)

			if errA := ub.BChg(u.EX_NETCORR, 0, correl); nil != errA {
				ac.TpLogError("Failed to set EX_NETCORR %d:%s",
					errA.Code(), errA.Message())
				return errors.New(errA.Message())
			}

			if errA := ub.BChg(u.EX_NETDATA, 0, ba); nil != errA {
				ac.TpLogError("Failed to set EX_NETDATA %d:%s",
					errA.Code(), errA.Message())
				return errors.New(errA.Message())
			}

			//The reply here kills the buffer,
			//Thus we need a copy...
			ub.TpLogPrintUBF(atmi.LOG_INFO, "Calling server")
			if _, errA = ac.TpCall(gw, ub, 0); nil != errA {
				ac.TpLogError("Failed to call [%s] %d:%s",
					gw, errA.Code(), errA.Message())
			}

			//The response should succeed
			if rsp_code, err := ub.BGetInt(u.EX_NERROR_CODE, 0); nil != err {
				ac.TpLogError("TESTERROR: Failed to get EX_NERROR_CODE: %s",
					err.Message())
				return errors.New(err.Message())
			} else if rsp_code != 0 {
				ac.TpLogError("TESTERROR: Response code must be 0 but got %d!",
					rsp_code)
				return errors.New("Invalid response code")
			}

			//Verify response
			arrRsp, err := ub.BGetByteArr(u.EX_NETDATA, 0)

			if err != nil {
				ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
				return errors.New("Failed to get EX_NETDATA!")
			}

			//Test the header in response, must match!
			for i := 0; i < 4; i++ {
				if arrRsp[i]!=ba[i] {
					ac.TpLogError("TESTERROR at index %d, expected %d got %d",
					i, ba[i], arrRsp[i])
					return errors.New("TESTERROR in header!")
				}
			}

			//Test the msg
			for i := 4; i < len(ba); i++ {
				exp:=byte((int(ba[i] + 1) % 256))
				if arrRsp[i] != exp {
					ac.TpLogError("TESTERROR at index %d, expected %d got %d",
					i, exp, arrRsp[i])
					return errors.New("TESTERROR in content!")
				}
			}
		}
		break
	case "corrtot":
		//case "nocon":
		ac.TpLogInfo("Command: [%s] 3", command)
		if len(os.Args) < 3 {
			return errors.New(fmt.Sprintf("Missing count: %s corrtot <gateway>",
				os.Args[0]))
		}

		gw := os.Args[2]


		ba := make([]byte, 2048)

		for i := 0; i < len(ba); i++ {
			ba[i] = byte(i % 256)
		}

		//OK Realloc buffer back
		ub, errA := ac.NewUBF(3000)

		if nil != errA {
			ac.TpLogError("Failed to allocate UBF buffer %d:%s",
				errA.Code(), errA.Message())
			return errors.New(errA.Message())
		}

		if errA := ub.BChg(u.EX_NETCORR, 0, "HELLO NO SUCH CORR"); nil != errA {
			ac.TpLogError("Failed to set EX_NETCORR %d:%s",
				errA.Code(), errA.Message())
			return errors.New(errA.Message())
		}

		if errA := ub.BChg(u.EX_NETDATA, 0, ba); nil != errA {
			ac.TpLogError("Failed to set EX_NETDATA %d:%s",
				errA.Code(), errA.Message())
			return errors.New(errA.Message())
		}

		//The reply here kills the buffer,
		//Thus we need a copy...
		ub.TpLogPrintUBF(atmi.LOG_INFO, "Calling server")
		if _, errA = ac.TpCall(gw, ub, 0); nil != errA {
			ac.TpLogError("Failed to call [%s] %d:%s",
				gw, errA.Code(), errA.Message())
		}
		ub.TpLogPrintUBF(atmi.LOG_INFO, "Got response")

		//The response should succeed
		if rsp_code, err := ub.BGetInt(u.EX_NERROR_CODE, 0); nil != err {
			ac.TpLogError("TESTERROR: Failed to get EX_NERROR_CODE: %s",
				err.Message())
			return errors.New(err.Message())
		} else if rsp_code != atmi.NETOUT {
			ac.TpLogError("TESTERROR: Response code must be %d but got %d!",
				atmi.NETOUT, rsp_code)
			return errors.New("TESTERROR: Invalid response code")
		}

		break
	}

	return nil
}

//Init function
//@param ac	ATMI context
//@return error (if erro) or nil
func appinit(ac *atmi.ATMICtx) error {

	if err := ac.TpInit(); err != nil {
		return errors.New(err.Error())
	}

	if len(os.Args) < 2 {
		return errors.New(fmt.Sprintf("Missing arguments: %s <command>",
			os.Args[0]))
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
