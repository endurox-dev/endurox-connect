/*
** Network -> Enduro/X
**
** @file atmiout.go
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

	buf, err := ac.NewUBF(len(data) + 1024)
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

	if err = buf.Bchg(u.EX_NETDATA, 0, data); err != nil {
		ac.TpLogError("Failed to set EX_NETDATA %d: %s", err.Code(), err.Message())
		return nil, err
	}

	if "" != corr {
		if buf.BChg(u.EX_NETCORR, 0, corr); err != nil {
			ac.TpLogError("Failed to set EX_NETCORR %d: %s", err.Code(), err.Message())
			return nil, err
		}
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
func NetDispatchCall(ac *atmi.ATMICtx, con *ExCon, corr string, data []byte) {

	//Setup UBF buffer, load the fields
	buf, err := AllocReplyDataBuffer(ac, con, corr, data, false)

	if err != nil {
		ac.TpLogError("failed to create the net->ex UBF buffer: %s",
			err.Message())
		return
	}

	//OK we are here, lets call the service
	//If we work on non req_reply mode, then just async call

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming message")

	if !MReqReply {
		ac.TpLogInfo("Calling in async mode")
		_, err := ac.TpACall(MIncomingSvc, buf, atmi.TPNOREPLY)
	} else {
		ac.TpLogInfo("Req-reply mode enabled and this is incoming call, " +
			"do call the service in sync mode")

		_, err := ac.TpCall(MIncomingSvc, buf, 0)

		if err != nil {
			ac.TpLogError("Failed to call %s service: %d: %s",
				MIncomingSvc, err.Code(), err.Message())
			//Nothing to reply back
		} else {
			//Read the data block and reply back
			var b DataBlock
			b.data = buf.BGetByteArr(u.EX_NETDATA, 0)
			b.send_and_shut = true
			//Maybe send to channel for reply
			//And then shutdown
			//We need a send + shutdown channel...
			con.outgoing <- b
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
	block <- buf

	//We should shutdown the connection if this is request/reply mode
	//with out persistent connections
	if MReqReply == RR_NONPERS_EX2NET {

		ac.TpLogWarn("Non peristent connection mode, got answer from network"+
			" - requesting connection shutdown", a)
		*doContinue = false
	}
}

//Dispatch connection answer
//@param call 	Call data block (what caller thread actually made)
//@param data	Data block received from network
//@param bool	ptr for finish off parameter
func NetDispatchCorAnswer(ac *atmi.ATMICtx, con *ExCon, call *DataBlock, data []byte, doContinue *bool) {
	call.atmi_chan <- data //Send the data to caller
	//Remove from corelator lists
	RemoveFromCallLists(call)
}

//Get correlator id
func NetGetCorID(ac *atmi.ATMICtx, data []byte) (string, error) {

	return "", nil
}
