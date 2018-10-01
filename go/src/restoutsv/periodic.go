/**
 * @brief Enduro/X periodic service polling callback (using for echo advertise)
 *
 * @file periodic.go
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
/* vim: set ts=4 sw=4 et smartindent: */
