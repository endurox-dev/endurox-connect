package main

import (
	"fmt"
	"os"
	"time"

	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED = 0
	FAIL    = -1
)

//Will set the trace file
func GETFILE(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	used, _ := ub.BUsed()
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	ub.BChg(u.EX_NREQLOGFILE, 0, fmt.Sprintf("%s/log/TRACE_%d",
		os.Getenv("NDRX_APPHOME"), time.Now().UnixNano()))

	return
}

//FAILSV1 service - returns error to caller, always
//@param ac ATMI Context
//@param svc Service call information
func FAILSV1(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	ret = FAIL

	return
}

//LONGOP service
//@param ac ATMI Context
//@param svc Service call information
func LONGOP(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, &svc.Data, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")
	ac.TpLogWarn("Sleeping 4 sec...")
	time.Sleep(4000 * time.Millisecond)

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//TODO: Run your processing here, and keep the succeed or fail status in
	//in "ret" flag.

	//Copy the incoming data from 1 to second field

	//Char
	char_val, err := ub.BGetByte(u.T_CHAR_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_CHAR_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_CHAR_2_FLD, 0, char_val)

	if nil != err {

		ac.TpLogError("Failed to set T_CHAR_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	return
}

//DATASV1 service
//@param ac ATMI Context
//@param svc Service call information
func DATASV1(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//Set request logging
	ac.TpLogSetReqFile(ub, "", "")

	//TODO: Run your processing here, and keep the succeed or fail status in
	//in "ret" flag.

	//Copy the incoming data from 1 to second field

	//Char
	char_val, err := ub.BGetByte(u.T_CHAR_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_CHAR_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_CHAR_2_FLD, 0, char_val)

	if nil != err {

		ac.TpLogError("Failed to set T_CHAR_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//Short
	short_val, err := ub.BGetInt16(u.T_SHORT_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_SHORT_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_SHORT_2_FLD, 0, short_val)

	if nil != err {

		ac.TpLogError("Failed to set T_SHORT_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//Long
	long_val, err := ub.BGetInt64(u.T_LONG_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_LONG_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_LONG_2_FLD, 0, long_val)

	if nil != err {

		ac.TpLogError("Failed to set T_LONG_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//Float
	float_val, err := ub.BGetFloat32(u.T_FLOAT_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_FLOAT_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_FLOAT_2_FLD, 0, float_val)

	if nil != err {

		ac.TpLogError("Failed to set T_FLOAT_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//Double
	double_val, err := ub.BGetFloat64(u.T_DOUBLE_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_DOUBLE_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_DOUBLE_2_FLD, 0, double_val)

	if nil != err {

		ac.TpLogError("Failed to set T_DOUBLE_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//String
	string_val, err := ub.BGetString(u.T_STRING_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_STRING_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_STRING_2_FLD, 0, string_val)

	if nil != err {

		ac.TpLogError("Failed to set T_STRING_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	//Byte array
	carray_val, err := ub.BGetString(u.T_CARRAY_FLD, 0)
	if nil != err {

		ac.TpLogError("Failed to get T_CARRAY_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	err = ub.BChg(u.T_CARRAY_2_FLD, 0, carray_val)

	if nil != err {

		ac.TpLogError("Failed to set T_CARRAY_2_FLD: %s", err.Message())
		ret = FAIL
		return
	}

	return
}

//LONGOP2 service (works with any buffer type)
//@param ac ATMI Context
//@param svc Service call information
func LONGOP2(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	//Return to the caller
	defer func() {
		ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
	}()

	ac.TpLogWarn("Sleeping 4 sec...")
	time.Sleep(4000 * time.Millisecond)

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")
	//Advertize TESTSVC
	if err := ac.TpAdvertise("DATASV1", "DATASV1", DATASV1); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("GETFILE", "GETFILE", GETFILE); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("LONGOP", "LONGOP", LONGOP); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("FAILSV1", "FAILSV1", FAILSV1); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("JSONSV", "JSONSV", JSONSV); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("TEXTSV", "TEXTSV", TEXTSV); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("BINARYSV", "BINARYSV", BINARYSV); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("VIEWSV1", "VIEWSV1", VIEWSV1); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("VIEWSV2", "VIEWSV2", VIEWSV2); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("VIEWFAIL", "VIEWFAIL", VIEWFAIL); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("VIEWFAIL2", "VIEWFAIL2", VIEWFAIL2); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("LONGOP2", "LONGOP2", LONGOP2); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("REGEXP", "REGEXP", REGEXP); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("REGEXPJSON", "REGEXPJSON", REGEXPJSON); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("COOKIES", "COOKIES", COOKIES); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("INMAND", "INMAND", INMAND); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("INOPT", "INOPT", INOPT); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("INERR", "INERR", INERR); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("OUTERR", "OUTERR", OUTERR); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("OUTMAND", "OUTMAND", OUTMAND); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("OUTOPT", "OUTOPT", OUTMAND); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("INOK", "INOK", INOK); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	if err := ac.TpAdvertise("INFAIL", "INFAIL", INFAIL); err != nil {
		fmt.Println(err)
		return atmi.FAIL
	}

	return atmi.SUCCEED
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
