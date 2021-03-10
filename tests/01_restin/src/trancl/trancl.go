/**
 * @brief Transactional Web Service testing
 *  will add / get messages from TMQ with transactional approach
 *
 * @file trancl.go
 */
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	atmi "github.com/endurox-dev/endurox-go"
)

const WS_URL = "http://localhost:8081"

/**
 * Transaction API request
 */
type TxReqData struct {
	Operation string `json:"operation"`
	Timeout   uint64 `json:"timeout"`
	Flags     int64  `json:"flags"`
	Tptranid  string `json:"tptranid"`
}

/**
 * Transaction API response
 */
type TxRspData struct {
	Operation    string `json:"operation,omitempty"`
	ErrorCode    int    `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	Tptranid     string `json:"tptranid,omitempty"`
}

//Call the operation
//@param ac ATMI Context
//@param op operation name
//@param timeout timeout value for new transaction
//@param flag transaction flags
//@param tptranid transaction id
//@return transaction id(if applicable), error code
func callTranApi(ac *atmi.ATMICtx, op string, timeout uint64, flags int64, tptranid string) (string, int) {

	var treq TxReqData
	var trsp TxRspData

	treq.Operation = op
	treq.Flags = flags
	treq.Timeout = timeout

	if "" != tptranid {
		treq.Tptranid = tptranid
	}

	reqmsg, err := json.Marshal(&treq)

	if nil != err {
		ac.TpLogError("Failed to marshal request: %s", err.Error())
		return "", atmi.FAIL
	}

	req, err := http.NewRequest("POST", WS_URL+"/transactions", bytes.NewBuffer(reqmsg))

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		ac.TpLogError("Failed to request: %s", err.Error())
		return "", atmi.FAIL
	}

	defer resp.Body.Close()

	ac.TpLogInfo("response StatusCode: %d", resp.StatusCode)

	body, _ := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &trsp)

	if nil != err {
		ac.TpLogError("TESTERROR: Failed to unmarshal: %s", err.Error())
		return "", atmi.FAIL
	}

	ac.TpLogInfo("Operation: [%s]", trsp.Operation)
	ac.TpLogInfo("ErrorCode: [%d]", trsp.ErrorCode)
	ac.TpLogInfo("ErrorMessage: [%s]", trsp.ErrorMessage)
	ac.TpLogInfo("Tptranid: [%s]", trsp.Tptranid)

	if trsp.ErrorCode == atmi.TPEINVAL || trsp.ErrorCode == atmi.TPEPROTO {
		if resp.StatusCode != http.StatusBadRequest {
			ac.TpLogError("TESTERROR: Expected: %d got: %d ErrorCode: %d",
				http.StatusBadRequest, resp.StatusCode, trsp.ErrorCode)
			return "", atmi.FAIL
		}

	} else if trsp.ErrorCode > 0 {
		if resp.StatusCode != http.StatusInternalServerError {
			ac.TpLogError("TESTERROR: Expected: %d got: %d ErrorCode: %d",
				http.StatusInternalServerError, resp.StatusCode, trsp.ErrorCode)
			return "", atmi.FAIL
		}
	}

	return trsp.Tptranid, trsp.ErrorCode

}

//Run the test case
func apprun(ac *atmi.ATMICtx) error {

	for i := 0; i < 100; i++ {

		tid, ecode := callTranApi(ac, "tpbegin", 60, 0, "")

		if 0 != ecode {
			ac.TpLogError("TESTERROR: tpbegin Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: tpbegin Expected OK, got %d", ecode))
		}

		ac.TpLogInfo("Transaction [%s] started OK", tid)

		if i%2 == 0 {

			//abort transaction
			tid, ecode := callTranApi(ac, "tpabort", 0, 0, tid)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: tpabort Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: tpabort Expected OK, got %d", ecode))
			}

			ac.TpLogInfo("Transaction [%s] aborted OK", tid)
		} else {

			//commit transaction
			tid, ecode := callTranApi(ac, "tpcommit", 0, 0, tid)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: tpcommit Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: tpcommit Expected OK, got %d", ecode))
			}

			ac.TpLogInfo("Transaction [%s] aborted OK", tid)

			//Commit twice..

			//commit transaction
			tid, ecode = callTranApi(ac, "tpcommit", 0, 0, tid)

			if atmi.TPEABORT != ecode {
				ac.TpLogError("TESTERROR: tpcommit twice expected TPEABORT, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: tpcommit twice Expected TPEABORT, got %d", ecode))
			}

			ac.TpLogInfo("Transaction [%s] aborted OK", tid)
		}
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
		ac.TpLogError("apprun fail: %s", err)
		unInit(ac, atmi.FAIL)
	}

	unInit(ac, atmi.SUCCEED)
}
