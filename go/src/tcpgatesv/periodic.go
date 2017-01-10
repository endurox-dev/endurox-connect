/*
** This module contains periodic callback processing
**
** @file periodic.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2015, ATR Baltic, SIA. All Rights Reserved.
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
** A commercial use license is available from ATR Baltic, SIA
** contact@atrbaltic.com
** -----------------------------------------------------------------------------
 */
package main

import (
	atmi "github.com/endurox-dev/endurox-go"
)

//TODO: Send period zero:

//Send zero length messages over the channels
func RunZeroOverOpenCons(ac *atmi.ATMICtx) {

	var zero_buf []byte
	//Lock all connections
	MConnMutex.Lock()

	for _, v := range MConnections {

		//Send the data block.
		v.outgoing <- zero_buf

	}

	MConnMutex.Unlock()

}

//Check the outgoint connections
func CheckDial(ac *atmi.ATMICtx) {

	//var openConns int64 = MMaxConnections - int64(len(MConnections))
	var i int64

	ac.TpLogInfo("CheckDial: Active connection, checking outgoing connections...")

	MConnMutex.Lock()
	for i = GetOpenConnectionCount(); i < MMaxConnections; i++ {

		//Spawn new connection threads
		var con ExCon

		//1. Prepare connection block
		con.id, con.id_stamp, con.id_comp = GetNewConnectionId()

		if con.id == FAIL {
			ac.TpLogError("Failed to get connection id - max reached?")
			break
		}

		//2. Add to hash
		MConnections[con.id] = &con

		//3. and spawn the routine...
		go GoDial(&con, nil)
	}

	MConnMutex.Unlock()
}

//Test is call block timed out
//@param v	Call block
//@return true - timed out, false - call not timed out
func IsBlockTimeout(ac *atmi.ATMICtx, v *DataBlock) bool {
	ac.TpLogDebug("Testing tout: tstamp_sent=%d, "+
		"MReqReplyTimeout=%d, sum=%d, current=%d",
		v.tstamp_sent, MReqReplyTimeout,
		v.tstamp_sent+MReqReplyTimeout, GetEpochMillis())
	if v.tstamp_sent+MReqReplyTimeout > GetEpochMillis() {
		ac.TpLogWarn("Call timed out!")
		return false
	}

	return true
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

		if v.corr != "" || MReqReply == RR_NONPERS_EX2NET ||
			MReqReply == RR_PERS_CONN_EX2NET {

			if IsBlockTimeout(ac, v) {
				buf, err := GenErrorUBF(ac, 0, atmi.NETOUT,
					"Timed out waiting for answer...")

				if nil == err {
					//Remove from list
					MConWaiter[k] = nil
					v.atmi_chan <- buf
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
					MCorrWaiter[k] = nil
					v.atmi_chan <- buf
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
	if MType == CON_TYPE_ACTIVE {
		CheckDial(ac)
	}

	if err := CheckTimeouts(ac); nil != err {

		ac.TpLogError("Failed check timeouts: %s - Aborting...",
			err.Message())
		return atmi.FAIL

	}

	//TODO: Check for any outstanding network calls...
	//Send the timeout message of tout got.
	//Close the connection if req/reply..

	if MShutdown == RUN_SHUTDOWN_FAIL {
		ac.TpLogWarn("Fail state shutdown requested! - Aborting...")
		ret = atmi.FAIL
	}

	return ret
}
