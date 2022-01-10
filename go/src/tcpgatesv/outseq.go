/**
 * @brief Outgiong (Ex->Net) message sequencing
 *
 * @file outseq.go
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
	"sync"

	atmi "github.com/endurox-dev/endurox-go"
)

//ATMI Out message dispatching block
//note that if this is second message in structure then "nr" is invalid
type ATMIOutBlock struct {
	id      int64
	nr      int
	pool    *XATMIPool
	ctxData *atmi.TPSRVCTXDATA
	buf     *atmi.TypedUBF
	cd      int
}

var MSeqOutMutex = &sync.Mutex{} //For out message sequencing
var MNrMessages = 0              //Number of messages enqueued
var MSeqOutMsgs map[int64][]*ATMIOutBlock

/*
//Get next work load block
//@param id connect id
//@return Message out or nil (EOF)
func XATMIDispatchCallNext(id int64) *ATMIOutBlock {
	//Assume current is done, extract next if have so...
	var ret *ATMIOutBlock = nil
	//Lock the queues
	MSeqOutMutex.Lock()

	MSeqOutMsgs[id] = append(MSeqOutMsgs[id][:0], MSeqOutMsgs[id][1:]...)

	if len(MSeqOutMsgs[id]) == 0 {
		MSeqOutMsgs[id] = nil
	}

	if nil != MSeqOutMsgs[id] {
		ret = MSeqOutMsgs[id][0]
	}

	MSeqOutMutex.Unlock()

	return ret
}
*/

//Process messages in loop for given connection id
//@param[in] id  connection id
func XATMIDispatchCallRunner(id int64, block *ATMIOutBlock) {

	//var nextBlock *ATMIOutBlock

	nrOurs := block.nr
	pool := block.pool //pool shall no change here amog the enqueued objects

	/*
		for nextBlock = block; nil != nextBlock; nextBlock = XATMIDispatchCallNext(id) {
			XATMIDispatchCall(nextBlock.pool, nrOurs,
				nextBlock.ctxData, nextBlock.buf, nextBlock.cd, false)
		}
	*/

	for {
		/* read block */
		MSeqOutMutex.Lock()

		if len(MSeqOutMsgs[id]) == 0 {
			MSeqOutMsgs[id] = nil
			MSeqOutMutex.Unlock()
			break
		}
		block = MSeqOutMsgs[id][0]
		MSeqOutMutex.Unlock()

		XATMIDispatchCall(block.pool, nrOurs,
			block.ctxData, block.buf, block.cd, false)

		/* delete block */
		MSeqOutMutex.Lock()
		MSeqOutMsgs[id] = MSeqOutMsgs[id][1:]
		MNrMessages--
		MSeqOutMutex.Unlock()
		MSeqNotif <- true

	}

	//Free up the chan
	pool.freechan <- nrOurs
}

//Sequenced message dispatching
//@param id connection id, either compiled or simple, up to user
//@param pool XATMI pool
//@param nr thread number in pool
//@param ctxData context data
//@param buf call buffer
//@param cd call descriptor (XATMI)
func XATMIDispatchCallSeq(id int64, pool *XATMIPool, nr int, ctxData *atmi.TPSRVCTXDATA,
	buf *atmi.TypedUBF, cd int) {

	//Clear the input channel..
	for len(MSeqNotif) > 0 {
		<-MSeqNotif
	}

	//Lock the queues
	MSeqOutMutex.Lock()

	startNew := false

	block := ATMIOutBlock{id: id, pool: pool, nr: nr, ctxData: ctxData, buf: buf, cd: cd}

	if nil == MSeqOutMsgs[id] {
		startNew = true
	}
	MSeqOutMsgs[id] = append(MSeqOutMsgs[id], &block)
	MNrMessages++
	if startNew {
		go XATMIDispatchCallRunner(id, &block)
	} else {
		//We shall release the channel as runner will work with his chan
		pool.freechan <- nr
	}

	MSeqOutMutex.Unlock()

	if MNrMessages >= MWorkersOut {
		//Wait on channel...
		//Stop the main thread to avoid consuming all incoming messages
		//If flow is one direction only (i.e. full async transfer)
		<-MSeqNotif
	}

}

/* vim: set ts=4 sw=4 et smartindent: */
