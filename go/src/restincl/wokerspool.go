package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"runtime"

	"github.com/endurox-dev/endurox-go/tests/06_ubf_marshal/src/atmi"
)

/*

So we will have a pool of goroutines which will wait on channel. This is needed
for doing less works with ATMI initialization and uninit.

Scheme will be following

- there will be array of goroutines, number is set in M_workwers
- there will be number same number of channels M_waitjobchan[M_workers]
- there will be M_freechan which will identify the free channel number (
when worker will complete its work, it will submit it's number to this channel)

So handler on new message will do <-M_freechan and then send message to -> M_waitjobchan[M_workers]
Workes will wait on <-M_waitjobchan[M_workers], when complete they will do Nr -> M_freechan

*/

//Needed for channel work submit
type HttpCall struct {
	w         http.ResponseWriter
	req       *http.Request
	terminate bool //Sent packet when thread should terminate
}

var M_freechan chan int //List of free channels submitted by wokers

var M_waitjobchan []chan HttpCall //Wokers channels each worker by it's number have a channel

// Request handler
//@param ac	ATMI Context
//@param w	Response writer (as usual)
//@param req	Request message (as usual)
func HandleMessage(ac *atmi.ATMICtx, w http.ResponseWriter, req *http.Request) {
	runtime.LockOSThread()
	ac.TpLog(atmi.LOG_DEBUG, "Got URL [%s]", req.URL)

	/* Send json to service */
	svc := M_url_map[req.URL.String()]
	if "" != svc.svc {

		body, _ := ioutil.ReadAll(req.Body)

		ac.TpLog(atmi.LOG_DEBUG, "Requesting service [%s] buffer [%s]", svc, body)

		buf, err := ac.NewJSON(body)

		if err != nil {
			ac.TpLog(atmi.LOG_ERROR, "ATMI Error %d:[%s]\n", err.Code(), err.Message())
			return
		}

		if _, err := ac.TpCall(svc.svc, buf.GetBuf(), 0); err != nil {
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

//Run the worker
//@param mynr	Woker number
func WorkerRun(mynr int) {
	terminate := false
	//Get the ATMI context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Errorf("Goroutine %d Failed to allocate cotnext!", mynr, err)
		os.Exit(atmi.FAIL)
	}

	err = ac.TpInit()

	if nil != err {
		ac.TpLogError("Goroutine %d failed to TpInit!", mynr, err)
		os.Exit(atmi.FAIL)
	}

	//Run until we get terminate message
	for !terminate {

		ac.TpLogDebug("Goroutine %d is free, waiting for next job", mynr)
		M_freechan <- mynr

		workblock := <-M_waitjobchan[mynr]

		if !workblock.terminate {
			HandleMessage(ac, workblock.w, workblock.req)
		} else {
			terminate = true
			ac.TpLogWarn("Thread %d got terminate message", mynr)
		}

	}
}

//Initialise channels and work pools
func InitPool() {

	M_freechan = make(chan int)

	for i := 0; i < M_workers; i++ {
		var callHanlder chan HttpCall
		M_waitjobchan = append(M_waitjobchan, callHanlder)
		go WorkerRun(i)
	}
}
