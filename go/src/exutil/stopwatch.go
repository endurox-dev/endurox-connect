/*
** Simple stopwatch implementation
**
** @file stopwatch.go
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
package exutil

import (
	"time"
)

//Get UTC milliseconds since epoch
//@return epoch milliseconds
func GetEpochMillis() int64 {
	now := time.Now()
	nanos := now.UnixNano()
	millis := nanos / 1000000

	return millis
}

//About incoming & outgoing messages:
type StopWatch struct {
	start int64 //Timestamp messag sent
}

//Reset the stopwatch
func (s *StopWatch) Reset() {
	s.start = GetEpochMillis()
}

//Get delta milliseconds
//@return time spent in milliseconds
func (s *StopWatch) GetDeltaMillis() int64 {
	return GetEpochMillis() - s.start
}

//Get delta seconds of the stopwatch
//@return return seconds spent
func (s *StopWatch) GetDetlaSec() int64 {
	return (GetEpochMillis() - s.start) / 1000
}
