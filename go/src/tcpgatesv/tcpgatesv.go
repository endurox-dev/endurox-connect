package main

import (
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

const (
	SUCCEED     = atmi.SUCCEED
	FAIL        = atmi.FAIL
	PROGSECTION = "@tcpgate"

	//Connection type
	CON_TYPE_PASSIVE = "P"
	CON_TYPE_ACTIVE  = "A"

	//Req-reply mode:
	RR_PERS_ASYNC_INCL_CORR = 0 //Persistent, async mode including correlation
	RR_PERS_CONN_EX2NET     = 1 //Persistent, sync by connection, EX sends to Net
	RR_PERS_CONN_NET2EX     = 2 //Persistent, sync by connection, Net sends to Ex
	RR_NONPERS_EX2NET       = 3 //Non-persistent, sync each request - new connection, Enduro sends to Net
	RR_NONPERS_NET2EX       = 4 //Non-persistent, sync each request - new connection, Net sends to Enduro

	//Connection flags
	FLAG_CON_DISCON      = "D"
	FLAG_CON_ESTABLISHED = "C"

	RUN_CONTINUE      = 0
	RUN_SHUTDOWN_OK   = 1
	RUN_SHUTDOWN_FAIL = 2

	CHANNEL_SIZE = 100
)

//XATMI sessions for outgoing (Enduro/X sends to Network)
var MWorkersOut int = 5

//XATMI sessions for incoming (network sends to Enduro/X)
var MworkersIn int = 5

//TCP Gateway
var MGateway string = "TCPGATE"
var MFraming string = "llll"
var MFramingCode rune = FRAME_LITTLE_ENDIAN
var MFramingLen int = len(MFraming)
var MFamingInclPfxLen bool = false //Does len format include prefix length it self?
var MFramingMaxMsgLen int = 0      //Max message len (checked if >0)

//In case if framing is "d"
var MDelimStart byte = 0x02  //can be optional
var MDelimStop byte = 0x03   //Can be optional
var MType string = "P"       //A - Active, P - Pasive
var MIp string = "0.0.0.0"   //IP to listen or to connect to if Active
var MPort int = 5555         //Port to connect to or listen on depending on active/passive role
var MAddr string = ""        //Compiled ip:port
var MIncomingSvc string = "" //Incomding service to send to incoming async traffic
var MPerZero int = 60        //Period by witch to which send zero length message to all channels...
var MStatussvc string = ""   //Status service to which send connection information
//Max number to connection to connect to server, or allow max incomings in the same time.
var MMaxConnections int64 = 5

//Request reply model, alos for in-out, sync mode
//Open connection for incoming wait for reply,
//and Close the connection .
var MReqReply int = RR_PERS_ASYNC_INCL_CORR

//Timeout for req-reply model
var MReqReplyTimeout int64 = 60

var MScanTime = 1 //Seconds for housekeeping

//Correlator service for incoming messages
//This is used case if driver operates in sync mode over the persistently conneced lines
var MCorrSvc = ""

var MShutdown int = RUN_CONTINUE

var MActiveConScan int = 5 //scan for new outgoing connections every 10 seconds

//TCPGATE service
//@param ac ATMI Context
//@param svc Service call information
func TCPGATE(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Return to the caller
	defer func() {

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			/* ac.TpContinue() - No need for this
			 * Or it have nothing todo.
			 * as operation  must be last.
			 */
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
	go XATMIDispatchCall(&MoutXPool, nr, ctxData, &ub)

	return
}

//Server init, called when process is booted
//@param ac ATMI Context
func Init(ac *atmi.ATMICtx) int {

	ac.TpLogWarn("Doing server init...")

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
		case "workers_out":
			MWorkersOut, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MWorkersOut)
			break
		case "workers_in":
			MworkersIn, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MworkersIn)
			break
		case "gateway":
			MGateway, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MGateway)
			break
		case "framing":
			MFraming, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MFraming)
			break
		case "max_msg_len":
			MFramingMaxMsgLen, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MFramingMaxMsgLen)
			break
		case "delim_start":
			MDelimStart, _ = buf.BGetByte(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, MDelimStart)
			break
		case "delim_stop":
			MDelimStop, _ = buf.BGetByte(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, MDelimStop)
			break
		case "type":
			MType, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MType)

			if MType == "a" {
				MType = CON_TYPE_ACTIVE
			} else if MType == "p" {
				MType = CON_TYPE_PASSIVE
			}

			if MType != "p" && MType != "P" && MType != "a" && MType != "A" {
				ac.TpLogError("Invalid connection type [%s] - "+
					"support a/A/p/P ", MType)
				return FAIL
			}
			break
		case "ip":
			MIp, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MIp)
			break
		case "port":
			MPort, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MPort)
			break
		case "incoming_svc":
			MIncomingSvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MIncomingSvc)
			break
		case "periodic_zero_msg":
			MPerZero, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MPerZero)
			break
		case "status_svc":
			MStatussvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MStatussvc)
			break
		case "max_connections":
			MMaxConnections, _ = buf.BGetInt64(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MMaxConnections)
			break
		case "req_reply":
			MReqReply, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MReqReply)
			break
		case "req_reply_timeout":
			MReqReplyTimeout, _ = buf.BGetInt64(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MReqReplyTimeout)
			break
		case "scan_time":
			MScanTime, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MScanTime)
			break
		case "corr_svc":
			//Corelator service for sync tpcall over mulitple persistent connectinos
			MCorrSvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MReqReplyTimeout)
			break
		default:

			break
		}
	}

	MinXPool.nrWorkers = MworkersIn
	MoutXPool.nrWorkers = MWorkersOut

	MAddr = MIp + ":" + strconv.Itoa(MPort)

	if err := initPool(ac, &MinXPool); err != nil {
		ac.TpLogError("Failed to init `in' XATMI pool %s", err)
		return FAIL
	}

	if err := initPool(ac, &MoutXPool); err != nil {
		ac.TpLogError("Failed to init `out' XATMI pool %s", err)
		return FAIL
	}

	ac.TpLogInfo("Period housekeeping: scan_time - %d", MScanTime)

	if errS := ConfigureNumberOfBytes(ac); errS != nil {
		ac.TpLogError("Failed to configure number of bytes to use for "+
			"message frame: %s", errS.Error())
		return FAIL
	}

	// Verify all work scenarios.
	if MReqReply == RR_PERS_ASYNC_INCL_CORR {
		ac.TpLogInfo("Persistent connections: Working on fully async " +
			"mode with or without corelators")

		if MIncomingSvc == "" {
			ac.TpLogError("Missing mandatory config key: incoming_svc!")
			return FAIL
		}

		ac.TpLogInfo("Incoming service: [%s]", MIncomingSvc)
		ac.TpLogInfo("Correlation service: [%s]", MCorrSvc)
		ac.TpLogInfo("Network timeout: %d", MReqReplyTimeout)

	} else if MReqReply == RR_PERS_CONN_EX2NET {
		ac.TpLogInfo("Persistent connections: Synchronous, Enduro/X requests to Network")
		ac.TpLogInfo("Network timeout: %d", MReqReplyTimeout)
	} else if MReqReply == RR_PERS_CONN_NET2EX {
		ac.TpLogInfo("Persistent connections: Synchronous, Network requests Enduro/X")
		ac.TpLogInfo("Network timeout: %d", MReqReplyTimeout)
	} else if MReqReply == RR_NONPERS_EX2NET {
		ac.TpLogInfo("Non-persistent connections: Synchronous, Enduro/X requests Network")
		if MType != CON_TYPE_ACTIVE {
			ac.TpLogInfo("Connection type (type) must be Active [%s], but got: %s!",
				CON_TYPE_ACTIVE, MType)
			return FAIL
		}

		if MPerZero > 0 {
			ac.TpLogError("Periodic zero (periodic_zero_msg) parameter"+
				" cannot be used with req_reply type %d", MReqReply)
			return FAIL
		}
	} else if MReqReply == RR_NONPERS_NET2EX {
		ac.TpLogInfo("Non-persistent connections: Synchronous, Network requests Enduro/X")
		if MType != CON_TYPE_PASSIVE {
			ac.TpLogInfo("Connection type (type) must be Passive [%s], but got: %s!",
				CON_TYPE_PASSIVE, MType)
			return FAIL
		}

		if MPerZero > 0 {
			ac.TpLogError("Periodic zero (periodic_zero_msg) parameter"+
				" cannot be used with req_reply type %d", MReqReply)
			return FAIL
		}
	}

	MZeroStopwatch.Reset()

	//Init the maps...
	MConnectionsSimple = make(map[int64]*ExCon)
	MConnectionsComp = make(map[int64]*ExCon)
	MConWaiter = make(map[int64]*DataBlock)
	MCorrWaiter = make(map[string]*DataBlock)

	Mfreeconns = make(chan *ExCon, MMaxConnections*2)

	//Advertize Gateway service
	if err := ac.TpAdvertise(MGateway, MGateway, TCPGATE); err != nil {
		ac.TpLogError("Advertise failed %s", err)
		return FAIL
	}

	//Send infos that connections are closed
	if MStatussvc != "" {
		var i int64
		for i = 0; i < MMaxConnections; i++ {
			ac.TpLogInfo("Notify connection %d down", i)
			NotifyStatus(ac, i, FLAG_CON_DISCON)
		}
	}

	if err := ac.TpExtAddPeriodCB(MScanTime, Periodic); err != nil {
		ac.TpLogError("Advertise failed %d: %s", err.Code(), err.Message())
		return FAIL
	}

	//Run the listener
	if MType == CON_TYPE_PASSIVE {
		ac.TpLogInfo("Starting connection listener...")
		go PassiveConnectionListener()
	}

	return SUCCEED
}

//Server shutdown
//@param ac ATMI Context
func Uninit(ac *atmi.ATMICtx) {
	ac.TpLogWarn("Server is shutting down...")

	if nil != MPassiveLisener {
		ac.TpLogWarn("Closing connection listener")
		MPassiveLisener.Close()
	}

	//Close any open connection
	CloseAllConnections(ac)

	//We will close all atmi contexts, but we will not reply to them,
	//Better they think that it is time-out condition

	deInitPoll(ac, &MinXPool)
	deInitPoll(ac, &MoutXPool)

	ac.TpLogInfo("Shutdown complete")

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
		if err = ac.TpRun(Init, Uninit); nil != err {
			ac.TpLogError("Exit with failure")
			os.Exit(atmi.FAIL)
		} else {
			ac.TpLogInfo("Exit with success")
			os.Exit(atmi.SUCCEED)
		}
	}
}
