/*
** Service "object" routines
**
** @file service.go
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
	"time"

	atmi "github.com/endurox-dev/endurox-go"
)

//Advertise service for given service definition
//@param ac 	Context into which run the operation
//@return	ATMI Error
func (s *ServiceMap) Advertise(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("About to advertise: %s", s.Svc)
	return ac.TpAdvertise(s.Svc, "RESTOUT", RESTOUT)
}

//Remove service from shared memory
//@param ac	ATMI Context
//@return ATMI error
func (s *ServiceMap) Unadvertise(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("About to unadvertise: %s", s.Svc)
	return ac.TpUnadvertise(s.Svc)
}

//Do the monitoring of the target service
//We need to make possible to shutdown threads cleanly...
func (s *ServiceMap) Monitor() {

	ac, err := atmi.NewATMICtx()

	if err != nil {
		ac.TpLogError("Failed to create context: %s!!!!", err.Message())
		MmonitorsShut <- true
		return
	}

	do_run := true

	for do_run {

		//Have a timout object
		ac.TpLogInfo("Service %s echo tread in sleeping: %d",
			s.Svc, s.EchoTime)
		wakeUp := make(chan bool, 1)
		go func() {
			time.Sleep(time.Second * time.Duration(s.EchoTime))
			wakeUp <- true
		}()

		select {
		case <-wakeUp:
			//Send echo (we will do tpcall, right?)
			//We will support all types of the buffer formats!
			//To Echo services....

		case <-s.shutdown:
			// the read from ch has timed out
			ac.TpLogWarn("%s - Monitor thread shutdown received...", s.Svc)
			do_run = false
			MmonitorsShut <- true
		}
	}

	ac.TpTerm()

	return
}
