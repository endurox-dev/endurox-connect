/*
** Enduro/X -> World (OUT) Request handling...
**
** @file outreq.go
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
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

//Generate error that connection is not found
//@param buf	UBF buffer
func GenError(ac *atmi.ATMICtx, buf *atmi.TypedUBF, id_comp int64, code int, message string) {

	ac.Binit(buf, buf.BSizeof())

	if id_comp > 0 {
		buf.BChg(u.EX_NETCONNID, 0, id_comp)
	}

	buf.BChg(u.EX_NERROR_CODE, 0, code)
	buf.BChg(u.EX_NERROR_MSG, 0, message)
}

//Dispatcht the XATMI call (in own go routine)
//@param pool XATMI Pool
//@param nr	XATMI client number
//@param ctxData	Context data for request
//@param buf	ATMI buffer with request data
func XATMIDispatchCall(pool *XATMIPool, nr int, ctxData *atmi.TPSRVCTXDATA, buf *atmi.TypedUBF) {

	ret := SUCCEED
	ac := pool.ctxs[nr]
	var connid int64 = 0
	var corr string = ""

	defer func() {

		if SUCCEED == ret {
			buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply with SUCCEED")
			ac.TpReturn(atmi.SUCCEED, 0, buf, 0)
		} else {
			buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Reply with FAIL")
			ac.TpReturn(atmi.TPFAIL, 0, buf, 0)
		}
	}()

	ac.TpSrvSetCtxData(ctxData, 0)

	//OK so our context have a call, now do something with it

	connid, err := buf.BGetInt64(u.EX_NETCONNID, 0)
	corr, err := buf.BGetInt64(u.EX_NETCORR, 0)

	if RR_PERS_ASYNC_INCL_CORR == MReqReply || RR_PERS_CONN_EX2NET == MReqReply {
		if GetOpenConnectionCount() > 0 {
			//Get the connection to send message to
			/* If connection id specified, then get that one.. */
			var con *ExCon
			var block DataBlock
			var errA atmi.ATMIError

			block.data, errA = buf.BGetByteArr(u.EX_NETDATA, 0)

			if nil != errA {
				ac.TpLogError("Missing EX_NETDATA: %s!", errA.Message())
				//Reply with failure

				GenErrorNoConnection(ac, buf, atmi.NEMANDATORY,
					"Mandatory field EX_NETDATA missing!")
				ret = FAIL
				return

			}

			if connid == 0 {
				con = GetOpenConnection(ac)
			} else {
				con = GetConnectionByID(connid)
			}

			if nil == con {
				GenErrorNoConnection(ac, buf, atmi.NENOCONN,
					"No open connections available")
				ret = FAIL
				return
			}

			block.corr = corr
			block.atmi_out_conn_id = connid
			block.tstamp_sent = t.Unix()

			//Register in tables (if needed by config)
			if corr != "" {
				ac.TpLogInfo("Adding request to corr table, by "+
					"correlator: [%s]", corr)
				MCorrWaiterMutex.Lock()
				MCorrWaiter[corr] = block
				MCorrWaiterMutex.Unlock()
			}

			//If we work on sync way, only one data exchange over
			//The single channel, then lets add to id waiter list
			if MReqReply == RR_PERS_CONN_EX2NET {
				MConWaiterMutex.Lock()
				MConWaiter[con.id_comp] = block
				MConWaiterMutex.Unlock()
			}

			ac.TpLogWarn("About to send data...")
			con.outgoing <- block

			//If we are in correl or sync mode we need to wait data
			//block back...

			if corr != "" || MReqReply == RR_PERS_CONN_EX2NET {
				ac.TpLogWarn("Waiting for reply: correl [%s] "+
					"req_reply %d", corr, MReqReply)
				//Override the reply buffer
				//No more checks... as tout should be already generated.
				buf := <-block.atmi_chan

				ac.TpLogInfo("Got reply back")
			}
		} else {
			//Reply - no connection
			GenErrorNoConnection(ac, buf, atmi.NENOCONN,
				"No open connections available")
			ret = FAIL
			return
		}
	} else if RR_NONPERS_EX2NET == MReqReply {
		ac.TpLogInfo("Non persistent mode, one connection per message. " +
			"Try to connect")

		//So we are about to open channel, get the connection id
		//Add connection to compiled connection list as normal
		//get the connection and send stuff away. The connection Handler
		//should know already that conn must be closed by req_reply

		var con ExCon

		//1. Prepare connection block
		MConnMutex.Lock()
		con.id, con.id_stamp, con.id_comp = GetNewConnectionId()

		if con.id == FAIL {
			ac.TpLogError("Failed to get connection id - max reached?")
			ret = FAIL
			MConnMutex.Unlock()
		}

		//2. Add to hash

		MConnections[con.id] = &con
		MConnMutex.Unlock()

		//3. and spawn the routine...
        //TODO: We need to pass in here channel to which reply if
        //Connection did not succeed.
		go GoDial(&con)

		//4. Now try to send stuff out?

	}

	//Put back the channel
	pool.freechan <- nr
}
