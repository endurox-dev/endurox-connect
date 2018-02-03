/*
** This module contains periodic callback processing
**
** @file periodic.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2015, ATR Baltic, Ltd. All Rights Reserved.
** This software is released under one of the following licenses:
** GPL or ATR Baltic's license for commercial use.
** -----------------------------------------------------------------------------
** GPL license:
**
** This program is free software; you can redistribute it and/or modify it under
** the terms of the GNU General Public License as published by the Free Software
** Foundation; either version 2 of the License, or (at your option) any later
** version.
**
** This program is distributed in the hope that it will be useful, but WITHOUT ANY
** WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
** PARTICULAR PURPOSE. See the GNU General Public License for more details.
**
** You should have received a copy of the GNU General Public License along with
** this program; if not, write to the Free Software Foundation, Inc., 59 Temple
** Place, Suite 330, Boston, MA 02111-1307 USA
**
** -----------------------------------------------------------------------------
** A commercial use license is available from ATR Baltic, Ltd
** contact@atrbaltic.com
** -----------------------------------------------------------------------------
 */
package main

import (
	"exutil"

	atmi "github.com/endurox-dev/endurox-go"
)

//Zero sending periodic stopwatch
var MZeroStopwatch exutil.StopWatch
var MStatusRefreshStopWatch exutil.StopWatch
var MInIdleCheckStopWatch exutil.StopWatch

//Send zero length messages over the channels
func RunZeroOverOpenCons(ac *atmi.ATMICtx) {

	//Lock all connections
	MConnMutex.Lock()

	for _, v := range MConnectionsComp {

		if v.is_open {
			var block DataBlock

			if MFramingOffset > 0 {
				//Go by default set to 0
				block.data = make([]byte, MFramingLen)
			}

			p_block := &block
			ac.TpLogInfo("Sending zero length message to id:%d conn_id: %d ",
				v.id, v.id_comp)

			//Remove connection from free list
			MarkConnAsBusy(ac, v)

			//Send the data block.
			v.outgoing <- p_block
		} else {
			ac.TpLogInfo("conn %d/%d is not yet open - not sending zero msg",
				v.id, v.id_comp)
		}
	}

	MConnMutex.Unlock()

}

//Check activity over open conns
//So that if we have not received anyting
func RunCheckInIdleChk(ac *atmi.ATMICtx) {

	//Lock all connections
	MConnMutex.Lock()
	for _, v := range MConnectionsComp {

		if v.is_open {

			spent := v.inIdle.GetDetlaSec()
			if spent > MInIdleMax {
				ac.TpLogWarn("RESET: Connect %d/%d not received any data in %d sec "+
					"(already spent: %d sec) - resetting conn",
					v.id, v.id_comp, MInIdleMax, spent)
				//Close connection
				v.shutdown <- true
			}

		} else {
			ac.TpLogInfo("conn %d/%d is not yet open - not sending zero msg",
				v.id, v.id_comp)
		}
	}

	MConnMutex.Unlock()
}

//Send zero length messages over the channels
func RunStatusRefresh(ac *atmi.ATMICtx) {

	//Lock all connections
	MConnMutex.Lock()
	var i int64

	for i = 1; i <= MMaxConnections; i++ {

		if nil != MConnectionsSimple[i] {
			ac.TpLogInfo("REFRESH: Notify connection %d UP", i)
			NotifyStatus(ac, i, MConnectionsSimple[i].id_comp, FLAG_CON_ESTABLISHED)
		} else {
			ac.TpLogInfo("REFRESH: Notify connection %d DOWN", i)
			NotifyStatus(ac, i, atmi.FAIL, FLAG_CON_DISCON)
		}
	}

	MConnMutex.Unlock()

}

//Check the outgoint connections
func CheckDial(ac *atmi.ATMICtx) {

	//var openConns int64 = MMaxConnections - int64(len(MConnections))
	var i int64

	ac.TpLogInfo("CheckDial: Active connection, checking outgoing connections...")

	for i = GetOpenConnectionCount(); i < MMaxConnections; i++ {

		//Spawn new connection threads
		var con ExCon

		SetupConnection(&con)

		//1. Prepare connection block
		MConnMutex.Lock()
		con.id, con.id_stamp, con.id_comp = GetNewConnectionId(ac)

		if con.id == FAIL {
			ac.TpLogError("Failed to get connection id - max reached?")
			MConnMutex.Unlock()
			break
		}

		//2. Add to hash, -- why not 2017/09/29 - we get the same connection ids...
		//when they are not connect but new ids are generated?
		/*
			mvitolin 2017/01/25 do it when connection is established in GoDial*/
		MConnectionsSimple[con.id] = &con
		MConnectionsComp[con.id_comp] = &con

		MConnMutex.Unlock()

		//3. and spawn the routine...
		go GoDial(&con, nil)
	}

}

//Test is call block timed out
//@param v	Call block
//@return true - timed out, false - call not timed out
func IsBlockTimeout(ac *atmi.ATMICtx, v *DataBlock) bool {

	cur := exutil.GetEpochMillis()
	sum := v.tstamp_sent + MReqReplyTimeout
	ac.TpLogDebug("Testing tout: tstamp_sent=%d, "+
		"MReqReplyTimeout=%d, sum=%d, current=%d, delta=%d",
		v.tstamp_sent, MReqReplyTimeout,
		sum, cur,
		(cur - v.tstamp_sent))

	if sum < cur {
		ac.TpLogWarn("Call timed out!")
		return true
	}

	return false
}

//Check the connection timeouts
//if needed generate timeout-response
//and repond to service. Remove from waiter list
//if timed out
//@param ac 	ATMI Context
func CheckTimeouts(ac *atmi.ATMICtx) atmi.ATMIError {

	//Lock the channels
	//The message shall not appear in both list correlated & by connection
	MConWaiterMutex.Lock()
	ac.TpLogDebug("Checking sync connection lists for timeouts")
	for k, v := range MConWaiter {

		if MReqReply == RR_PERS_CONN_EX2NET || MReqReply == RR_PERS_CONN_NET2EX ||
			MReqReply == RR_NONPERS_EX2NET || MReqReply == RR_NONPERS_NET2EX {

			if IsBlockTimeout(ac, v) {
				ac.TpLogWarn("Call expired!")
				buf, err := GenErrorUBF(ac, 0, atmi.NETOUT,
					"Timed out waiting for answer...")

				if nil == err {
					//Remove from list
					delete(MConWaiter, k)
					ac.TpLogInfo("Sending reply back to ATMI")
					v.atmi_chan <- buf
					ac.TpLogInfo("Sending reply back to ATMI, done")

					//Will kill a connection
					//Because the other end will might sent reply
					//later and that will confuse next caller.
					ac.TpLogInfo("Killing connection")
					ac.TpLogDebug("v=%p", v)
					ac.TpLogDebug("v.con=%p", v.con)

					v.con.shutdown <- true
					ac.TpLogInfo("Killing connection, done")

				} else {
					MConWaiterMutex.Unlock()
					return err
				}
			}
		}
	}
	MConWaiterMutex.Unlock()

	MCorrWaiterMutex.Lock()
	ac.TpLogDebug("Checking async correlation connection lists for timeout")
	for k, v := range MCorrWaiter {

		if v.corr != "" || MReqReply == RR_NONPERS_EX2NET ||
			MReqReply == RR_PERS_CONN_EX2NET {

			if IsBlockTimeout(ac, v) {
				buf, err := GenErrorUBF(ac, 0, atmi.NETOUT,
					"Timed out waiting for answer...")

				if nil == err {
					//Remove from list
					delete(MCorrWaiter, k)
					ac.TpLogInfo("Sending reply back to ATMI")
					v.atmi_chan <- buf
					ac.TpLogInfo("Sending reply back to ATMI, done")

					//Kill the connection, if non persistent
					if MReqReply == RR_NONPERS_EX2NET ||
						MReqReply == RR_PERS_CONN_EX2NET {
						ac.TpLogInfo("Killing connection")
						ac.TpLogDebug("v=%p", v)
						ac.TpLogDebug("v.con=%p", v.con)
						v.con.shutdown <- true
						ac.TpLogInfo("Killing connection, done")
					}

				} else {
					MCorrWaiterMutex.Unlock()
					return err
				}
			}
		}
	}
	MCorrWaiterMutex.Unlock()

	return nil

}

//Periodic callback function
//Hmm do we have some context here?
//We will spawn connections here..
func Periodic(ac *atmi.ATMICtx) int {

	ret := atmi.SUCCEED
	//if we are active, check that we have enought connections
	if MType == CON_TYPE_ACTIVE && (MReqReply == RR_PERS_ASYNC_INCL_CORR ||
		MReqReply == RR_PERS_CONN_EX2NET) {
		CheckDial(ac)
	}

	if err := CheckTimeouts(ac); nil != err {

		ac.TpLogError("Failed check timeouts: %s - Aborting...",
			err.Message())
		return atmi.FAIL

	}

	//Send the zero length messages to network...
	if MPerZero > 0 && MZeroStopwatch.GetDetlaSec() > int64(MPerZero) {
		ac.TpLogDebug("Time for periodic zero message over " +
			"the active connections")
		RunZeroOverOpenCons(ac)

		MZeroStopwatch.Reset()
	}

	if MStatusRefresh > 0 && MStatusRefreshStopWatch.GetDetlaSec() > int64(MStatusRefresh) {
		ac.TpLogInfo("Time for status refresh messages to be sent...")

		RunStatusRefresh(ac)
		MStatusRefreshStopWatch.Reset()

	}

	//Check the idle time incoming activitiy, if no messages received in given
	//time frame then connection is reset
	if MInIdleCheck > 0 && MInIdleCheckStopWatch.GetDetlaSec() > MInIdleCheck {
		ac.TpLogInfo("Time for idle connection checks with no incomming traffic")

		RunCheckInIdleChk(ac)
		MInIdleCheckStopWatch.Reset()

	}

	//TODO: Check for any outstanding network calls...
	//Send the timeout message of tout got.
	//Close the connection if req/reply..

	if MShutdown == RUN_SHUTDOWN_FAIL {
		//Hmm does not cause shutdown!!!
		ac.TpLogWarn("Fail state shutdown requested! - Aborting...")
		ret = atmi.FAIL
	}

	return ret
}
