/*
** Service "object" routines
**
** @file service.go
** -----------------------------------------------------------------------------
** Enduro/X Middleware Platform for Distributed Transaction Processing
** Copyright (C) 2015, ATR Baltic, Ltd. All Rights Reserved.
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
	"encoding/base64"
	"fmt"
	"time"

	atmi "github.com/endurox-dev/endurox-go"
)

/*
#include <string.h>
*/
import "C"

//Advertise service for given service definition
//@param ac 	Context into which run the operation
//@return	ATMI Error
func (s *ServiceMap) Advertise(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("About to advertise: %s", s.Svc)

	errA := ac.TpAdvertise(s.Svc, "RESTOUT", RESTOUT)

	if nil == errA {
		s.echoIsAdvertised = true
		s.echoSchedAdv = false
	}

	return errA
}

//Remove service from shared memory
//@param ac	ATMI Context
//@return ATMI error
func (s *ServiceMap) Unadvertise(ac *atmi.ATMICtx) atmi.ATMIError {
	ac.TpLogInfo("About to unadvertise: %s", s.Svc)
	errA := ac.TpUnadvertise(s.Svc)

	if nil == errA {
		s.echoIsAdvertised = false
		s.echoSchedUnAdv = false
	}

	return errA
}

//Preparese echo buffers
//@param ac	ATMI Context
//@return ATMI error
func (s *ServiceMap) PreparseEchoBuffers(ac *atmi.ATMICtx) atmi.ATMIError {

	var errA atmi.ATMIError = nil
	if s.Echo {
		switch s.echoConvInt {

		case CONV_JSON2UBF:
			//Allocate the buffer
			s.echoUBF, errA = ac.NewUBF(atmi.ATMI_MSG_MAX_SIZE)

			if nil != errA {
				ac.TpLogError("failed to alloca ubf buffer %d:[%s]",
					errA.Code(), errA.Message())

				return errA
			}

			//Restore the data from JSON config...
			if errU := s.echoUBF.TpJSONToUBF(s.EchoData); nil != errU {
				ac.TpLogError("Failed to build UBF from JSON [%s] %d:[%s]",
					s.EchoData, errU.Code(), errU.Message())

				return atmi.NewCustomATMIError(atmi.TPEINVAL, "Failed to create "+
					"UBF buffer from JSON!")
			}
			break
		case CONV_JSON2VIEW:

			//Restore the data from JSON config...
			var errA atmi.ATMIError

			if s.echoVIEW, errA = ac.TpJSONToVIEW(s.EchoData); nil != errA {
				ac.TpLogError("Failed to build VIEW from JSON [%s] %d:[%s]",
					s.EchoData, errA.Code(), errA.Message())

				return errA
			}

			if errA := VIEWResetEchoError(ac, s, s.echoVIEW); nil != errA {
				ac.TpLogError("Failed to reset echo view...")
				return errA
			}

			break
		case CONV_RAW:
			data, err := base64.StdEncoding.DecodeString(s.EchoData)
			if err != nil {
				ac.TpLogError("Failed to decode json data [%s]: %s",
					s.EchoData, err.Error())
				return atmi.NewCustomATMIError(atmi.TPEINVAL,
					"Invalid echo_data for ["+s.Svc+"")
			}

			s.echoCARRAY, errA = ac.NewCarray(data)

			if nil != errA {
				ac.TpLogError("failed to alloca ubf buffer: %s",
					errA.Error())

				return errA
			}
			break

		}
	}

	return nil
}

//Call the echo service with JSON2UBF format data
//@param ac	ATMI Context
//@return nil (all ok) or ATMI error
func (s *ServiceMap) EchoJSON2UBF(ac *atmi.ATMICtx) atmi.ATMIError {

	//Allocate the buffer
	buf, errA := ac.NewUBF(atmi.ATMI_MSG_MAX_SIZE)

	if nil != errA {
		ac.TpLogError("failed to alloca ubf buffer %d:[%s]",
			errA.Code(), errA.Message())

		return errA
	}

	if errB := ac.BCpy(buf, s.echoUBF); nil != errB {
		ac.TpLogError("Failed to copy echo buffer:%s ",
			errB.Error())
		return atmi.NewCustomATMIError(atmi.TPESYSTEM, errB.Error())
	}

	ac.TpLogDebug("About to call echo service: [%s]", s.Svc)
	if _, errA = ac.TpCall(s.Svc, buf.GetBuf(), 0); nil != errA {
		ac.TpLogError("Failed to call echo service [%s]",
			errA.Error())
		return errA
	}

	ac.TpLogDebug("JSON2UBF: Echo Test to service [%s] OK", s.Svc)

	return nil
}

//Call the echo service with JSON2VIEW format data
//@param ac	ATMI Context
//@return nil (all ok) or ATMI error
func (s *ServiceMap) EchoJSON2VIEW(ac *atmi.ATMICtx) atmi.ATMIError {

	//Allocate the buffer
	buf, errA := ac.NewVIEW(s.echoVIEW.BVName(), 0)

	if nil != errA {
		ac.TpLogError("failed to alloca ubf buffer %d:[%s]",
			errA.Code(), errA.Message())
		return errA
	}

	_, errU := s.echoVIEW.BVCpy(buf)
	if nil != errU {
		ac.TpLogError("Failed to copy echo view: %s", errU.Error())
		return atmi.NewCustomATMIError(atmi.TPESYSTEM,
			fmt.Sprintf("Failed to copy echo view: %s!",
				errU.Error()))
	}

	ac.TpLogDebug("About to call echo service: [%s]", s.Svc)
	if _, errA = ac.TpCall(s.Svc, buf.GetBuf(), 0); nil != errA {
		ac.TpLogError("Failed to call echo service [%s]",
			errA.Error())
		return errA
	}

	ac.TpLogDebug("JSON2VIEW: Echo Test to service [%s] OK", s.Svc)

	return nil
}

//Call service with JSON buffer (directly loaded from config string)
//@param ac	ATMI Context
//@return nil (all ok) or ATMI error
func (s *ServiceMap) EchoJSON(ac *atmi.ATMICtx) atmi.ATMIError {
	//Allocate the buffer
	buf, errA := ac.NewJSON([]byte(s.EchoData))

	if nil != errA {
		ac.TpLogError("failed to set/alloc buffer: %s",
			errA.Error())

		return errA
	}

	ac.TpLogDebug("About to call echo service: [%s]", s.Svc)
	if _, errA = ac.TpCall(s.Svc, buf.GetBuf(), 0); nil != errA {
		ac.TpLogError("Failed to call echo service [%s]",
			errA.Error())
		return errA
	}

	ac.TpLogDebug("JSON: Echo Test to service [%s] OK", s.Svc)

	return nil

}

//Call service with TEXT/STRING buffer (directly loaded from config string)
//@param ac	ATMI Context
//@return nil (all ok) or ATMI error
func (s *ServiceMap) EchoText(ac *atmi.ATMICtx) atmi.ATMIError {
	//Allocate the buffer
	buf, errA := ac.NewString(s.EchoData)

	if nil != errA {
		ac.TpLogError("failed to alloca ubf buffer: %s",
			errA.Error())

		return errA
	}

	ac.TpLogDebug("About to call echo service: [%s]", s.Svc)
	if _, errA = ac.TpCall(s.Svc, buf.GetBuf(), 0); nil != errA {
		ac.TpLogError("Failed to call echo service [%s]",
			errA.Error())
		return errA
	}

	ac.TpLogDebug("STRING: Echo Test to service [%s] OK", s.Svc)

	return nil
}

//Call service with RAW/CARRAY buffer (directly loaded from config string)
//@param ac	ATMI Context
//@return nil (all ok) or ATMI error
func (s *ServiceMap) EchoRaw(ac *atmi.ATMICtx) atmi.ATMIError {

	//Allocate the buffer
	buf, errA := ac.NewCarray(s.echoCARRAY.GetBytes())

	if nil != errA {
		ac.TpLogError("failed to allocate carray buffer: %s",
			errA.Error())

		return errA
	}

	ac.TpLogDebug("About to call echo service: [%s]", s.Svc)
	if _, errA = ac.TpCall(s.Svc, buf.GetBuf(), 0); nil != errA {
		ac.TpLogError("Failed to call echo service [%s]",
			errA.Error())
		return errA
	}

	ac.TpLogDebug("CARRAY/RAW: Echo Test to service [%s] OK", s.Svc)

	return nil

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
			var result atmi.ATMIError = nil

			switch s.echoConvInt {

			case CONV_JSON2UBF:
				result = s.EchoJSON2UBF(ac)
				break
			case CONV_JSON:
				result = s.EchoJSON(ac)
				break
			case CONV_TEXT:
				result = s.EchoText(ac)
				break
			case CONV_RAW:
				result = s.EchoRaw(ac)
				break
			case CONV_JSON2VIEW:
				result = s.EchoJSON2VIEW(ac)
				break
			}

			trendSucc := " "
			trendFail := " "
			if nil == result {
				// Echo is OK
				if s.echoSucceeds <= s.EchoMinOK {
					s.echoSucceeds++
				} else {
					trendSucc = "+ "
				}
				s.echoFails = 0
			} else {
				//Echo failed
				if s.echoFails <= s.EchoMaxFail {
					s.echoFails++
				} else {
					trendFail = "+ "
				}
				s.echoSucceeds = 0
			}

			ac.TpLogInfo("Consecutive stats: succeed: %d%s(min OK: %d), "+
				"fail: %d%s(max Fail: %d)",
				s.echoSucceeds, trendSucc, s.EchoMinOK,
				s.echoFails, trendFail, s.EchoMaxFail)

			MadvertiseLock.Lock()

			if s.echoSucceeds == s.EchoMinOK {
				ac.TpLogWarn("Scheduling services to advertise")

				for _, ds := range s.Dependies {
					if !ds.echoIsAdvertised && !ds.echoSchedAdv {
						ac.TpLogWarn("Scheduling [%s] by "+
							"[%s] to advertise!",
							ds.Svc, s.Svc)
						ds.echoSchedAdv = true
						ds.echoSchedUnAdv = false
					}
				}

			} else if s.echoFails == s.EchoMaxFail {
				ac.TpLogWarn("Scheduling services to unadvertise")

				for _, ds := range s.Dependies {
					if ds.echoIsAdvertised && !ds.echoSchedUnAdv {
						ac.TpLogWarn("Scheduling [%s] by "+
							"[%s] to unadvertise!",
							ds.Svc, s.Svc)
						ds.echoSchedAdv = false
						ds.echoSchedUnAdv = true
					}
				}
			}

			MadvertiseLock.Unlock()

		case <-s.shutdown:
			// the read from ch has timed out
			ac.TpLogWarn("%s - Monitor thread shutdown received...",
				s.Svc)
			do_run = false
			MmonitorsShut <- true
		}
	}

	ac.TpTerm()

	return
}
