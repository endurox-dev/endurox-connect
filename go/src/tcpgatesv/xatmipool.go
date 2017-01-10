/*
** Pool of XATMI sessions
**
** @file xatmipool.go
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
	"sync"

	atmi "github.com/endurox-dev/endurox-go"
)

type XATMIPool struct {
	freechansync *sync.Mutex     //We need to lock the freechan
	freechan     chan int        //List of free channels submitted by wokers
	ctxs         []*atmi.ATMICtx //List of contexts
	nrWorkers    int             //Number of contexts

}

var MinXPool XATMIPool  //In XATMI pool
var MoutXPool XATMIPool //Out XATMI pool

var MXDispatcher = &sync.Mutex{}

//Initialize out pool
//@param ac 	ATMI contexts
//@param pool	XATMI pool
//@return error in case of error or nil if ok
func initPool(ac *atmi.ATMICtx, pool *XATMIPool) error {

	pool.freechan = make(chan int, pool.nrWorkers)

	pool.freechansync = &sync.Mutex{}

	for i := 0; i < pool.nrWorkers; i++ {

		ctx, err := atmi.NewATMICtx()

		if err != nil {
			ac.TpLogError("Failed to create context: %s", err.Message())
			return err
		}

		pool.ctxs = append(pool.ctxs, ctx)

		//Submit the free ATMI context
		pool.freechan <- i
	}
	return nil
}

//Close the open xatmi contexts
//@param ac	XATMI contexts
//@param pool	XATMI pool
func deInitPoll(ac *atmi.ATMICtx, pool *XATMIPool) {

	for i := 0; i < pool.nrWorkers; i++ {
		nr := <-pool.freechan

		ac.TpLogWarn("Terminating %d context", nr)
		pool.ctxs[nr].TpTerm()
		pool.ctxs[nr].FreeATMICtx()
	}
}

//Return the free X context
func getFreeXChan(ac *atmi.ATMICtx, pool *XATMIPool) int {
	//Should we use locking here?

	pool.freechansync.Lock()

	nr := <-pool.freechan

	pool.freechansync.Unlock()

	return nr
}
