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
		go GoDial(&con)
	}

	MConnMutex.Unlock()
}

//Periodic callback function
//Hmm do we have some context here?
//We will spawn connections here..
func Periodic(ac *atmi.ATMICtx) int {

	//if we are active, check that we have enought connections
	if MType == CON_TYPE_ACTIVE {
		CheckDial(ac)
	}

	return SUCCEED
}
