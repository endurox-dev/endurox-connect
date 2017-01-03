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
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	atmi "github.com/endurox-dev/endurox-go"
)

//This is data block for sending messages int/out
type DataBlock struct {
	data []byte
	//sender_chan //optional if we want recieve reply back
	sender_chan chan DataBlock
	conn_id     int //Connection id if specified (0) - then random.
}

//Enduro/X connection
type ExCon struct {
	con      net.Conn
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

//Get the outgoing channel
func SendOut(data *DataBlock) {

}

// Start a goroutine to read from our net connection
func ReadConData(con *ExCon, ch chan []byte, eCh chan error) {
	for {
		// try to read the data
		data := make([]byte, 512)
		_, err := con.con.Read(data)
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
	/* Need a:
	 * - byte array channel
	 * - error channel for socket
	 */

	go ReadConData(con, dataIn, dataInErr)

	for {
		select {
		case dataIncoming := <-dataIn:
			break
		case err := <-dataInErr:
			break
		case shutdown := <-con.shutdown:
			break
		case dataOutgoing := <-con.outgoing:
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

}
