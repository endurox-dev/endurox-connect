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

var Mdone chan bool

func runMany(gw string, n int) {

	ac, errA := atmi.NewATMICtx()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "TESTERROR: Failed to allocate cotnext %d:%s!\n",
			errA.Code(), errA.Message())
		os.Exit(atmi.FAIL)
	}

	ba := make([]byte, 2048)

	for i := 2; i < len(ba); i++ {
		ba[i] = byte(i % 256)

		//Avoid stx/etx for later tests
		if ba[i] == 2 {
			ba[i] = 5
		}

		if ba[i] == 3 {
			ba[i] = 6
		}

	}

	//OK Realloc buffer back
	ub, errA := ac.NewUBF(3000)

	if nil != errA {
		ac.TpLogError("TESTERROR: Failed to allocate UBF buffer %d:%s",
			errA.Code(), errA.Message())
		os.Exit(atmi.FAIL)
	}

	//Test case with correlation
	ba[0] = 'A' //Test case A
	ba[1] = 'B' + byte(n%40)
	ba[2] = 'C' + byte(n%40)
	ba[3] = 'D' + byte(n%40)

	correl := string(ba[:4])

	ac.TpLogInfo("Built correlator [%s]", correl)

	if errA := ub.BChg(u.EX_NETCORR, 0, correl); nil != errA {
		ac.TpLogError("TESTERROR: Failed to set EX_NETCORR %d:%s",
			errA.Code(), errA.Message())
		Mdone <- false
		return
	}

	if errA := ub.BChg(u.EX_NETDATA, 0, ba); nil != errA {
		ac.TpLogError("TESTERROR: Failed to set EX_NETDATA %d:%s",
			errA.Code(), errA.Message())
		Mdone <- false
		return
	}

	//The reply here kills the buffer,
	//Thus we need a copy...
	ub.TpLogPrintUBF(atmi.LOG_INFO, "Calling server")
	ac.TpLogWarn("#%d [%s] Calling server", n, correl)
	if _, errA = ac.TpCall(gw, ub, 0); nil != errA {
		ac.TpLogError("TESTERROR: Failed to call [%s] %d:%s",
			gw, errA.Code(), errA.Message())
		Mdone <- false
		return
	}
	ac.TpLogWarn("#%d [%s] After server call", n, correl)

	//The response should succeed
	if rsp_code, err := ub.BGetInt(u.EX_NERROR_CODE, 0); nil != err {
		ac.TpLogError("TESTERROR: Failed to get EX_NERROR_CODE: %s",
			err.Message())
		Mdone <- false
		return
	} else if rsp_code != 0 {
		ac.TpLogError("TESTERROR: Response code must be 0 but got %d!",
			rsp_code)
		Mdone <- false
		return
	}

	//Verify response
	arrRsp, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("TESTERRO: Failed to get EX_NETDATA: %s", err.Message())
		Mdone <- false
		return
	}

	//Test the header in response, must match!
	correlGot := string(arrRsp[:4])

	ac.TpLogInfo("Built got [%s]", correlGot)

	if correlGot != correl {
		ac.TpLogError("TESTERROR: Correl sent: [%s] got [%s]", correlGot, correl)
		ac.TpLogDump(atmi.LOG_ERROR, "TESTERROR Message sent", ba, len(ba))
		ac.TpLogDump(atmi.LOG_ERROR, "TESTERROR Message received", arrRsp, len(arrRsp))
		Mdone <- false
		return
	}

	for i := 0; i < 4; i++ {
		if arrRsp[i] != ba[i] {
			ac.TpLogError("TESTERROR at index %d, expected %d got %d",
				i, ba[i], arrRsp[i])
			Mdone <- false
			return
		}
	}

	for i := 4; i < len(ba); i++ {
		exp := byte((int(ba[i]+1) % 256))
		//Avoid stx/etx for later tests
		if exp == 2 {
			exp = 5
		}

		if exp == 3 {
			exp = 6
		}

		if arrRsp[i] != exp {
			ac.TpLogError("TESTERROR at index %d, expected %d got %d",
				i, exp, arrRsp[i])
			Mdone <- false
			return
		}
	}

	ac.TpLogInfo("#%d done..", n)

	Mdone <- true

	ac.TpLogInfo("#%d done.. (return)", n)

	return

}

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
	case "corrsim":
		ac.TpLogInfo("Command: [%s] 2", command)
		if len(os.Args) < 4 {
			return errors.New(fmt.Sprintf("Missing count: %s corrsim <count> <gateway>",
				os.Args[0]))
		}

		nrOfTimes, err := strconv.Atoi(os.Args[2])

		Mdone = make(chan bool, nrOfTimes)

		gw := os.Args[3]

		if err != nil {
			return err
		}

		for i := 0; i < nrOfTimes; i++ {
			go runMany(gw, i)
		}

		for i := 0; i < nrOfTimes; i++ {
			ac.TpLogInfo("Waiting for reply of thread #%d", i)
			result := <-Mdone

			if !result {
				return errors.New(fmt.Sprintf("Thread %d failed", i))
			}
		}

		break
	case "corr":
		//case "nocon":
		//TODO: Try to send over connection number 1 it must be open
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
			//Avoid stx/etx for later tests
			if ba[i] == 2 {
				ba[i] = 5
			}

			if ba[i] == 3 {
				ba[i] = 6
			}
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

			correl := string(ba[:4])

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
				if arrRsp[i] != ba[i] {
					ac.TpLogError("TESTERROR at index %d, expected %d got %d",
						i, ba[i], arrRsp[i])
					return errors.New("TESTERROR in header!")
				}
			}

			//Test the msg
			for i := 4; i < len(ba); i++ {
				exp := byte((int(ba[i]+1) % 256))

				//Avoid stx/etx for later tests
				if exp == 2 {
					exp = 5
				}

				if exp == 3 {
					exp = 6
				}

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

			//Avoid stx/etx for later tests
			if ba[i] == 2 {
				ba[i] = 5
			}

			if ba[i] == 3 {
				ba[i] = 6
			}
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

	case "offsetsync":
		/* Call sync service test the request and resposne data
		 * we will send 300 bytes and we will receive 400 bytes
		 */
		ac.TpLogInfo("Command: [%s] 2", command)
		if len(os.Args) < 5 {
			return errors.New(fmt.Sprintf("Missing count: %s offsetsync <count> <gateway> <len_incl>",
				os.Args[0]))
		}

		nrOfTimes, err := strconv.Atoi(os.Args[2])

		gw := os.Args[3]

		if err != nil {
			return err
		}

		//Len of full header we have offset 4 + len bytes 4 => 8
		inclLen, err := strconv.Atoi(os.Args[4])

		if err != nil {
			return err
		}

		//Send the stuff out!!!
		//To async target
		for i := 0; i < nrOfTimes; i++ {

			ba := make([]byte, 300)

			if inclLen > 0 {
				ba[0] = 1
			} else {
				ba[0] = 0
			}

			ba[1] = 0x12
			ba[2] = 0x13
			ba[3] = 0x15

			for i := 4; i < len(ba); i++ {
				ba[i] = byte(i % 256)
			}

			//OK Realloc buffer back
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
				if arrRsp[i] != ba[i] {
					ac.TpLogError("TESTERROR at index %d, expected %d got %d",
						i, ba[i], arrRsp[i])
					return errors.New("TESTERROR in header!")
				}
			}

			//The lenght  of response packet is 400 bytes, thus verify the
			//header bytes. As we swap the halves and it was little endian
			//then first two bytes should be in front

			rsp_len := 400

			if inclLen <= 0 {
				rsp_len -= 8
			}
			first_byte := byte((rsp_len >> 8) & 0xff)
			second_byte := byte(rsp_len & 0xff)

			if inclLen > 0 {
				//In this case we swap bytes too

				if first_byte != arrRsp[4] {
					ac.TpLogError("TESTERROR LEN ind  at index %d, expected %d got %d",
						4, first_byte, arrRsp[4])
					return errors.New("TESTERROR in header!")
				}

				if second_byte != arrRsp[5] {
					ac.TpLogError("TESTERROR LEN ind at index %d, expected %d got %d",
						5, first_byte, arrRsp[5])
					return errors.New("TESTERROR in header!")
				}
			} else {

				if first_byte != arrRsp[5] {
					ac.TpLogError("TESTERROR (2) LEN ind  at index %d, expected %d got %d",
						5, first_byte, arrRsp[5])
					return errors.New("TESTERROR in header!")
				}

				if second_byte != arrRsp[6] {
					ac.TpLogError("TESTERROR (2) LEN ind at index %d, expected %d got %d",
						6, first_byte, arrRsp[6])
					return errors.New("TESTERROR in header!")
				}
			}

			if len(ba) != 400 {
				ac.TpLogError("Invalid response len: expected 400, got: %d", len(ba))
				return errors.New("Invalid response len!")
			}

			//Test the msg
			for i := 8; i < len(ba); i++ {
				exp := byte(int(i+1) % 256)

				if arrRsp[i] != exp {
					ac.TpLogError("TESTERROR at index %d, expected %d got %d",
						i, exp, arrRsp[i])
					return errors.New("TESTERROR in content!")
				}
			}
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
