/*
** Outgiong (Ex->Net) message sequencing
**
** @file outseq.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2018, ATR Baltic, Ltd. All Rights Reserved.
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

var MSeqOutMsgs map[int64][]*ATMIOutBlock

//Get next work load block
//@param id connect id
//@return Message out or nil (EOF)
func XATMIDispatchCallNext(id int64) *ATMIOutBlock {
	//Assume current is done, extract next if have so...
	var ret *ATMIOutBlock = nil
	//Lock the queues
	MSeqOutMutex.Lock()

	MSeqOutMsgs[id] = append(MSeqOutMsgs[id][:0], MSeqOutMsgs[id][1:]...)

	ret = MSeqOutMsgs[id][0]

	MSeqOutMutex.Unlock()

	return ret
}

//Process messages in loop for given connection id
//@param[in] id  connection id
//@param[in] block call blcok
func XATMIDispatchCallRunner(id int64, block *ATMIOutBlock) {

	var nextBlock *ATMIOutBlock

	for nextBlock = block; nil != nextBlock; nextBlock = XATMIDispatchCallNext(id) {
		XATMIDispatchCall(nextBlock.pool, nextBlock.nr, nextBlock.ctxData, nextBlock.buf, nextBlock.cd)
	}
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

	//Lock the queues
	MSeqOutMutex.Lock()

	startNew := false

	block := ATMIOutBlock{id: id, pool: pool, nr: nr, ctxData: ctxData, buf: buf, cd: cd}
	if nil == MSeqOutMsgs[id] {
		startNew = true
	}
	MSeqOutMsgs[id] = append(MSeqOutMsgs[id], &block)

	if startNew {
		go XATMIDispatchCallRunner(id, &block)
	}

	MSeqOutMutex.Unlock()

}