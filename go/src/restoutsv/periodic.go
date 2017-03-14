/*
** Enduro/X periodic service polling callback (using for echo advertise)
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

//Run scheduled tasks for advertise/unadvertise
//@param ac	ATMI Context (server main)
//@return SUCCEED 0, or -1 FAIL
func Periodic(ac *atmi.ATMICtx) int {

	ret := atmi.SUCCEED

	ac.TpLogDebug("Periodic()")

	MadvertiseLock.Lock()

	//Loop over the all services and check the required actions
	for _, v := range Mservices {

		if v.echoSchedAdv {

			ac.TpLogInfo("periodic: [%s] needs to be advertised",
				v.Svc)

			if errA := v.Advertise(ac); errA != nil {

				ac.TpLogError("Failed to advertise [%s]: %s",
					v.Svc, errA.Error())

				return FAIL
			}

		} else if v.echoSchedUnAdv {

			ac.TpLogInfo("periodic: [%s] needs to be unadvertised",
				v.Svc)
			if errA := v.Unadvertise(ac); errA != nil {

				ac.TpLogError("Failed to unadvertise [%s]: %s",
					v.Svc, errA.Error())

				return FAIL
			}
		}
	}

	MadvertiseLock.Unlock()

	return ret
}
