package main

import (
	"fmt"
	"os"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "@tcpgate"

	CON_TYPE_PASSIVE = "P"
	CON_TYPE_ACTIVE  = "A"
)

//XATMI sessions for outgoing (Enduro/X sends to Network)
var MworkersOut int = 5

//XATMI sessions for incoming (network sends to Enduro/X)
var MworkersIn int = 5

//TCP Gateway
var Mgateway string = "tcpgate"
var Mframing string = "llll"

//In case if framing is "d"
var MdelimStart byte = 0x02      //can be optional
var MdelimStop byte = 0x03       //Can be optional
var Mtype string = "P"           //A - Active, P - Pasive
var Mip string = "0.0.0.0"       //IP to listen or to connect to if Active
var Mport int = 5555             //Port to connect to or listen on depending on active/passive role
var MincomingService string = "" //Incomding service to send to incoming async traffic
var MperZero int = 60            //Period by witch to which send zero length message to all channels...
var Mstatussvc string = ""       //Status service to which send connection information
//Max number to connection to connect to server, or allow max incomings in the same time.
var MmaxConnections int = 5

//Request reply model, alos for in-out, sync mode
//Open connection for incoming wait for reply,
//and Close the connection .
var MreqReply bool = false

//Timeout for req-reply model
var MreqReplyTimeout = 60

//TCPGATE service
//@param ac ATMI Context
//@param svc Service call information
func TCPGATE(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpContinue()
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, &svc.Data, 0)
		}
	}()

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	//Print the buffer to stdout
	//fmt.Println("Incoming request:")
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	if err := ub.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//Pack the request data to pass to thread
	ctxData, err := ac.TpSrvGetCtxData()
	if nil != err {
		ac.TpLogError("Failed to get context data - dropping request",
			err.Code(), err.Message())
		ret = FAIL
		return
	}
	nr := getFreeXChan(ac, &MoutXPool)
	go XATMIDispatchCall(&MoutXPool, nr, ctxData, ub)

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

		case "workers_out":
			MworkersOut, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MworkersOut)
			break
		case "workers_in":
			MworkersIn, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MworkersIn)
			break
		case "gateway":
			Mgateway, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, Mgateway)
			break
		case "framing":
			Mframing, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, Mframing)
			break
		case "delim_start":
			MdelimStart, _ = buf.BGetByte(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, MdelimStart)
			break
		case "delim_stop":
			MdelimStop, _ = buf.BGetByte(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, MdelimStop)
			break
		case "type":
			Mtype, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, Mtype)

			if Mtype == "a" {
				Mtype = CON_TYPE_ACTIVE
			} else if Mtype == "p" {
				Mtype = CON_TYPE_PASSIVE
			}

			if Mtype != "p" && Mtype != "P" && Mtype != "a" && Mtype != "A" {
				ac.TpLogError("Invalid connection type [%s] - "+
					"support a/A/p/P ", Mtype)
				return FAIL
			}
			break
		case "ip":
			Mip, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, Mip)
			break
		case "port":
			Mport, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, Mport)
			break
		case "incoming_service":
			MincomingService, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MincomingService)
			break
		case "per_zero":
			MperZero, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MperZero)
			break
		case "status_service":
			Mstatussvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, Mstatussvc)
			break
		case "max_connections":
			MmaxConnections, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MmaxConnections)
			break
		case "req_reply":
			tmp, _ := buf.BGetInt(u.EX_CC_VALUE, occ)
			if 1 == tmp {
				MreqReply = true
			} else {
				MreqReply = false
			}

			ac.TpLogDebug("Got [%s] = [%b] ", fldName, MreqReply)
			break
		case "req_reply_timeout":
			MreqReplyTimeout, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MreqReplyTimeout)
			break

		default:

			break
		}
	}

	MinXPool.nrWorkers = MworkersIn
	MoutXPool.nrWorkers = MworkersOut

	if err := initPool(ac, &MinXPool); err != nil {
		ac.TpLogError("Failed to init `in' XATMI pool %s", err)
		return FAIL
	}

	if err := initPool(ac, &MoutXPool); err != nil {
		ac.TpLogError("Failed to init `out' XATMI pool %s", err)
		return FAIL
	}

	//Advertize TESTSVC
	if err := ac.TpAdvertise(Mgateway, Mgateway, TCPGATE); err != nil {
		ac.TpLogError("Advertise failed %s", err)
		return FAIL
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
