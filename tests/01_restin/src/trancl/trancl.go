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

/**
 * Enqueue message body
 */
type Enqmsg struct {
	Msgid    int64  `json:"T_LONG_FLD"`
	SomeData string `json:"T_STRING_FLD"`
}

/**
 * Generic response
 *
 */
type Genrsp struct {

	//Have all fields form Enqmsg
	Enqmsg

	ErrorCode    int    `json:"EX_IF_ECODE"`
	ErrorMessage string `json:"EX_IF_EMSG"`
	ErrorCodeQ   int    `json:"T_LONG_2_FLD"` //Error code from queue
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

//As enqueueURL, wrapper to enqueue
func enqueue(ac *atmi.ATMICtx, tptranid string, longfld int64) (string, int) {

	return enqueueURL(ac, tptranid, longfld, "/enqueue")
}

//Enqueue message
//@param ac ATMI context
//@param tptranid transaction id for the enqueue scope
//@param longfld long filed value
//@param url path to acces
//@return new TID (updated), ATMI error or -1 for generic error, 0 for ok
func enqueueURL(ac *atmi.ATMICtx, tptranid string, longfld int64, url string) (string, int) {

	var enq Enqmsg
	var enqrsp Genrsp

	enq.Msgid = longfld
	enq.SomeData = fmt.Sprintf("This is message No. %d", longfld)

	reqmsg, err := json.Marshal(&enq)

	if nil != err {
		ac.TpLogError("Failed to marshal request: %s", err.Error())
		return tptranid, atmi.FAIL
	}

	req, err := http.NewRequest("POST", WS_URL+url, bytes.NewBuffer(reqmsg))

	if err != nil {
		ac.TpLogError("Failed to prepare request: %s", err.Error())
		return tptranid, atmi.FAIL
	}

	//Set transaction header
	req.Header.Set("endurox-tptranid-req", tptranid)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		ac.TpLogError("Failed to request: %s", err.Error())
		return tptranid, atmi.FAIL
	}

	defer resp.Body.Close()

	ac.TpLogInfo("response StatusCode: %d", resp.StatusCode)

	body, _ := ioutil.ReadAll(resp.Body)

	tidrsp := resp.Header.Get("endurox-tptranid-rsp")

	err = json.Unmarshal(body, &enqrsp)

	if nil != err {
		ac.TpLogError("TESTERROR: Failed to unmarshal: %s", err.Error())
		return tidrsp, atmi.FAIL
	}

	ac.TpLogInfo("ErrorCode: [%d]", enqrsp.ErrorCode)
	ac.TpLogInfo("ErrorMessage: [%s]", enqrsp.ErrorMessage)

	//Return TID, if have one..

	return tidrsp, enqrsp.ErrorCode

}

//Dequeue message
//@param ac ATMI context
//@param tptranid transaction id for the enqueue scope
//@param longfld long field (check value)
//@return ATMI error or -1 for generic error, 0 for ok
func dequeue(ac *atmi.ATMICtx, longfld int64) int {

	var enq Enqmsg
	var enqrsp Genrsp

	enq.Msgid = longfld
	enq.SomeData = fmt.Sprintf("This is message No. %d", longfld)

	req, err := http.NewRequest("POST", WS_URL+"/dequeue", bytes.NewBuffer([]byte("{}")))

	if err != nil {
		ac.TpLogError("Failed to prepare request: %s", err.Error())
		return atmi.FAIL
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		ac.TpLogError("Failed to request: %s", err.Error())
		return atmi.FAIL
	}

	defer resp.Body.Close()

	ac.TpLogInfo("response StatusCode: %d", resp.StatusCode)

	body, _ := ioutil.ReadAll(resp.Body)

	err = json.Unmarshal(body, &enqrsp)

	if nil != err {
		ac.TpLogError("TESTERROR: Failed to unmarshal: %s", err.Error())
		return atmi.FAIL
	}

	ac.TpLogInfo("ErrorCode: [%d]", enqrsp.ErrorCode)
	ac.TpLogInfo("ErrorMessage: [%s]", enqrsp.ErrorMessage)
	ac.TpLogInfo("ErrorCodeQ: [%d]", enqrsp.ErrorCodeQ)

	//Return the error firstly...
	if enqrsp.ErrorCode > 0 {

		if enqrsp.ErrorCodeQ > 0 {
			return enqrsp.ErrorCodeQ
		}

		return enqrsp.ErrorCode
	}

	//validate the content
	if enqrsp.Msgid != enq.Msgid {
		ac.TpLogError("Invalid msgid expected %d got %d", enq.Msgid, enqrsp.Msgid)
		return atmi.FAIL
	}

	if enqrsp.SomeData != enq.SomeData {
		ac.TpLogError("Invalid SomeData expected [%s] got [%s]", enq.Msgid, enqrsp.Msgid)
		return atmi.FAIL
	}

	return 0

}

//Run the test case
func apprun(ac *atmi.ATMICtx) error {

	for i := 0; i < 2; i++ {

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

	for i := 0; i < 100; i++ {

		//Run some real transactions
		tid, ecode := callTranApi(ac, "tpbegin", 60, 0, "")

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (2) tpbegin Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (2) tpbegin Expected OK, got %d", ecode))
		}

		ac.TpLogInfo("Transaction [%s] started OK", tid)

		//Enqueue under the transaction some stuff 1
		tid, ecode = enqueue(ac, tid, 1)

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (1) enqueue Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (1) enqueue Expected OK, got %d", ecode))
		}

		//Enqueue under the transaction some stuff 2
		tid, ecode = enqueue(ac, tid, 2)

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (2) enqueue Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (2) enqueue Expected OK, got %d", ecode))
		}

		if i%2 == 0 {

			//abort transaction
			tid, ecode = callTranApi(ac, "tpabort", 0, 0, tid)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: (2) tpabort Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (2) tpabort Expected OK, got %d", ecode))
			}

			ac.TpLogInfo("Transaction [%s] aborted OK", tid)

			//Try to dequeue..., shall be empty...
			ecode = dequeue(ac, 1)
			if atmi.TPEDIAGNOSTIC != ecode {

				ac.TpLogError("TESTERROR: (1) dequeue Expected TPEDIAGNOSTIC, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (1) dequeue Expected TPEDIAGNOSTIC, got %d", ecode))

			}

			//Try to enqueue after the transaction is over..., expected to fail...

			//Enqueue under the transaction some stuff 2
			ac.TpLogInfo("Enqueue outside transaction tid [%s]", tid)
			tid, ecode = enqueue(ac, tid, 3)

			if 0 == ecode {
				ac.TpLogError("TESTERROR: (2.1) enqueue Expected fail, got %d tid = [%s]", ecode, tid)
				return errors.New(fmt.Sprintf("TESTERROR: (2.1) enqueue Expected fail, got %d tid = [%s]", ecode, tid))
			}

		} else {

			//commit transaction
			tid, ecode = callTranApi(ac, "tpcommit", 0, 0, tid)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: (3) tpcommit Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (3) tpcommit Expected OK, got %d", ecode))
			}

			//Dequeue 2x msgs ... & validate

			ecode = dequeue(ac, 1)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: (2) dequeue Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (2) dequeue Expected OK, got %d", ecode))
			}

			ecode = dequeue(ac, 2)

			if 0 != ecode {
				ac.TpLogError("TESTERROR: (3) dequeue Expected OK, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (3) dequeue Expected OK, got %d", ecode))
			}

			//space empty...
			ecode = dequeue(ac, 3)
			if atmi.TPEDIAGNOSTIC != ecode {

				ac.TpLogError("TESTERROR: (4) dequeue Expected TPEDIAGNOSTIC, got %d", ecode)
				return errors.New(fmt.Sprintf("TESTERROR: (4) dequeue Expected TPEDIAGNOSTIC, got %d", ecode))

			}

		}

		////////////////////////////////////////////////////////////////////////
		//Check auto-abort settings...
		////////////////////////////////////////////////////////////////////////

		////////////////////////////////////////////////////////////////////////
		//Enqueue to bad, but txnoabort is true
		////////////////////////////////////////////////////////////////////////
		tid, ecode = callTranApi(ac, "tpbegin", 60, 0, "")

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (2) tpbegin Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (2) tpbegin Expected OK, got %d", ecode))
		}

		ac.TpLogInfo("NOFAIL: Transaction [%s] started OK", tid)

		tid, ecode = enqueueURL(ac, tid, 1, "/enqueue_nofail")

		if atmi.TPESVCFAIL != ecode {
			ac.TpLogError("TESTERROR: (1) enqueue_nofail Expected TPESVCFAIL, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (1) enqueue_nofail Expected TPESVCFAIL, got %d", ecode))
		}

		//Commit shall be OK
		tid, ecode = callTranApi(ac, "tpcommit", 0, 0, tid)

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (4) tpcommit Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (4) tpcommit Expected OK, got %d", ecode))
		}

		////////////////////////////////////////////////////////////////////////
		//Enqueue to bad, but txnoabort is false (default)_
		////////////////////////////////////////////////////////////////////////
		tid, ecode = callTranApi(ac, "tpbegin", 60, 0, "")

		if 0 != ecode {
			ac.TpLogError("TESTERROR: (2) tpbegin Expected OK, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (2) tpbegin Expected OK, got %d", ecode))
		}

		ac.TpLogInfo("NOFAIL: Transaction [%s] started OK", tid)

		tid, ecode = enqueueURL(ac, tid, 1, "/enqueue_fail")

		if atmi.TPESVCFAIL != ecode {
			ac.TpLogError("TESTERROR: (1) enqueue_nofail Expected TPESVCFAIL, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (1) enqueue_nofail Expected TPESVCFAIL, got %d", ecode))
		}

		//Commit shall be OK
		tid, ecode = callTranApi(ac, "tpcommit", 0, 0, tid)

		if atmi.TPEABORT != ecode {
			ac.TpLogError("TESTERROR: (5) tpcommit Expected TPEABORT, got %d", ecode)
			return errors.New(fmt.Sprintf("TESTERROR: (5) tpcommit Expected TPEABORT, got %d", ecode))
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
