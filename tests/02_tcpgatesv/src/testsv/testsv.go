package main

import (
	"errors"
	"fmt"
	"os"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "testsv"
)

//Validate the IP/PORT settings in UBF buffer
func validate_ip(ac *atmi.ATMICtx, ub *atmi.TypedUBF) error {

	//Test the IP and port existence
	if !ub.BPres(u.EX_NETOURIP, 0) {
		ac.TpLogError("CONSTAT: TESTERROR! Missing EX_NETOURIP!")
		return errors.New("CONSTAT: TESTERROR! Missing EX_NETOURIP!")
	}

	if !ub.BPres(u.EX_NETOURPORT, 0) {
		ac.TpLogError("CONSTAT: TESTERROR! Missing our EX_NETOURPORT!")
		return errors.New("CONSTAT: TESTERROR! Missing our EX_NETOURPORT!")
	}

	//Test the format of the values...
	ourip, _ := ub.BGetString(u.EX_NETOURIP, 0)

	if ourip == "" {
		ac.TpLogError("CONSTAT: TESTERROR! EX_NETOURPORT is empty!")
		return errors.New("CONSTAT: TESTERROR! EX_NETOURPORT is empty!")
	}

	ourport, _ := ub.BGetInt(u.EX_NETOURPORT, 0)

	if ourport <= 0 {
		ac.TpLogError("CONSTAT: TESTERROR! EX_NETOURPORT <=0!")
		return errors.New("CONSTAT: TESTERROR! EX_NETOURPORT <=0!")
	}

	if !ub.BPres(u.EX_NETTHEIRIP, 0) {
		ac.TpLogError("CONSTAT: TESTERROR! Missing EX_NETTHEIRIP!")
		return errors.New("CONSTAT: TESTERROR! Missing EX_NETTHEIRIP!")
	}

	if !ub.BPres(u.EX_NETTHEIRPORT, 0) {
		ac.TpLogError("CONSTAT: TESTERROR! Missing our EX_NETTHEIRPORT!")

		return errors.New("CONSTAT: TESTERROR! Missing our EX_NETTHEIRPORT!")
	}

	//Test the format of the values...
	theirip, _ := ub.BGetString(u.EX_NETTHEIRIP, 0)

	if theirip == "" {
		ac.TpLogError("CONSTAT: TESTERROR! EX_NETTHEIRIP is empty!")
		return errors.New("CONSTAT: TESTERROR! EX_NETTHEIRIP is empty!")
	}

	theirport, _ := ub.BGetInt(u.EX_NETTHEIRPORT, 0)

	if theirport <= 0 {
		ac.TpLogError("CONSTAT: TESTERROR! EX_NETTHEIRPORT <=0!")
		return errors.New("CONSTAT: TESTERROR! EX_NETTHEIRPORT <=0!")
	}

	//Validate connection mode

	conmode, _ := ub.BGetString(u.EX_NETCONMODE, 0)

	if conmode != "A" && conmode != "P" {
		ac.TpLogError("CONSTAT: TESTERROR! EX_NETCONMODE invalid value, "+
			"expected: A,B got: %s!", conmode)
		return errors.New("CONSTAT: TESTERROR! EX_NETCONMODE invalid value")
	}

	return nil
}

//Connection status service
func CONSTAT(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "CONSTAT: Incoming request:")

	gateway, _ := ub.BGetString(u.EX_NETGATEWAY, 0)
	con, _ := ub.BGetInt64(u.EX_NETCONNID, 0)

	flag, _ := ub.BGetString(u.EX_NETFLAGS, 0)

	var comp int64 = atmi.FAIL

	if ub.BPres(u.EX_NETCONNIDCOMP, 0) {
		comp, _ = ub.BGetInt64(u.EX_NETCONNIDCOMP, 0)
	}

	ac.TpLogInfo("CONSTAT: Gatway %s Connection %d (comp: %d) status %s",
		gateway, comp, con, flag)
	//Test the composite value

	if "C" == flag {

		if atmi.FAIL == comp {
			ac.TpLogError("CONSTAT: TESTERROR! Missing EX_NETCONNIDCOMP for connect!")
			ret = FAIL
			return
		}

		//Test that it matches basic id

		plain_from_comp := comp & 0xffffff

		if con != plain_from_comp {
			ac.TpLogError("CONSTAT: TESTERROR! Invalid connection ids plain "+
				"from comp (EX_NETCONNIDCOMP 0xffffff) = [%d] plain = [%d]!",
				plain_from_comp, con)
			ret = FAIL
			return
		}

		if nil != validate_ip(ac, ub) {
			ret = FAIL
			return
		}
	}
}

//Correlation service
//Will return correlator as first 4x bytes, if buffer larger than 4x bytes
func CORSVC(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {
	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "CORSVC: Incoming request:")

	arr, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}
	if arr[0] == 1 && arr[1] == 1 {
		ac.TpLogInfo("Test case 11 - no need for correlation")
	} else if len(arr) > 4 {

		corr := string(arr[:4])

		ac.TpLogInfo("Extracted correlator: [%s]", corr)

		if err := ub.BChg(u.EX_NETCORR, 0, corr); nil != err {
			ac.TpLogError("Failed to set EX_NETCORR: %s", err.Message())
			ret = FAIL
			return
		}

	}

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply buffer afrer correl")

}

//TESTOFFSETI service, process the offset, include len bytes
//@param ac ATMI Context
//@param svc Service call information
func TESTOFFSET(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "TESTSVC: Incoming request:")

	used, _ := ub.BUsed()
	//Resize buffer, to have some more space
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	ba, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}

	//Check the header
	inclLen := ba[0]

	ac.TpLogInfo("Len bytes included: %d", ba[0])

	if ba[1] != 0x12 {
		ac.TpLogError("Invalid header byte at idx 1 got exptected %d got %d", 0x12, ba[2])
		ret = FAIL
		return
	}

	if ba[2] != 0x13 {
		ac.TpLogError("Invalid header byte at idx 2 got exptected %d got %d", 0x13, ba[2])
		ret = FAIL
		return
	}

	if ba[3] != 0x15 {
		ac.TpLogError("Invalid header byte at idx 3 got exptected %d got %d", 0x15, ba[3])
		ret = FAIL
		return
	}

	rsp_len := 300

	if inclLen <= 0 {
		rsp_len -= 8
	}
	first_byte := byte((rsp_len >> 8) & 0xff)
	second_byte := byte(rsp_len & 0xff)

	if inclLen > 0 {
		//In this case we swap bytes too

		if first_byte != ba[4] {
			ac.TpLogError("TESTERROR LEN ind  at index %d, expected %d got %d",
				4, first_byte, ba[4])
			ret = FAIL
			return
		}

		if second_byte != ba[5] {
			ac.TpLogError("TESTERROR LEN ind at index %d, expected %d got %d",
				5, first_byte, ba[5])
			ret = FAIL
			return
		}
	} else {

		if first_byte != ba[6] {
			ac.TpLogError("TESTERROR (2) LEN ind  at index %d, expected %d got %d",
				5, first_byte, ba[5])
			ret = FAIL
			return
		}

		if second_byte != ba[7] {
			ac.TpLogError("TESTERROR (2) LEN ind at index %d, expected %d got %d",
				6, first_byte, ba[6])
			ret = FAIL
			return
		}
	}

	// OK now check the data
	for i := 8; i < 300; i++ {
		if ba[i] != byte(i%256) {
			ac.TpLogError("Invalid data recevied, expected: %d but got %d",
				byte(i%256), ba[i])
			ret = FAIL
			return
		}
	}

	//prepare outgoing buffer

	rsp := make([]byte, 400)

	copy(rsp, ba[0:7])

	for i := 8; i < len(rsp); i++ {
		rsp[i] = byte(int(i+1) % 256)
	}

	if errA := ub.BChg(u.EX_NETDATA, 0, rsp); nil != errA {
		ac.TpLogError("Failed to set EX_NETDATA %d:%s",
			errA.Code(), errA.Message())
		ret = FAIL
		return
	}

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply buffer")

	return
}

//TESTSVC service
//@param ac ATMI Context
//@param svc Service call information
func TESTSVC(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "TESTSVC: Incoming request:")

	//Test IP/PORt

	if nil != validate_ip(ac, ub) {
		ac.TpLogError("Failed to validate IP/PORT in incoming buffer!")
		ret = FAIL
		return
	}

	used, _ := ub.BUsed()
	//Resize buffer, to have some more space
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	arr, err := ub.BGetByteArr(u.EX_NETDATA, 0)

	if err != nil {
		ac.TpLogError("Failed to get EX_NETDATA: %s", err.Message())
		ret = FAIL
		return
	}

	//Test case 'A' - with correlation, reply back with cor
	if arr[0] == 'A' {
		ac.TpLogInfo("Running test case A")
		for i := 4; i < len(arr); i++ {
			arr[i] = byte((int(arr[i]+1) % 256))
			//Avoid stx/etx for later tests
			if arr[i] == 2 {
				arr[i] = 5
			}

			if arr[i] == 3 {
				arr[i] = 6
			}
		}

		err = ub.BChg(u.EX_NETDATA, 0, arr)

		if nil != err {
			ac.TpLogError("Failed to set EX_NETDATA: %s", err.Message())
			ret = FAIL
			return
		}

		//Kill the outgoing correlator, otherwise service will not just
		//Send the message, but also put it in waiters list!
		//But we need to send a reply to caller service....

		ub.BDel(u.EX_NETCORR, 0)

		ac.TpACall("TCP_P_ASYNC_A", ub, atmi.TPNOREPLY)

		//Check the if it is first test case (11), then
		//Verify all data sent
	} else if arr[0] == 1 && arr[1] == 1 {

		ac.TpLogInfo("First test case")
		for i := 2; i < 2048; i++ {
			if arr[i] != byte(i%256) {
				ac.TpLogError("TESTERROR: buffer index %d got "+
					"%d expected %d", i, arr[i], byte(i%256))
			}
		}

		ac.TpLogInfo("Test case 11 OK")

	} else {
		//NOTE: This basically is dumped, because we do not do reply back
		//and we were invoked in async way.
		ub.BDel(u.EX_NETDATA, 0)
	}

	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply buffer")

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
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
	if err := ac.TpAdvertise("TESTSVC", "TESTSVC", TESTSVC); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("CORSVC", "CORSVC", CORSVC); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("CONSTAT", "CONSTAT", CONSTAT); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("TESTOFFSET", "TESTOFFSET", TESTOFFSET); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("SEQTEST", "SEQTEST", SEQTEST); err != nil {
		ac.TpLogError("Failed to Advertise: ATMI Error %d:[%s]\n",
			err.Code(), err.Message())
		return atmi.FAIL
	}

	return SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...")
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
		ac.TpRun(Init, Uninit)
	}
}
