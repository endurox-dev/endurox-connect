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
	"fmt"
	"io"
	"strconv"

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
	FRAME_DELIM_STOP         = 'd' //Delimiter, stop
	FRAME_DELIM_BOTH         = 'D' //Delimiter, stop & start
)

//This sets number of bytes to read from message, if not running in delimiter
//mode
//@param ac	ATMI Context into which we run
//@return Error or nil
func ConfigureNumberOfBytes(ac *atmi.ATMICtx) error {
	var c rune
	var n int
	first := true

	ac.TpLogInfo("Framing mode from config [%s]", MFraming)

	for _, r := range MFraming {
		if first {
			c = r
		} else if c != r {
			ac.TpLogError("Different symbols in message framing: [" +
				string(c) + "] and [" + string(r) + "]")
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
	case FRAME_DELIM_STOP:
		MFramingLen = 0
		ac.TpLogInfo("Stopping delimiter: %x", MDelimStop)
		break
	case FRAME_DELIM_BOTH:
		MFramingLen = 0
		ac.TpLogInfo("Start delimiter %x, Stop delimiter: %x",
			MDelimStart, MDelimStop)
		break
	default:
		ac.TpLogError("Invalid framing...")
		return errors.New("Invalid message framing...")
		break
	}

	return nil
}

//Read the message from connection
//@param con 	Connection handler
//@return <Binary message read>, <Error or nil>
func GetMessage(con *ExCon) ([]byte, error) {
	ac := con.ctx
	if MFramingLen > 0 {

		header := make([]byte, MFramingLen)
		var mlen int64 = 0
		var mlenStr = ""

		ac.TpLogError("Reading %d number of bytes as header", MFramingLen)

		//Read number of bytes, or up till the symbol
		n, err := io.ReadFull(con.reader, header)

		if nil != err {
			ac.TpLogError("Failed to read header of %d bytes: %s",
				MFramingLen, err)
			return nil, err
		}

		if n != MFramingLen {

			emsg := fmt.Sprintf("Invlid header len read, expected: %d got %d",
				MFramingLen, n)
			ac.TpLogError("%s", emsg)
			return nil, errors.New(emsg)
		}

		ac.TpLogDump(atmi.LOG_DEBUG, "Got message prefix", header, len(header))

		//Decode the length now...
		if MFramingCode != FRAME_ASCII && MFramingCode != FRAME_ASCII_ILEN {

			for i := 0; i < MFramingLen; i++ {

				switch MFramingCode {
				case FRAME_BIG_ENDIAN:
				case FRAME_BIG_ENDIAN_ILEN:
					mlen <<= 8               //Move the current byte to front
					mlen |= int64(header[i]) //Add current byte
					break
				case FRAME_LITTLE_ENDIAN:
				case FRAME_LITTLE_ENDIAN_ILEN:
					mlen <<= 8                                  //Move the current byte to end
					mlen |= int64(header[int(MFramingLen-1)-i]) //Add current byte, but take from older
					break
				}
			}
		} else {
			mlenStr = string(header)
		}

		if MFramingCode == FRAME_ASCII || MFramingCode == FRAME_ASCII_ILEN {
			ac.TpLogInfo("Got string prefix len: [%s]", mlenStr)
			itmp, e1 := strconv.Atoi(mlenStr)

			if nil != e1 {
				ac.TpLogError("Invalid message length received: "+
					"[%s] - cannot parse as decimal: %s",
					mlenStr, e1)
				return nil, e1
			}

			mlen = int64(itmp)

		}

		if MFamingInclPfxLen {
			mlen -= int64(MFramingLen)
		}

		ac.TpLogInfo("Got header, indicating message len to read: %d", mlen)

		if mlen < 0 {
			ac.TpLogError("Reiceived invalid message len: %d", mlen)
			return nil, errors.New(fmt.Sprintf(
				"Reiceived invalid message len: %d", mlen))
		}

		if MFramingMaxMsgLen > 0 && mlen > int64(MFramingMaxMsgLen) {
			ac.TpLogError("Error ! Message len received: %d,"+
				" max message size configured: %d", mlen, MFramingMaxMsgLen)
			return nil, errors.New(fmt.Sprintf("Error ! Message len received: %d,"+
				" max message size configured: %d", mlen, MFramingMaxMsgLen))
		}

		//..And read the number of bytes...
		data := make([]byte, mlen)
		n, err = io.ReadFull(con.reader, data)

		if err != nil {
			ac.TpLogError("Failed to read %d bytes: %s", mlen, err)
			return nil, err
		}

		if int64(n) != mlen {
			emsg := fmt.Sprintf("Invalid bytes read, expected: %d got %d",
				mlen, n)

			ac.TpLogError("%s", emsg)
			return nil, errors.New(emsg)
		}

		ac.TpLogDump(atmi.LOG_DEBUG, "Message read", data, len(data))

		return data, nil
	} else {
		ac.TpLogInfo("About to read message until delimiter %x", MDelimStop)

		//If we use delimiter, then read pu till that
		data, err := con.reader.ReadBytes(MDelimStop)

		if err != nil {

			ac.TpLogError("Failed to read message with %x seperator: %s",
				MDelimStop, err)
			return nil, err
		}

		ac.TpLogDump(atmi.LOG_DEBUG, "Got the message with end seperator",
			data, len(data))

		if MFramingCode == FRAME_DELIM_BOTH {
			//Check the start of the message to match the delimiter
			if data[0] != MDelimStart {
				emsg := fmt.Sprintf("Expected message start byte %x but got %x",
					MDelimStart, data[0])
				ac.TpLogError("%s", emsg)
				return nil, errors.New(emsg)
			}

			//Strip off the first byte.
			data = data[1:]

			return data, nil
		}
	}

	//We should not get here anyway
	return nil, errors.New("Unexpeced EOF")
}

//Put message on socket
func PutMessage(con *ExCon, data []byte) error {

	ac := con.ctx

	ac.TpLogInfo("Building ougoing message: len %d", MFramingLen)

	ac.TpLogDump(atmi.LOG_DEBUG, "Preparing message for sending", data, len(data))

	if MFramingLen > 0 {
		var mlen int64 = int64(len(data))
		header := make([]byte, MFramingLen)

		if MFamingInclPfxLen {
			mlen += int64(MFramingLen)
		}

		//Generate the header
		if MFramingCode != FRAME_ASCII && MFramingCode != FRAME_ASCII_ILEN {

			for i := 0; i < MFramingLen; i++ {

				switch MFramingCode {
				case FRAME_BIG_ENDIAN:
				case FRAME_BIG_ENDIAN_ILEN:
					//So the least significant byte goes to end the array
					header[(MFramingLen-1)-i] = byte(mlen & 0xff)
					break
				case FRAME_LITTLE_ENDIAN:
				case FRAME_LITTLE_ENDIAN_ILEN:
					//So the least significant byte goes in front of the array
					header[i] = byte(mlen & 0xff)
					break
				}

				mlen >>= 8
			}

		} else {
			mlenStr := fmt.Sprintf("%0*d", MFramingLen, mlen)
			header = []byte(mlenStr)
		}

		// Print len
		ac.TpLogDump(atmi.LOG_INFO, "Built message header",
			header, len(header))

		//About to send message.
		dataToSend := append(header[:], data[:]...)

		ac.TpLogDump(atmi.LOG_DEBUG, "Sending message, w len pfx",
			dataToSend, len(dataToSend))

		nw, err := con.writer.Write(dataToSend)

		if nil != err {
			errMsg:=fmt.Sprintf("Failed to write data to socket: %s", err);
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		err = con.writer.Flush()

		if nil != err {
			errMsg:=fmt.Sprintf("Failed to flush socket: %s", err);
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		if nil != err {
			ac.TpLogError("Failed to write data to socket: %s", err)
		}

		ac.TpLogInfo("Written %d bytes to socket", nw)

	} else {

		var dataToSend []byte
		//Put (STX)ETX
		if MFramingCode == FRAME_DELIM_BOTH {
			dataToSend = append(([]byte{MDelimStart})[:], data[:]...)
			dataToSend = append(data[:], ([]byte{MDelimStop})[:]...)
		} else {
			dataToSend = append(data[:], ([]byte{MDelimStop})[:]...)
		}
		ac.TpLogDump(atmi.LOG_DEBUG, "Sending message (etx/stx)", dataToSend, len(dataToSend))

		nw, err := con.writer.Write(dataToSend)

		if nil != err {
			ac.TpLogError("Failed to write data to socket: %s", err)
		}

		ac.TpLogInfo("Written %d bytes to socket", nw)
	}

	return nil
}
