/**
 * @brief Network -> Enduro/X
 *
 * @file netin.go
 */
/* -----------------------------------------------------------------------------
 * Enduro/X Middleware Platform for Distributed Transaction Processing
 * Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
 * Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
 * This software is released under one of the following licenses:
 * GPL or Mavimax's license for commercial use.
 * -----------------------------------------------------------------------------
 * GPL license:
 * 
 * This program is free software; you can redistribute it and/or modify it under
 * the terms of the GNU General Public License as published by the Free Software
 * Foundation; either version 3 of the License, or (at your option) any later
 * version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with
 * this program; if not, write to the Free Software Foundation, Inc., 59 Temple
 * Place, Suite 330, Boston, MA 02111-1307 USA
 *
 * -----------------------------------------------------------------------------
 * A commercial use license is available from Mavimax, Ltd
 * contact@mavimax.com
 * -----------------------------------------------------------------------------
 */
package main

import (
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

//Allocate UBF buffer for holding the full
//Hmm buf we have a problem here with error, as the interface appears to be the
//same for UBF and ATMI, and also error codes to collide. But for now it is not
//important.
//@param con	Connection object
//@param corr	Correlator
//@param data	Data received from network
//@param isRsp	true if request is response, else it is assumed as request
//@return UBF buffer if no error, ATMI Error if problem occurred.
func AllocReplyDataBuffer(ac *atmi.ATMICtx, con *ExCon, corr string, data []byte, isRsp bool) (*atmi.TypedUBF, atmi.ATMIError) {

	buf, err := ac.NewUBF(int64(len(data) + 1024))
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s] - dropping incoming message",
			err.Error())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETGATEWAY, 0, MGateway); err != nil {
		ac.TpLogError("Failed to set EX_NETGATEWAY %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETCONNID, 0, con.id_comp); err != nil {
		ac.TpLogError("Failed to set EX_NETCONNID %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETDATA, 0, data); err != nil {
		ac.TpLogError("Failed to set EX_NETDATA %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if "" != corr {
		if buf.BChg(u.EX_NETCORR, 0, corr); err != nil {
			ac.TpLogError("Failed to set EX_NETCORR %d: %s", err.Code(), err.Message())
			return nil, err
		}
	}

	//Setup IP/port our/their and role
	if err = buf.BChg(u.EX_NETOURIP, 0, con.ourip); err != nil {
		ac.TpLogError("Failed to set EX_NETOURIP %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETOURPORT, 0, con.outport); err != nil {
		ac.TpLogError("Failed to set EX_NETOURPORT %d: %s", err.Code(), err.Message())
		return nil, err
	}

	//Setup IP/port our/their and role
	if err = buf.BChg(u.EX_NETTHEIRIP, 0, con.theirip); err != nil {
		ac.TpLogError("Failed to set EX_NETTHEIRIP %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETTHEIRPORT, 0, con.theirport); err != nil {
		ac.TpLogError("Failed to set EX_NETTHEIRPORT %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if err = buf.BChg(u.EX_NETCONMODE, 0, con.conmode); err != nil {
		ac.TpLogError("Failed to set EX_NETCONMODE %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if isRsp {
		buf.BChg(u.EX_NERROR_CODE, 0, 0)
		buf.BChg(u.EX_NERROR_MSG, 0, "SUCCEED")
	}

	return buf, nil
}

//We have recieved new call from Network
//So shall wait for new ATMI context & send the message in
//This should be run on go routine.
//@param data 	Data received from Network
//@param bool	set to false if do not need to continue (i.e. close conn)
func NetDispatchCall(pool *XATMIPool, nr int, con *ExCon,
	preAllocUBF *atmi.TypedUBF, corr string, data []byte) {

	buf := preAllocUBF
	ac := pool.ctxs[nr]

	//Return to the caller
	defer func() {
		ac.TpLogInfo("About to put back XATMI-in object %d", nr)
		//put batch in channel
		pool.freechan <- nr
	}()

	var errA atmi.ATMIError
	//Setup UBF buffer, load the fields
	if nil == buf {
		buf, errA = AllocReplyDataBuffer(ac, con, corr, data, false)

		if errA != nil {
			ac.TpLogError("failed to create the net->ex UBF buffer: %s",
				errA.Message())
			return
		}
	} else {
		//Set the current context of the buffer
		buf.GetBuf().TpSetCtxt(ac)
	}

	//OK we are here, lets call the service
	//If we work on non req_reply mode, then just async call

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming message")

	//Full async mode
	// Feature #204 - allow in full async mode sync invopcation of incomings...
	if (RR_PERS_ASYNC_INCL_CORR == MReqReply ||
		RR_PERS_CONN_EX2NET == MReqReply) && !MIncomingSvcSync {
		ac.TpLogInfo("Calling in async mode (fully async or conn_ex2net mode)")
		_, errA := ac.TpACall(MIncomingSvc, buf, atmi.TPNOREPLY)

		if nil != errA {
			ac.TpLogError("Failed to acall [%s]: %s",
				MIncomingSvc, errA.Message())
		}
	} else {
		ac.TpLogInfo("Req-reply mode enabled and this is incoming call, " +
			"do call the service in sync mode")

		_, errA := ac.TpCall(MIncomingSvc, buf, 0)

		if errA != nil {
			ac.TpLogError("Failed to call %s service: %d: %s",
				MIncomingSvc, errA.Code(), errA.Message())
			//Nothing to reply back
		} else {
			//Read the data block and reply back
			var b DataBlock
			b.data, errA = buf.BGetByteArr(u.EX_NETDATA, 0)
			if nil != errA {
				ac.TpLogError("Protocol error: failed to get "+
					"EX_NETDATA: %s", errA)
				//Shutdonw the sync incoming connection only
				//If needed (i.e. if it one connection per request)
				con.shutdown <- true

			} else {
				ac.TpLogInfo("Got message from EX, sending to net len: %d",
					len(b.data))
				//Maybe send to channel for reply
				//And then shutdown (if needed, will by done by con it self)
				//How about locking, connection is already locked!!!!
				ac.TpLogDebug("No lock mode")
				b.nolock = true
				con.outgoing <- &b
			}
		}
	}
}

//Dispatch connection answer

//@param call 	Call data block (what caller thread actually made)
//@param data	Data block received from network
//@param bool	ptr for finish off parameter
func NetDispatchConAnswer(ac *atmi.ATMICtx, con *ExCon, block *DataBlock, data []byte, doContinue *bool) {

	//Setup UBF buffer, load the fields
	buf, err := AllocReplyDataBuffer(ac, con, "", data, true)

	if err != nil {
		ac.TpLogError("failed to create the net->ex UBF buffer: %s",
			err.Message())
		return
	}

	//Network answer on connection
	block.atmi_chan <- buf

	//We should shutdown the connection if this is request/reply mode
	//with out persistent connections
	if MReqReply == RR_NONPERS_EX2NET {

		ac.TpLogWarn("Non peristent connection mode, got answer from network" +
			" - requesting connection shutdown")
		*doContinue = false
	}
}

//Dispatch connection answer
//@param call 	Call data block (what caller thread actually made)
//@param data	Data block received from network
//@param bool	ptr for finish off parameter
func NetDispatchCorAnswer(ac *atmi.ATMICtx, con *ExCon, block *DataBlock,
	buf *atmi.TypedUBF, doContinue *bool) {
	ac.TpLogInfo("Doing reply to correlated ex->net call")
	block.atmi_chan <- buf //Send the data to caller
}

//Get correlator id from incoming message. The correlator is set in UBF buffer
//@param ac	ATMI Context
//@param buf	ATMI buffer
//@return 	ATMI error if fail, or nil if all ok
func NetGetCorID(ac *atmi.ATMICtx, buf *atmi.TypedUBF) (string, atmi.ATMIError) {

	_, err := ac.TpCall(MCorrSvc, buf, 0)

	if nil != err {
		ac.TpLogError("Failed to call [%s] service: %s",
			MCorrSvc, err.Message())
		return "", err
	}

	ret, _ := buf.BGetString(u.EX_NETCORR, 0)
	ac.TpLogInfo("Got correlation from service: [%s]", ret)

	return ret, nil
}
/* vim: set ts=4 sw=4 et smartindent: */
