/**
 * @brief This module is responsible for connections handling
 *
 * @file tcpgatesv.go
 */
/* -----------------------------------------------------------------------------
 * Enduro/X Middleware Platform for Distributed Transaction Processing
 * Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
 * Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
 * This software is released under one of the following licenses:
 * AGPL or Mavimax's license for commercial use.
 * -----------------------------------------------------------------------------
 * AGPL license:
 *
 * This program is free software; you can redistribute it and/or modify it under
 * the terms of the GNU Affero General Public License, version 3 as published
 * by the Free Software Foundation;
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU Affero General Public License, version 3
 * for more details.
 *
 * You should have received a copy of the GNU Affero General Public License along
 * with this program; if not, write to the Free Software Foundation, Inc.,
 * 59 Temple Place, Suite 330, Boston, MA 02111-1307 USA
 *
 * -----------------------------------------------------------------------------
 * A commercial use license is available from Mavimax, Ltd
 * contact@mavimax.com
 * -----------------------------------------------------------------------------
 */
package main

import (
	"crypto/tls"
	"exutil"
	"fmt"
	"os"
	"strconv"
	"strings"

	"log"
	"os/signal"
	"runtime"
	"syscall"

	//	"runtime"
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
var MFramingLenReal int = len(MFraming)
var MFamingInclPfxLen bool = false //Does len format include prefix length it self?
var MFramingMaxMsgLen int = 0      //Max message len (checked if >0)
var MFramingHalfSwap bool = false  //Should we swap on the half incoming length bytes
var MFramingKeepHdr bool = false   //Should we keep the len header?
//This does count in the header
var MFramingOffset int = 0 //Number of bytes to ignore after which header follows

//In case if framing is "d"
var MDelimStart byte = 0x02       //can be optional
var MDelimStop byte = 0x03        //Can be optional
var MType string = "P"            //A - Active, P - Pasive
var MIp string = "0.0.0.0"        //IP to listen or to connect to if Active
var MPort int = 7921              //Port to connect to or listen on depending on active/passive role
var MAddr string = ""             //Compiled ip:port
var MIncomingSvc string = ""      //Incomding service to send to incoming async traffic
var MIncomingSvcSync bool = false //Is incoming service Synchronous and needs tpcall with rsp back to net
var MPerZero int = 0              //Period by witch to which send zero length message to all channels...
var MStatussvc string = ""        //Status service to which send connection information
var MStatusRefresh int = 0        //Send periodic status refreshes, seconds
//Max number to connection to connect to server, or allow max incomings in the same time.
var MMaxConnections int64 = 5

//Request reply model, alos for in-out, sync mode
//Open connection for incoming wait for reply,
//and Close the connection .
var MReqReply int = RR_PERS_ASYNC_INCL_CORR

//Timeout for req-reply model
var MReqReplyTimeout int64 = 60
var MConnWaitTime int64 = 60 //Max time to wait for connection in pool
var MInIdleMax int64 = 0     //By default no connection restart
var MInIdleCheck int64 = 0   //Time into which check idle seconds
var MScanTime = 1            //Seconds for housekeeping

var MSeqOut bool = false //Use outgoing sequence of messages (by connection number)
//Basically process incoming messages in single threaded mode as no reason to put in queue
//We shall wait for service to complete to serve next message anyway
var MSeqIn bool = false

//Correlator service for incoming messages
//This is used case if driver operates in sync mode over the persistently conneced lines
var MCorrSvc = ""

var MShutdown int = RUN_CONTINUE

var MActiveConScan int = 5 //scan for new outgoing connections every 10 seconds

var MLinger int = -1 //Set linger <0 is default OS setting

/* TLS Settings: */
var MTls_enable bool                    //Is TLS enabled
var MTls_skip_verify bool               //Ignore non verified connections (continue OK)
var MTls_cert_file string               //Client/Server certificate
var MTls_key_file string                //Key file
var MTls_ca_roots string                //Semicolon seperated root certificates
var MTls_client_auth tls.ClientAuthType //Is client auth required?
var MTls_min_version uint16             //minum tls version

//Resolved TLS settings
var MTls_certificate tls.Certificate
var MTls_config tls.Config

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
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Resize buffer, to have some more space
	buf_size, err := ub.BUsed()

	if err != nil {
		ac.TpLogError("Failed to get incoming buffer used space: %d:%s",
			err.Code(), err.Message())
		ret = FAIL
		return
	}

	//Realloc to have some free space for buffer manipulations
	if err := ub.TpRealloc(buf_size + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//Pack the request data to pass to thread
	ctxData, err := ac.TpSrvGetCtxData()
	if nil != err {
		ac.TpLogError("Failed to get context data - dropping request %d:%s",
			err.Code(), err.Message())
		ret = FAIL
		return
	}

	ac.TpLogInfo("Waiting for free XATMI out object")
	nr := getFreeXChan(ac, &MoutXPool)
	ac.TpLogInfo("Got XATMI out object")

	//If connection ID is present, then serialise all
	//in queue. We could make somehow chained go routines.

	//we shall create dynamic queue. if there is already started with
	//Q then it shall finish it...
	if MSeqOut && ub.BPres(u.EX_NETCONNID, 0) {

		id, err := ub.BGetInt64(u.EX_NETCONNID, 0)

		if nil != err {
			ac.TpLogError("Failed to get EX_NETCONNID - dropping request: %d:%s",
				err.Code(), err.Message())
			ret = FAIL
			return
		}
		//Run sequencing
		//Firstly we need to understand is this first message in queue or
		//there is queue already in progress...
		XATMIDispatchCallSeq(id, &MoutXPool, nr, ctxData, ub, svc.Cd)

	} else {
		//No sequencing...
		go XATMIDispatchCall(&MoutXPool, nr, ctxData, ub, svc.Cd, true)
	}

	//XATMIDispatchCall(&MoutXPool, nr, ctxData, ub, svc.Cd, true)

	//runtime.GC()
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
				C.signal(6, nil)
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
		case "framing_half_swap":
			tmpSwap, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if 1 == tmpSwap {
				ac.TpLogInfo("Will swap framing bytes in half")
				MFramingHalfSwap = true
			}
		case "seqout":
			tmpseq, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if 1 == tmpseq {
				ac.TpLogInfo("Outoing messages be sent in sequence")
				MSeqOut = true
			}
		case "seqin":
			tmpseq, _ := buf.BGetInt(u.EX_CC_VALUE, occ)

			if 1 == tmpseq {
				ac.TpLogInfo("Incoming message from network will be in sequence")
				MSeqIn = true
			}

		case "max_msg_len":
			MFramingMaxMsgLen, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MFramingMaxMsgLen)
			break
		case "delim_start":
			tmpDelimStart, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, tmpDelimStart)
			cleaned := strings.Replace(tmpDelimStart, "0x", "", -1)
			val, err := strconv.ParseUint(cleaned, 16, 64)

			if err != nil {
				ac.TpLogError("Failed to parse delim_start hex string: %s",
					err.Error())
				return atmi.FAIL
			}
			MDelimStart = byte(val)
			ac.TpLogInfo("etx=[%x]", MDelimStart)
			break
		case "delim_stop":
			tmpDelimStop, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%b] ", fldName, tmpDelimStop)
			cleaned := strings.Replace(tmpDelimStop, "0x", "", -1)
			val, err := strconv.ParseUint(cleaned, 16, 64)

			if err != nil {
				ac.TpLogError("Failed to parse delim_stop hex string: %s",
					err.Error())
				return atmi.FAIL
			}
			MDelimStop = byte(val)
			ac.TpLogInfo("etx=[%x]", MDelimStop)
			break
		case "framing_offset":

			MFramingOffset, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MFramingOffset)

			if MFramingOffset < 0 {
				ac.TpLogError("Invalid framing offset, must be >=0, but: %d",
					MFramingOffset)
				return atmi.FAIL
			}

			MFramingKeepHdr = true
			break
		case "framing_keephdr":

			tmp, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, tmp)

			if "Y" == string(tmp[0]) || "y" == string(tmp[0]) {
				MFramingKeepHdr = true
			}

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
		case "incoming_svc_sync":

			tmp, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, tmp)

			if "Y" == string(tmp[0]) || "y" == string(tmp[0]) {
				MIncomingSvcSync = true
			}

			break
		case "periodic_zero_msg":
			MPerZero, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MPerZero)
			break
		case "status_svc":
			MStatussvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MStatussvc)
			break
		case "status_refresh":
			MStatusRefresh, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MStatusRefresh)
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
		case "conn_wait_time":
			//Max time to wait for connection
			MConnWaitTime, _ = buf.BGetInt64(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MConnWaitTime)
			break
		case "in_idle_max":
			//Restart connection if in idle (no incoming traffice) for more than
			//given seconds
			MInIdleMax, _ = buf.BGetInt64(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MInIdleMax)
			break
		case "in_idle_check":
			//Number in seconds into which to check the connection idle time
			MInIdleCheck, _ = buf.BGetInt64(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MInIdleCheck)
			break
		case "corr_svc":
			//Corelator service for sync tpcall over mulitple persistent connectinos
			MCorrSvc, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, MCorrSvc)
			break
		case "debug":
			//Set debug configuration string
			debug, _ := buf.BGetString(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%s] ", fldName, debug)
			if err := ac.TpLogConfig((atmi.LOG_FACILITY_NDRX | atmi.LOG_FACILITY_UBF | atmi.LOG_FACILITY_TP),
				-1, debug, "TCPG", ""); nil != err {
				ac.TpLogError("Invalid debug config [%s] %d:[%s]\n",
					debug, err.Code(), err.Message())
				return FAIL
			}
			break
		case "linger":
			MLinger, _ = buf.BGetInt(u.EX_CC_VALUE, occ)
			ac.TpLogDebug("Got [%s] = [%d] ", fldName, MLinger)
			break

		case "tls_enable":
			val, _ := buf.BGetInt16(u.EX_CC_VALUE, occ)
			if 1 == val {
				MTls_enable = true
			}

		case "tls_skip_verify":
			val, _ := buf.BGetInt16(u.EX_CC_VALUE, occ)
			if 1 == val {
				MTls_skip_verify = true
			}
			break
		case "tls_cert_file":
			MTls_cert_file, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "tls_key_file":
			MTls_key_file, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "tls_ca_roots":
			MTls_ca_roots, _ = buf.BGetString(u.EX_CC_VALUE, occ)
			break
		case "tls_client_auth":

			val, _ := buf.BGetInt16(u.EX_CC_VALUE, occ)
			if 1 == val {
				MTls_client_auth = tls.RequireAndVerifyClientCert
			}

			break
		case "tls_min_version":

			min_ver, _ := buf.BGetString(u.EX_CC_VALUE, occ)

			switch min_ver {
			case "TLS10":
				MTls_min_version = tls.VersionTLS10
			case "TLS11":
				MTls_min_version = tls.VersionTLS11
			case "TLS12":
				MTls_min_version = tls.VersionTLS12
			default:
				ac.TpLogError("Invalid tls_min_version [%s], " +
					"expected: TLS10,TLS11,TLS12")
				return FAIL
			}
			break

		//do not handle the connection pool (if always work by connection id...)
		/* NOT USED ANYMORE! Just round robin over the connections...
		case "nofreelist":
			tmp, _ := buf.BGetString(u.EX_CC_VALUE, occ)

			ac.TpLogDebug("Got [%s] = [%s] ", fldName, tmp)

			if "Y" == string(tmp[0]) || "y" == string(tmp[0]) {
				MNofreelist = true
			}
			break
		*/
		default:

			break
		}
	}

	MReqReplyTimeout *= 1000 //Convert to millis

	MinXPool.nrWorkers = MworkersIn
	MoutXPool.nrWorkers = MWorkersOut

	MSeqOutMsgs = make(map[int64][]*ATMIOutBlock)

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

	if MFramingHalfSwap && MFramingLen%2 > 0 {
		ac.TpLogWarn("Using half swap of framing bytes, but byte length is odd: %d",
			MFramingLen)
		return FAIL
	}

	//Check the idle time settings
	if MInIdleCheck > 0 && MInIdleMax <= 0 || MInIdleCheck <= 0 && MInIdleMax > 0 {
		ac.TpLogError("ERROR: paramters 'in_idle_check' (%d) and 'in_idle_max'(%d) "+
			"both must be either 0 or greater than 0", MInIdleCheck, MInIdleMax)
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

		if MIncomingSvcSync {
			ac.TpLogInfo("Incoming service: [%s] - sync", MIncomingSvc)
		} else {
			ac.TpLogInfo("Incoming service: [%s] - async", MIncomingSvc)
		}

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

		if int64(MWorkersOut) > MMaxConnections {
			ac.TpLogError("In request/reply mode %d `workers_out' "+
				"must be equal or lower to `max_connections', but "+
				"got workers_out (%d) > max_connections (%d). "+
				"Recommended: max_connections=workers_out*2",
				MReqReply, MWorkersOut, MMaxConnections)
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

	//mvitolin 18/03/2017 #97
	if MStatusRefresh > 0 {

		if MReqReply != RR_PERS_ASYNC_INCL_CORR && MReqReply != RR_PERS_CONN_EX2NET &&
			MReqReply != RR_PERS_CONN_NET2EX {

			ac.TpLogError("`status_refresh' valid only for persistent connections, "+
				"`req_reply' modes: %d/%d/%d",
				RR_PERS_ASYNC_INCL_CORR,
				RR_PERS_CONN_EX2NET,
				RR_PERS_CONN_NET2EX)

			return FAIL
		}

		if MStatussvc == "" {
			ac.TpLogError("For `status_refresh' `status_svc' must be defined!")
			return FAIL
		}
	}

	ac.TpLogInfo("Keep framing header: %t", MFramingKeepHdr)
	ac.TpLogInfo("Framing offset: %d", MFramingOffset)
	ac.TpLogInfo("Periodic status broadcast: %d", MStatusRefresh)

	if errS := ConfigureNumberOfBytes(ac); errS != nil {
		ac.TpLogError("Failed to configure number of bytes to use for "+
			"message frame: %s", errS.Error())
		return FAIL
	}

	// print TLS config:
	ac.TpLogInfo("TLS Enable: %v", MTls_enable)
	ac.TpLogInfo("TLS Skip Verify: %v", MTls_skip_verify)
	ac.TpLogInfo("TLS Key file: %v", MTls_key_file)
	ac.TpLogInfo("TLS Cert file: %v", MTls_cert_file)
	ac.TpLogInfo("TLS CA Roots: %v", MTls_ca_roots)
	ac.TpLogInfo("TLS Client Auth: %v", MTls_client_auth)
	ac.TpLogInfo("TLS Min version: %v", MTls_min_version)

	//Load roots if any...
	if MTls_ca_roots != "" && MTls_enable {
		if err := exutil.LoadRootCAs(ac, MTls_ca_roots); nil != err {
			ac.TpLogError("Failed to load CA roots: %s", err.Error())
			return FAIL
		}
	}

	if MTls_enable {

		var certs []tls.Certificate

		if MTls_key_file != "" && MTls_cert_file == "" ||
			MTls_key_file == "" && MTls_cert_file != "" {
			ac.TpLogError("TLS certificate must have settings: " +
				"tls_cert_file and tls_key_file (one is missing)")
			return FAIL
		}

		//Load certficate
		if MTls_key_file != "" {

			var err error
			MTls_certificate, err = tls.LoadX509KeyPair(MTls_cert_file, MTls_key_file)
			if err != nil {
				ac.TpLogError("Failed to load TLS certificate: %s", err.Error())
				return FAIL
			}

			certs = append(certs, MTls_certificate)
		}

		if CON_TYPE_PASSIVE == MType && MTls_key_file == "" {
			ac.TpLogError("ERROR: For passive (server) connection role " +
				"certificate is mandatory (tls_cert_file/tls_key_file)")
			return FAIL
		}

		//Prepare config object
		MTls_config = tls.Config{Certificates: certs,
			RootCAs:            exutil.MRootCAs,
			ClientCAs:          exutil.MRootCAs,
			InsecureSkipVerify: MTls_skip_verify,
			ClientAuth:         MTls_client_auth,
			MinVersion:         MTls_min_version}

		ac.TpLogInfo("TLS Configured")
	}

	//MZeroStopwatch.Reset()
	MStatusRefreshStopWatch.Reset()

	//Init the maps...
	MConnectionsSimple = make(map[int64]*ExCon)
	MConnectionsComp = make(map[int64]*ExCon)
	MConWaiter = make(map[int64]*DataBlock)
	MCorrWaiter = make(map[string]*DataBlock)

	//	Mfreeconns = make(chan *ExCon, MMaxConnections*2)
	//	MSeqNotif = make(chan bool, MWorkersOut*2)

	//Advertize Gateway service
	if err := ac.TpAdvertise(MGateway, MGateway, TCPGATE); err != nil {
		ac.TpLogError("Advertise failed %s", err)
		return FAIL
	}

	//Send infos that connections are closed
	if MReqReply == RR_PERS_ASYNC_INCL_CORR || MReqReply == RR_PERS_CONN_EX2NET ||
		MReqReply == RR_PERS_CONN_NET2EX {
		if MStatussvc != "" {
			var i int64
			for i = 1; i <= MMaxConnections; i++ {
				ac.TpLogInfo("Notify connection %d down", i)
				NotifyStatus(ac, i, atmi.FAIL, FLAG_CON_DISCON, nil)
			}
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

	ac.TpLogWarn("Startup finished")

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

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGQUIT)
		buf := make([]byte, 1<<20)
		for {
			<-sigs
			stacklen := runtime.Stack(buf, true)
			log.Printf("=== received SIGQUIT ===\n*** goroutine dump...\n%s\n*** end\n", buf[:stacklen])
		}
	}()

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

/* vim: set ts=4 sw=4 et smartindent: */
