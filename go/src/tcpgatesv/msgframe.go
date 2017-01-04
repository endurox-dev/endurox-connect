/*
** Message framing, varous format support, including socket reading support
**
** @file msgframe.go
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
	"errors"

	atmi "github.com/endurox-dev/endurox-go"
)

/*
 * Mode constant table
 */
const (
	FRAME_LITTLE_ENDIAN      = 'l' //Little Endian, does not include len bytes it self
	FRAME_LITTLE_ENDIAN_ILEN = 'L' //Big Endian, include len bytes
	FRAME_BIG_ENDIAN         = 'b' //Big endian, does not includ bytes len it self
	FRAME_BIG_ENDIAN_ILEN    = 'B' //Big endian, include len it self
	FRAME_ASCII              = 'a' //Ascii, does not include len it self
	FRAME_ASCII_ILEN         = 'A' //Ascii, does not include len it self
	FRAME_NO                 = 'n' //No frame
	FRAME_DELIM_START        = 'd' //Delimiter, start
	FRAME_DELIM_STOP         = 'D' //Delimiter, stop
	FRAME_DELIM_BOTH         = 'E' //Delimiter, both
)

//This sets number of bytes to read from message, if not running in delimiter
//mode
//@param ac	ATMI Context into which we run
//@return Error or nil
func ConfigureNumberOfBytes(ac *atmi.ATMICtx) error {
	var c rune
	var n int
	first := false

	ac.TpLogInfo("Framing mode from config [%s]", MFraming)

	for _, r := range MFraming {
		if first {
			c = r
		} else if c != r {
			return errors.New("Different symbols in message framing: [" +
				string(c) + "] and [" + string(r) + "]")
		}
		n++
	}

	MFramingCode = c
	MFramingLen = n

	switch MFramingCode {
	case FRAME_LITTLE_ENDIAN:
		ac.TpLogInfo("Little endian mode, %d bytes, "+
			"does not include prefix len", MFramingLen)
		break
	case FRAME_LITTLE_ENDIAN_ILEN:
		ac.TpLogInfo("Little endian mode, %d bytes, "+
			"does include prefix len", MFramingLen)
		MFamingInclPfxLen = true
		break
	case FRAME_BIG_ENDIAN:
		ac.TpLogInfo("Big endian mode, %d bytes, "+
			"does not include prefix len", MFramingLen)
		break
	case FRAME_BIG_ENDIAN_ILEN:
		ac.TpLogInfo("Big endian mode, %d bytes, "+
			"does include prefix len", MFramingLen)
		MFamingInclPfxLen = true
		break
	case FRAME_ASCII:
		ac.TpLogInfo("Ascii len pfx mode, %d bytes, "+
			"does not include prefix len", MFramingLen)
		break
	case FRAME_ASCII_ILEN:
		ac.TpLogInfo("Ascii len pfx mode, %d bytes, "+
			"does include prefix len", MFramingLen)
		MFamingInclPfxLen = true
		break
	case FRAME_NO:
		MFramingLen = 0
		ac.TpLogInfo("No framing used")
		break
	case FRAME_DELIM_START:
		MFramingLen = 0
		ac.TpLogInfo("Starting delimiter: %x", MDelimStart)
		break
	case FRAME_DELIM_STOP:
		MFramingLen = 0
		ac.TpLogInfo("Stopping delimiter: %x", MDelimStop)
		break
	case FRAME_DELIM_BOTH:
		MFramingLen = 0
		ac.TpLogInfo("Start delimiter %x, Stop delimiter: %x",
			MDelimStart, MDelimStop)
		break
	}

	return nil
}

//Function - read the number of bytes...

//Function - read until the delimiter (stx only used for verification...)

//Write off the message with len

//Write off message with delimiter encapuslation
