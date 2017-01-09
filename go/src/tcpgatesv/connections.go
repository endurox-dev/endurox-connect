/*
** This module is responsible for connections handling
**
** @file connections.go
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
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	atmi "github.com/endurox-dev/endurox-go"
)

//About incoming & outgoing messages:

//If we run new connection per request + close, then we need:
//1. Open new connection
//2. Send the call
//3. Register in Per connection id waiting list of the replies channels
//4. Once we got the incoming message, we check the list 3, if message connection is registered
//5. We send the reply to the specified channel and connection gets closed
//6. If timeout occurrs we shall send to thread back info that this is timeout
//   so that it can clean up the resources.
//6.1. On tout we shall close connection too.

//If we run in request & reply mode, but we have few permanent connections
//then
//1. Once doing call, we need corelator string
//2. We add corelator with back channel to goroutine waiting for reply
//3. If timeout occurs, then special scanner thread should send back notification of tout
//4. If we get back response, we shall call MCorrSvc service with message dump, the service
//5. shall provide us with "EX_TCPCORR"
//6. If EX_TCPCORR is found in hash list then do the reply back to specified channel
//7. If EX_TCPCORR is not provided back, then send message to MIncomingSvc

//This is data block for sending messages int/out
type DataBlock struct {
	data            []byte
	addToConWaiter  bool
	addToCorrWaiter bool
	//sender_chan //optional if we want recieve reply back
	/* atmi_chan        chan []byte */
	atmi_chan        chan *atmi.TypedUBF
	atmi_out_conn_id int64  //Connection id if specified (0) - then random.
	corr             string //Correlator string (opt)
	net_conn_id      int64  //Network connection id (when sending in)

	tstamp_sent   int64 //Timestamp messag sent, TODO: We need cleanup monitor...
	send_and_shut bool  //Send and shutdown
}

//Enduro/X connection
type ExCon struct {
	con net.Conn

	reader *bufio.Reader
	writer *bufio.Writer

	ctx      *atmi.ATMICtx //ATMI Context
	id       int64         //Connection ID (clear), index by this
	id_comp  int64         //Compiled id
	id_stamp int64         //Part of timestamp (first 32 bits of id)
	contype  int           //Connection type

	outgoing chan DataBlock //This is for outgoing
	shutdown chan bool      //This is if we get shutdown messages
}

//We need a hash list of open connection (no matter incoming our outgoing...)
var MConnections map[int64]*ExCon
var MConnMutex = &sync.Mutex{}

//List of reply waiters on particular (compiled id)
var MConWaiter map[int64]*DataBlock
var MConWaiterMutex = &sync.Mutex{}

//List of reply waiters on given correlation id
var MCorrWaiter map[string]*DataBlock
var MCorrWaiterMutex = &sync.Mutex{}

//Have a channel of connections that are ready to accept the message
//(outgoing)

var Mfreeconns chan *ExCon
var MfreeconsLock sync.Mutex
var MPassiveLisener *net.IPConn

//Get open connection
//@param ac	ATMI Context
//@return Connection object acquired or nil (if no connection found)
func GetOpenConnection(ac *atmi.ATMICtx) *ExCon {
	ok := false
	MfreeconsLock.Lock()

	for !ok {
		con := <-Mfreeconns

		//Check that it is in connection hash list of opens
		//Maybe some old stuff in channel left
		MConnMutex.Lock()
		if nil != MConnections[con.id_comp] {
			ok = true
		} else {
			ac.TpLogInfo("Got expired connection: %d, try next",
				con.id_comp)
		}

	}

	MfreeconsLock.Unlock()

	return con
}

//Search for connection object by connection id
//@param connid	Compiled or simple connection id
func GetConnectionByID(ac *atmi.ATMICtx, connid int64) *ExCon {

	//If it is compiled we will lookup by hash.

	tstamp := connid >> 24
	id := connid & 0xffffff

	ac.TpLogInfo("Compiled id: %d, tstamp: %d, simple id: %d",
		connid, tstamp, id)

	if tstamp > 0 {
		ac.TpLogInfo("Looks like compiled connection id - lookup by hash")

		MConnMutex.Lock()
		ret := MConnections[connid]
		MConnMutex.Unlock()

		if ret == nil {
			ac.TpLogError("Connection by id %d not found", connid)
		}

		return ret
	} else {
		ac.TpLogInfo("Search by simple connection id")

		var ret *ExCon
		//TODO: Might want another index by simple id
		MConnMutex.Lock()
		for k, v := range MConnections {
			ac.TpLogDebug("got %d vs needle %d", v.id, connid)

			if v.id == connid {
				ret = v
				break
			}
		}

		MConnMutex.Unlock()

		if ret == nil {
			ac.TpLogError("Connection by id %d not found", connid)
		}

		return ret
	}

	//If it is simple, then we will iterate over the connections
	//Should never happen
	return nil

}

//Close all connections that are currently open
func CloseAllConnections(ac *atmi.ATMICtx) {
	ac.TpLogInfo("Closing all open connections...")

	MConnMutex.Lock()
	for k, v := range MConnections {

		ac.TpLogInfo("Closing %d", k)

		//Send infos that connection is closed.
		NotifyStatus(ac, id, FLAG_CON_DISCON)

		if err := v.con.Close(); err != nil {
			ac.TpLogError("Failed to close connection id %d: %s",
				k, err.Error())
		} else {
			ac.TpLogInfo("Connection closed ok")
		}

	}
	MConnMutex.Unlock()
}

//TODO: Remove from both lists
func RemoveFromCallLists(call *DataBlock) {

}

//This assumes that MConnections is locked
//@return <id> <tstamp> <compiled id> new connection id >0 or FAIL (-1)
func GetNewConnectionId() (int64, int64, int64) {

	var i int64

	for i = 1; i < MMaxConnections; i++ {
		if nil == MConnections[i] {
			/* return time.Uni */
			var t time.Time
			tstamp := t.Unix()
			//We have oldest 40 bit timestamp, youngest 24 bit - id
			var compiled_id = tstamp<<24 | (i & 0xffffff)

			return i, tstamp, compiled_id

		}
	}

	return FAIL, FAIL, FAIL
}

// Start a goroutine to read from our net connection
func ReadConData(con *ExCon, ch chan []byte, eCh chan error) {
	for {
		// try to read the data
		data, err := GetMessage(con)
		if err != nil {
			// send an error if it's encountered
			eCh <- err
			return
		}
		// send data if we read some.
		ch <- data
	}
}

//Operate with open connection
func HandleConnection(con *ExCon) {

	var dataIn chan []byte
	var dataInErr chan error
	ok := true
	ac := con.ctx
	last_out := true
	/* Need a:
	 * - byte array channel
	 * - error channel for socket
	 */

	go ReadConData(con, dataIn, dataInErr)

	for ok {

		//Connection ready, submit to list of available conns
		//(if last msg wast not) ougoing

		if last_out {
			Mfreeconns <- con
			last_out = false
		}

		//Add the connection to
		select {
		case dataIncoming := <-dataIn:

			inCorr := "" //Use for sending to incoming service (if not found in tables)
			//We should call the server or check that reply is needed
			//for some call in progress.
			//If this is connect per call, then we should keep the track
			//of the calls that wait for specific connetions to be replied

			//1. Check that we do have some reply waiters on connection
			MConWaiterMutex.Lock()

			call := MConWaiter[con.id_comp]
			if nil != call {
				//Send to connection
				MConWaiterMutex.Unlock()
				//This will tell should we terminate or not...
				NetDispatchConAnswer(call, dataIncoming, &ok)

				continue //<<< Continue!
			} else {
				MConWaiterMutex.Unlock()
			}

			if MCorrSvc != "" {
				var err error
				inCorr, err = NetGetCorID(call, dataIncoming)

				if nil == err {
					ac.TpLogWarn("Error calling correlator service: %s", err)
				} else if corr != "" {
					ac.TpLogWarn("Got correlator for incoming "+
						"message: [%s] - looking up for reply waiter", err)

					MCorrWaiterMutex.Lock()
					corwait := MCorrWaiter[inCorr]

					if nil != corwait {
						MConWaiterMutex.Unlock()
						NetDispatchCorAnswer(corwait)
						continue //<<< Continue!
					} else {
						MConWaiterMutex.Unlock()
					}
				}
			}

			//OK we have not found any corelation or this is incoming
			//Message, so submit to ATMI
			ac.TpLogInfo("Incoming mesage: corr: [%s]", inCorr)

			//If we work in sync mode, we shall wait for reply or
			//timeout...
			//Send the channel of reply data
			//In select on timeout channel
			//Do the action which comes first...
			//Or thread will wait until TPCALL terminates, and then do
			//reply if socket will be still open...
			go NetDispatchCall(con, data, inCorr)

			break
		case err := <-dataInErr:
			ac.TpLogError("Connection failed: %s - terminating", err)
			ok = false
			break
		case shutdown := <-con.shutdown:
			if shutdown {
				ac.TpLogWarn("Shutdown notification received - terminating")
				ok = false
			}
			break
		case dataOutgoing := <-con.outgoing:

			//Put the conneciton back to RR channel
			last_out = true

			//Send data away
			if err := PutMessage(con, dataOutgoing.data); nil != err {
				ac.TpLogError("Failed to send message to network"+
					": %s - terminating", err)
				ok = false
			}

			if dataOutgoing.send_and_shut {
				ac.TpLogInfo("CONN: %d - send_and_shut recieved - terminating",
					con.id_comp)
				ok = false
			}

			//TODO: If we expect to get reply back, and reply to caller
			//then we shall register the call in some list
			if MReqReply {
				ac.TpLogInfo("CONN: %d - Adding request to outgoing lists",
					con.id_comp)
				MConWaiter[con.id_comp] = dataOutgoing
			}

			break
		}
	}

}

//Handle the connection - connect to server
//Once finished, we should remove our selves from hash list
func GoDial(con *ExCon) {
	var err error
	var errA atmi.ATMIError
	con.ctx, errA = atmi.NewATMICtx()

	ac := con.ctx

	//Free up the slot once we are done
	defer func() {
		MConnMutex.Lock()

		if nil != con.ctx {
			con.ctx.TpLogWarn("Terminating connection object: id=%d, "+
				"tstamp=%d, id_comp=%d", con.id, con.id_stamp, con.id_comp)
		}
		MConnections[con.id] = nil

		MConnMutex.Unlock()

	}()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate ATMI Context: %d:%s\n",
			errA.Code(), errA.Message())
		return
	}

	con.ctx.TpLogWarn("Connection id=%d, "+
		"tstamp=%d, id_comp=%d doing connect to: %s", con.id, con.id_stamp, con.id_comp, MAddr)

	//Get the ATMI Context
	con.con, err = net.Dial("tcp", MAddr)

	if err != nil {
		// handle error
		con.ctx.TpLogError("Failed to connect to [%s]:%s", MAddr, err)
		return
	}

	//Have buffered read/write API to socket
	con.writer = bufio.NewWriter(con.con)
	con.reader = bufio.NewReader(con.con)

	//Connection open...
	NotifyStatus(con.ctx, con.id, FLAG_CON_ESTABLISHED)

	HandleConnection(con)

	//Connection closed...
	NotifyStatus(con.ctx, con.id, FLAG_CON_DISCON)

	//Close connection
	ac.TpLogWarn("Connection id=%d, "+
		"tstamp=%d, id_comp=%d closing...",
		con.id, con.id_stamp, con.id_comp)

	err = con.con.Close()

	if nil != err {
		ac.TpLogError("Failed to close connection: %s", err)
	}
}

//Call the status service if defined
func NotifyStatus(ac *atmi.ATMICtx, id int, flags string) {

	if MStatussvc == "" {
		return
	}

	buf, err := ac.NewUBF(1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s] - dropping incoming message",
			err.Error())
		return
	}

	if err = buf.BChg(u.EX_NETCONNID, 0, id); err != nil {
		ac.TpLogError("Failed to set EX_NETCONNID %d: %s", err.Code(), err.Message())
		return
	}

	if err = buf.Bchg(u.EX_NETFLAGS, 0, flags); err != nil {
		ac.TpLogError("Failed to set EX_NETFLAGS %d: %s", err.Code(), err.Message())
		return
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Sending notification")

	//Call the service for status notification
	if _, err = ac.TpCall(MStatussvc, buf, atmi.TPNOREPLY); nil != err {
		ac.TpLogError("Failed to call [%s]: %s", MStatussvc, err.Error())
		return
	}

}

//Return number of open connections
func GetOpenConnectionCount() int {

	MConnMutex.Lock()

	ret := len(MConnections)

	MConnMutex.Unlock()

	return ret
}

//Open the socket and wait for incoming connections
//On every new connection check the limits of total number
//of active conns.
func PassiveConnectionListener() {

	/* We need a atmi context here */
	ac, errA := atmi.NewATMICtx()
	var err error

	if nil != err {
		ac.TpLogError("Failed to create ATMI Context: %d:%s",
			errA.Code(), errA.Message())
		MShutdown = RUN_SHUTDOWN_FAIL
		return
	}
	ac.TpLogInfo("About to listen on: %s", MAddr)
	MPassiveLisener, err = net.Listen("tcp", MAddr)

	if err != nil {
		ac.TpLogError("Failed to listen on [%s]:%s", MAddr, err.Error())
		MShutdown = RUN_SHUTDOWN_FAIL
		return
	}

	for MShutdown == RUN_CONTINUE {

		for {
			var con ExCon
			con.con, err = MPassiveLisener.Accept()
			if err != nil {
				ac.TpLogError("Failed to accept connection: %s",
					err.Error())
				ln.Close()
				MShutdown = RUN_SHUTDOWN_FAIL
				return
			}

			//Add get connection number & add to hashes.

			//1. Prepare connection block
			con.id, con.id_stamp, con.id_comp = GetNewConnectionId()

			if con.id == FAIL {
				ac.TpLogError("Failed to get connection id - max reached?")
				MShutdown = true
				break
			}

			//2. Add to hash
			MConnections[con.id] = &con

			go handleConnection(conn)
		}
	}

	ac.TpLogWarn("Terminating listener thread...")

	//Termiante connection if shutdown requested
	MPassiveLisener.Close()
}
