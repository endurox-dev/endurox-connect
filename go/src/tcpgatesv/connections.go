/**
 * @brief This module is responsible for connections handling
 *
 * @file connections.go
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
	"bufio"
	"crypto/tls"
	"errors"
	"exutil"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	u "ubftab"

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
	con              *ExCon //Req-reply connection (for ex2net)

	tstamp_sent   int64 //Timestamp messag sent, TODO: We need cleanup monitor...
	send_and_shut bool  //Send and shutdown
}

//Enduro/X connection
type ExCon struct {
	mu  sync.Mutex
	con net.Conn

	reader *bufio.Reader
	//writer *bufio.Writer

	ctx      *atmi.ATMICtx //ATMI Context
	id       int64         //Connection ID (clear), index by this
	id_comp  int64         //Compiled id
	id_stamp int64         //Part of timestamp (first 32 bits of id)
	contype  int           //Connection type

	outgoing chan *DataBlock //This is for outgoing
	shutdown chan bool       //This is if we get shutdown messages
	is_open  bool            //Is connection open?

	theirip   string //Remote IP address
	theirport int    //Remote Port

	ourip   string //Local IP address
	outport int    //Local Port

	conmode string //Connection mode, [A]ctive or [P]assive

	busy      bool             //Is connection busy?
	schedZero bool             //Periodic zero shall be sent
	inIdle    exutil.StopWatch //Max idle time
}

//We need a hash list of open connection (no matter incoming our outgoing...)
var MConnectionsComp map[int64]*ExCon
var MConnectionsSimple map[int64]*ExCon
var MConnMutex = &sync.Mutex{}

//List of reply waiters on particular (compiled id)
//It is up to callers to remove them selves from this list.
var MConWaiter map[int64]*DataBlock
var MConWaiterMutex = &sync.Mutex{}

//List of reply waiters on given correlation id
//It is up to callers to remove them selves from this list.
var MCorrWaiter map[string]*DataBlock
var MCorrWaiterMutex = &sync.Mutex{}

var MfreeconsLock sync.Mutex
var MPassiveLisener net.Listener

//Round robin connection selector
var Mrrcon int = 0

//Get open connection
//@param ac	ATMI Context
//@return Connection object acquired or nil (if no connection found)
func GetOpenConnection(ac *atmi.ATMICtx) *ExCon {

	var con *ExCon

	//Have a timout object
	ac.TpLogInfo("GetOpenConnection: Setting alarm clock to: %d", MConnWaitTime)
	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(time.Second * time.Duration(MConnWaitTime))
		timeout <- true
	}()

	//Just use round-robin lookup over the connections...

	MConnMutex.Lock()

	i := 0
	len_c := len(MConnectionsComp)
	con_ok := false

	//Reset if some cons are removed..
	if Mrrcon >= len_c {
		Mrrcon = 1
	} else {
		Mrrcon++
	}

	for _, con_v := range MConnectionsComp {
		//Get first acceptable
		i++
		if i >= Mrrcon && con_v.is_open {
			con = con_v
			con_ok = true
			Mrrcon = i
			break
		}
	}

	//Just take first free & reset RR to that position
	if !con_ok {
		i = 0
		for _, con_v := range MConnectionsComp {
			//Get first acceptable
			i++
			if con_v.is_open {
				con = con_v
				Mrrcon = i
				break
			}
		}
	}

	ac.TpLogInfo("Current RR: %d", Mrrcon)

	MConnMutex.Unlock()

	if nil == con {
		ac.TpLogError("No connection found")
	}
	return con
}

//Search for connection object by connection id
//@param connid	Compiled or simple connection id
func GetConnectionByID(ac *atmi.ATMICtx, connid int64) *ExCon {

	//If it is compiled we will lookup by hash.

	var tstamp, id int64

	if atmi.ExSizeOfLong() == 8 {
		tstamp = connid >> 24
		id = connid & 0xffffff
	} else {
		//16 bit tstamp, 15 bit conn id
		tstamp = connid >> 15
		id = connid & 0x7fff
	}

	ac.TpLogInfo("Compiled id: %d, tstamp: %d, simple id: %d",
		connid, tstamp, id)

	if tstamp > 0 {
		ac.TpLogInfo("Looks like compiled connection id - lookup by hash")

		MConnMutex.Lock()
		ret := MConnectionsComp[connid]
		MConnMutex.Unlock()

		if ret == nil {
			ac.TpLogError("Connection by id %d not found", connid)
			return nil
		}

		return ret
	} else {
		ac.TpLogInfo("Search by simple connection id")

		MConnMutex.Lock()
		ret := MConnectionsSimple[connid]
		MConnMutex.Unlock()

		if ret == nil {
			ac.TpLogError("Connection by id %d not found", connid)

		}

		return ret
	}

}

//Close all connections that are currently open
func CloseAllConnections(ac *atmi.ATMICtx) {
	ac.TpLogInfo("Closing all open connections...")

	MConnMutex.Lock()

	//Request shutdown for connects...
	for _, v := range MConnectionsSimple {
		v.shutdown <- true
	}

	MConnMutex.Unlock()

	MConWait.Wait()

	/*
		//Will run in non locked mode...
		for k, v := range ch {

			ac.TpLogInfo("Closing %d (%d)", k, v.id)

			//Send infos that connection is closed.
			//No need these will be closed when go threads exit...
			//NotifyStatus(ac, v.id, FLAG_CON_DISCON)

			v.mu.Lock()
			if nil != v.con {
				if err := v.con.Close(); err != nil {
					ac.TpLogError("Failed to close connection id %d: %s",
						k, err.Error())
				} else {
					ac.TpLogInfo("Connection closed ok")
				}
			}
			v.mu.Unlock()

		}
	*/
}

//This assumes that MConnections is locked
//@return <id> <tstamp> <compiled id> new connection id >0 or FAIL (-1)
func GetNewConnectionId(ac *atmi.ATMICtx) (int64, int64, int64) {

	var i int64

	ac.TpLogDebug("Generating new connectiond Id, MMaxConnections=%d", MMaxConnections)
	//Will enumerate connections from
	for i = 1; i < MMaxConnections+1; i++ {
		if nil == MConnectionsSimple[i] {
			/* return time.Uni */
			tstamp := exutil.GetEpochMillis()
			//We have oldest 40 bit timestamp, youngest 24 bit - id
			var compiled_id int64

			if atmi.ExSizeOfLong() == 8 {
				compiled_id = tstamp<<24 | (i & 0xffffff)
				compiled_id &= 0x7fffffffffffffff;
			} else {
				//Have 16 bit time stamp and 15 bit txn id
				compiled_id = tstamp<<15 | (i & 0x7fff)
				//Delete the sign
				compiled_id &= 0x7fffffff
			}

			ac.TpLogWarn("Generated connection id: %d/%d/%d",
				i, tstamp, compiled_id)

			return i, tstamp, compiled_id

		} else {
			ac.TpLogDebug("Having conn %d/%d, thus +1",
				MConnectionsSimple[i].id, MConnectionsSimple[i].id_comp)
		}
	}

	ac.TpLogWarn("Cannot get connection id")

	return FAIL, FAIL, FAIL
}

// Start a goroutine to read from our net connection
func ReadConData(con *ExCon, ch chan<- []byte, eCh chan<- error) {

	//This guy also needs it's own atmi context
	ac, err := atmi.NewATMICtx()

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to allocate new context: %s\n",
			err.Message())
		eCh <- errors.New(fmt.Sprintf("Failed to allocate new context: %s\n",
			err.Message()))
		return
	}

	for {
		// try to read the data
		data, err := GetMessage(ac, con)
		if err != nil {
			// send an error if it's encountered
			ac.TpLogInfo("conn %d got error: %s - sending to eCh",
				con.id_comp, err.Error())
			eCh <- err
			return
		}

		ac.TpLogInfo("conn %d got message len: %d",
			con.id_comp, len(data))

		con.inIdle.Reset() //Reset connection idle timer

		//Detect if it is zero len, then drop the header
		dlen := len(data)
		if dlen > 0 && (!MFramingKeepHdr || dlen > MFramingLen) {
			// send data if we read some.
			ch <- data
		} else {
			ac.TpLogInfo("conn %d zero length message - ignore",
				con.id_comp)
		}
	}
}

//Set IP Addreess
//@param address ip address is format ip:port
//@param ip (out) ip address - parsed
//@param port (out) port parsed
func SetIPPort(ac *atmi.ATMICtx, addr net.Addr, ip *string, port *int) {

	*port = addr.(*net.TCPAddr).Port
	*ip = addr.(*net.TCPAddr).IP.String()

	ac.TpLogDebug("Got %s:%d", *ip, *port)

}

//Operate with open connection
func HandleConnection(con *ExCon) {

	MConWait.Add(1)
	defer MConWait.Done()

	dataIn := make(chan []byte)
	dataInErr := make(chan error)

	periodic_time := MPerZero

	var w exutil.StopWatch

	ok := true
	ac := con.ctx

	w.Reset()

	if !MTls_enable {

		/* Need a:
		 * - byte array channel
		 * - error channel for socket
		 */
		tcpcon := con.con.(*net.TCPConn)

		//Set options, for normal conn
		tcpcon.SetNoDelay(true)

		if MLinger > -1 {
			tcpcon.SetLinger(MLinger)
		}
	}

	//Really no wakeup
	if 0 == periodic_time {
		periodic_time = 9999999
	}

	//Connection open...
	NotifyStatus(ac, con.id, con.id_comp, FLAG_CON_ESTABLISHED, con)

	go ReadConData(con, dataIn, dataInErr)

	for ok {

		var preAllocUBF *atmi.TypedUBF = nil

		cur_timeout := periodic_time - int(w.GetDetlaSec())

		//Send zero away if it is time..
		if cur_timeout <= 0 {
			w.Reset()
			cur_timeout = periodic_time
			//we might get full channel...
			if MPerZero > 0 {
				RunZero(ac, con)
			}
			//else do nothing...
		}

		//Add the connection to
		ac.TpLogInfo("Conn: %d polling...", con.id_comp)
		select {
		case dataIncoming := <-dataIn:

			ac.TpLogDebug("dataIn: conn %d/%d got something on channel",
				con.id, con.id_comp)

			ac.TpLogDump(atmi.LOG_DEBUG, "Got message prefix (before swapping)",
				dataIncoming, len(dataIncoming))

			inCorr := "" //Use for sending to incoming service (if not found in tables)
			//We should call the server or check that reply is needed
			//for some call in progress.
			//If this is connect per call, then we should keep the track
			//of the calls that wait for specific connetions to be replied

			//1. Check that we do have some reply waiters on connection
			//Reduce the lock range...
			MConWaiterMutex.Lock()

			block := MConWaiter[con.id_comp]
			if nil != block {
				ac.TpLogInfo("Wo get a waiter on this conn reply")
				//Send to connection
				MConWaiterMutex.Unlock()
				//This will tell should we terminate or not...
				NetDispatchConAnswer(ac, con, block, dataIncoming, &ok)

				continue //<<< Continue!
			} else {
				MConWaiterMutex.Unlock()
			}

			if MCorrSvc != "" {

				buf, errA := AllocReplyDataBuffer(ac, con, "", dataIncoming, false)
				if nil != errA {
					ac.TpLogError("Failed to allocate buffer %d: %s",
						errA.Code(), errA.Message())
					//will terminat connection
					ok = false
					break
				}

				preAllocUBF = buf
				inCorr, errA = NetGetCorID(ac, buf)

				if nil != errA {
					ac.TpLogWarn("Error calling correlator service: %s",
						errA.Message())
				} else if inCorr != "" {
					ac.TpLogWarn("Got correlator for incoming "+
						"message: [%s] - looking up for reply waiter", inCorr)

					MCorrWaiterMutex.Lock()
					block := MCorrWaiter[inCorr]

					if nil != block {
						MCorrWaiterMutex.Unlock()

						//So this is answer, add some answer fields
						buf.BChg(u.EX_NERROR_CODE, 0, 0)
						buf.BChg(u.EX_NERROR_MSG, 0, "SUCCEED")

						ac.TpLogInfo("Reply waiter found! "+
							"Waiting on corr [%s] got corr [%s]",
							block.corr, inCorr)
						NetDispatchCorAnswer(ac, con, block,
							buf, &ok)
						continue //<<< Continue!
					} else {
						ac.TpLogInfo("Got request with " +
							"correlator (or waiter " +
							"did time-out...)")
						MCorrWaiterMutex.Unlock()
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

			//Well this guy looks like needs a handler from IN pool...

			ac.TpLogInfo("Waiting for free XATMI-in object")
			nr := getFreeXChan(ac, &MinXPool)
			ac.TpLogInfo("Got XATMI in object")

			//We might want to sync incoming messages
			//Wait for dispatch to finish
			if MSeqIn {
				NetDispatchCall(&MinXPool, nr, con, preAllocUBF, inCorr, dataIncoming)
			} else {
				go NetDispatchCall(&MinXPool, nr, con, preAllocUBF, inCorr, dataIncoming)
			}

			break
		case err := <-dataInErr:
			ac.TpLogError("Connection failed: %s - terminating", err.Error())
			ok = false
			break
		case shutdown := <-con.shutdown:
			if shutdown {
				ac.TpLogWarn("Shutdown notification received - terminating")
				ok = false
			}
			break

		case <-time.After(time.Second * time.Duration(cur_timeout)):
			break
		case dataOutgoing := <-con.outgoing:

			//Do not unlock as message was not locked
			//nolock = dataOutgoing.nolock
			//The caller did remove our selves from connection list...
			//Thos conn is already locked to him.

			//Send data away
			if err := PutMessage(ac, con, dataOutgoing.data); nil != err {
				ac.TpLogError("Failed to send message to network"+
					": %s - terminating", err)
				ok = false
			}
			//If the is non-persistent Net->EX, then shutdow

			if MReqReply == RR_NONPERS_NET2EX {
				ac.TpLogInfo("CONN: %d - send_and_shut recieved - terminating",
					con.id_comp)
				ok = false
			}

			break
		}
	}

	//Remove our selves from connection list
	ac.TpLogInfo("Removing %d/%d from connection list", con.id_comp, con.id)
	MConnMutex.Lock()
	delete(MConnectionsSimple, con.id)
	delete(MConnectionsComp, con.id_comp)

	//Close connection
	if con.is_open {
		//Bug #464
		con.is_open = false
		con.con.Close()
	}
	//Connection closed...
	NotifyStatus(ac, con.id, con.id_comp, FLAG_CON_DISCON, con)

	MConnMutex.Unlock()

	//Support #828
	//However, consider TpTerm() to be used after each NotifyStatus() call
        //(to avoid extra IPC queues open for each connection) or use works for this
        //purpose.
	ac.TpTerm()
}

//This will setup connection
//@param con 	Newly created connection object
func SetupConnection(con *ExCon) {

	con.outgoing = make(chan *DataBlock, 10)
	con.shutdown = make(chan bool, 10)

	con.inIdle.Reset() //Reset idle timeout counter...
}

//Setup data block commons
//@param block	Data block to setup
func SetupDataBlock(block *DataBlock) {

	block.atmi_chan = make(chan *atmi.TypedUBF, 10)
}

//Handle the connection - connect to server
//Once finished, we should remove our selves from hash list
func GoDial(con *ExCon, block *DataBlock) {
	var err error
	var errA atmi.ATMIError
	con.ctx, errA = atmi.NewATMICtx()

	ac := con.ctx

	//Free up the slot once we are done
	defer func() {
		//		MConnMutex.Lock()

		if nil != con.ctx {
			ac.TpLogWarn("Terminating connection object: id=%d, "+
				"tstamp=%d, id_comp=%d", con.id, con.id_stamp, con.id_comp)
		}
		//		MConnMutex.Unlock()

	}()

	if nil != errA {
		fmt.Fprintf(os.Stderr, "Failed to allocate ATMI Context: %d:%s\n",
			errA.Code(), errA.Message())
		return
	}

	ac.TpLogWarn("Connection id=%d, "+
		"tstamp=%d, id_comp=%d doing connect to: %s", con.id, con.id_stamp, con.id_comp, MAddr)

	//Get the ATMI Context
	con.mu.Lock()

	if MTls_enable {

		ac.TpLogInfo("TLS Dial...")
		con.con, err = tls.Dial("tcp", MAddr, &MTls_config)

	} else {
		con.con, err = net.Dial("tcp", MAddr)
	}

	con.mu.Unlock()

	if err != nil {
		// handle error
		ac.TpLogError("Failed to connect to [%s]:%s", MAddr, err)

		//Remove connection from hashes
		/*
			Not in let yet - why not?
		*/
		MConnMutex.Lock()
		delete(MConnectionsSimple, con.id)
		delete(MConnectionsComp, con.id_comp)
		MConnMutex.Unlock()

		//Generate erro buffer
		if block != nil {
			if rply_buf, _ := GenErrorUBF(ac, 0, atmi.NENOCONN,
				fmt.Sprintf("Failed to connect to [%s]:%s", MAddr, err)); nil != rply_buf {
				block.atmi_chan <- rply_buf
			}
		}
		return
	}

	ac.TpLogInfo("Marking connection %d/%d as open", con.id, con.id_comp)
	con.inIdle.Reset();

	//Print peer cert...

	if MTls_enable {
		logTlsPeer(ac, con)
	}

	/*  Bug #225 - register connection already when doing to dia
	MConnMutex.Lock()
	MConnectionsSimple[con.id] = con
	MConnectionsComp[con.id_comp] = con
	MConnMutex.Unlock()
	*/

	SetIPPort(ac, con.con.LocalAddr(), &con.ourip, &con.outport)
	SetIPPort(ac, con.con.RemoteAddr(), &con.theirip, &con.theirport)

	//Bug #304
	//con.is_open = true

	//Have buffered read/write API to socket
	//con.writer = bufio.NewWriter(con.con)
	con.reader = bufio.NewReader(con.con)

	con.conmode = CON_TYPE_ACTIVE

	//Bug #304
	//The last thing we want is to mark connection open, otherwise periodic
	//peridic message sender might pick up not yet prepared object and send
	//invalid data to network (like connection mode - got empty string...!)
	//The true/false is atomic, should be ok with periodic..
	con.is_open = true

	HandleConnection(con)

	//Close connection
	ac.TpLogWarn("Connection id=%d, "+
		"tstamp=%d, id_comp=%d terminating...",
		con.id, con.id_stamp, con.id_comp)

	if nil != err {
		ac.TpLogError("Failed to close connection: %s", err)
	}
}

//Print the TLS pper infos
func logTlsPeer(ac *atmi.ATMICtx, con *ExCon) {

	ac.TpLogInfo("*** TLS PEER INFO START ***")

	tlscon, _ := con.con.(*tls.Conn)

	state := tlscon.ConnectionState()

	ac.TpLogInfo("HandshakeComplete: %v", state.HandshakeComplete)
	ac.TpLogInfo("ServerName: %v", state.HandshakeComplete)
	ac.TpLogInfo("Version: %v", state.Version)
	ac.TpLogInfo("NegotiatedProtocol: %v", state.NegotiatedProtocol)
	ac.TpLogInfo("DidResume: %v", state.NegotiatedProtocolIsMutual)
	ac.TpLogInfo("NegotiatedProtocolIsMutual: %v", state.DidResume)
	ac.TpLogInfo("CipherSuite: %v", state.CipherSuite)

	ac.TpLogInfo("Certificate chain:")
	for i, cert := range state.PeerCertificates {
		subject := cert.Subject
		issuer := cert.Issuer
		ac.TpLogInfo("no: %d subject: Country =%v Province=%v Locality=%v "+
			"Organization=%v OrganizationalUnit=%v CommonName=[%v] SerialNumber=[%v]", i,
			subject.Country, subject.Province, subject.Locality,
			subject.Organization, subject.OrganizationalUnit,
			subject.CommonName, subject.SerialNumber)

		ac.TpLogInfo("no: %d issuer: Country =%v Province=%v Locality=%v "+
			"Organization=%v OrganizationalUnit=%v CommonName=[%v] SerialNumber=[%v]", i,
			issuer.Country, issuer.Province, issuer.Locality,
			issuer.Organization, issuer.OrganizationalUnit,
			issuer.CommonName, issuer.SerialNumber)
	}

	ac.TpLogInfo("*** TLS PEER INFO END   ***")
}

//Call the status service if defined
func NotifyStatus(ac *atmi.ATMICtx, id int64, idcomp int64, flags string, con *ExCon) {

	if MStatussvc == "" {
		return
	}

	buf, err := ac.NewUBF(1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s] - dropping incoming message",
			err.Error())
		return
	}

	if err = buf.BChg(u.EX_NETGATEWAY, 0, MGateway); err != nil {
		ac.TpLogError("Failed to set EX_NETGATEWAY %d: %s", err.Code(), err.Message())
		return
	}

	if err = buf.BChg(u.EX_NETCONNID, 0, id); err != nil {
		ac.TpLogError("Failed to set EX_NETCONNID %d: %s", err.Code(), err.Message())
		return
	}

	if idcomp != atmi.FAIL {
		if err = buf.BChg(u.EX_NETCONNIDCOMP, 0, idcomp); err != nil {
			ac.TpLogError("Failed to set EX_NETCONNIDCOMP %d: %s", err.Code(), err.Message())
			return
		}
	}

	if err = buf.BChg(u.EX_NETFLAGS, 0, flags); err != nil {
		ac.TpLogError("Failed to set EX_NETFLAGS %d: %s", err.Code(), err.Message())
		return
	}

	if nil != con {

		//Setup IP/port our/their and role (optional):
		if err = buf.BChg(u.EX_NETOURIP, 0, con.ourip); err != nil {
			ac.TpLogError("Failed to set EX_NETOURIP %d: %s", err.Code(), err.Message())
		}

		if err = buf.BChg(u.EX_NETOURPORT, 0, con.outport); err != nil {
			ac.TpLogError("Failed to set EX_NETOURPORT %d: %s", err.Code(), err.Message())
		}

		//Setup IP/port our/their and role
		if err = buf.BChg(u.EX_NETTHEIRIP, 0, con.theirip); err != nil {
			ac.TpLogError("Failed to set EX_NETTHEIRIP %d: %s", err.Code(), err.Message())
		}

		if err = buf.BChg(u.EX_NETTHEIRPORT, 0, con.theirport); err != nil {
			ac.TpLogError("Failed to set EX_NETTHEIRPORT %d: %s", err.Code(), err.Message())
		}

		if err = buf.BChg(u.EX_NETCONMODE, 0, con.conmode); err != nil {
			ac.TpLogError("Failed to set EX_NETCONMODE %d: %s", err.Code(), err.Message())
		}
	}

	buf.TpLogPrintUBF(atmi.LOG_DEBUG, "Sending notification")

	//Call the service for status notification
	if _, err = ac.TpACall(MStatussvc, buf, atmi.TPNOREPLY|atmi.TPNOBLOCK); nil != err {
		ac.TpLogError("Failed to call [%s]: %s", MStatussvc, err.Error())
		return
	}

}

//Return number of open connections
func GetOpenConnectionCount() int64 {

	MConnMutex.Lock()

	ret := len(MConnectionsComp)

	MConnMutex.Unlock()

	return int64(ret)
}

//Open the socket and wait for incoming connections
//On every new connection check the limits of total number
//of active conns.
func PassiveConnectionListener() {

	/* We need a atmi context here */
	ac, errA := atmi.NewATMICtx()
	var err error

	if nil != err {
		fmt.Fprintf(os.Stderr, "Failed to create ATMI Context: %d:%s\n",
			errA.Code(), errA.Message())
		MShutdown = RUN_SHUTDOWN_FAIL
		return
	}
	ac.TpLogInfo("About to listen on: %s", MAddr)

	if MTls_enable {
		//TLS Mode...
		MPassiveLisener, err = tls.Listen("tcp", MAddr, &MTls_config)
		if err != nil {
			ac.TpLogError("Failed to listen on [%s]:%s", MAddr, err.Error())
			MShutdown = RUN_SHUTDOWN_FAIL
			return
		}
	} else {

		MPassiveLisener, err = net.Listen("tcp", MAddr)

		if err != nil {
			ac.TpLogError("Failed to listen on [%s]:%s", MAddr, err.Error())
			MShutdown = RUN_SHUTDOWN_FAIL
			return
		}
	}

	for MShutdown == RUN_CONTINUE {

		for {
			var con ExCon
			//Create ATMI context for connection
			con.ctx, errA = atmi.NewATMICtx()

			if nil != errA {
				ac.TpLogError("Failed to create ATMI "+
					"Context for connection: %d:%s",
					errA.Code(), errA.Message())
				MShutdown = RUN_SHUTDOWN_FAIL
				return
			}

			ac.TpLogInfo("Got connection atmi object: %p",
				con.ctx)

			SetupConnection(&con)

			con.con, err = MPassiveLisener.Accept()
			if err != nil {
				ac.TpLogError("Failed to accept connection: %s",
					err.Error())
				MPassiveLisener.Close()
				MShutdown = RUN_SHUTDOWN_FAIL
				return
			}

			//Print some debug infos about connection...
			if MTls_enable {
				tlscon := con.con.(*tls.Conn)

				if err := tlscon.Handshake(); nil != err {
					ac.TpLogError("Failed to handshake: %s", err)
					con.con.Close()
					//continue to wait for connections...
					continue
				}
				logTlsPeer(ac, &con)
			}

			//Have buffered read/write API to socket
			//con.writer = bufio.NewWriter(con.con)
			con.reader = bufio.NewReader(con.con)

			//Add get connection number & add to hashes.

			//1. Prepare connection block
			MConnMutex.Lock()
			con.id, con.id_stamp, con.id_comp = GetNewConnectionId(ac)

			//Fill conn details here!

			SetIPPort(ac, con.con.LocalAddr(), &con.ourip, &con.outport)
			SetIPPort(ac, con.con.RemoteAddr(), &con.theirip, &con.theirport)

			//Here it is open for 100%
			con.is_open = true

			ac.TpLogWarn("Got new connection id=%d tstamp=%d id_comp=%d",
				con.id, con.id_stamp, con.id_comp)

			if con.id == FAIL {
				ac.TpLogError("Failed to get connection id - max reached? " +
					"Will close connection...")
				con.con.Close()
				/* MShutdown = RUN_SHUTDOWN_FAIL */
				MConnMutex.Unlock()
				break
			}

			//2. Add to hash

			MConnectionsSimple[con.id] = &con
			MConnectionsComp[con.id_comp] = &con
			MConnMutex.Unlock()
			con.conmode = CON_TYPE_PASSIVE
			go HandleConnection(&con)
		}
	}

	ac.TpLogWarn("Terminating listener thread...")

	//Termiante connection if shutdown requested
	MPassiveLisener.Close()
}

/* vim: set ts=4 sw=4 et smartindent: */
