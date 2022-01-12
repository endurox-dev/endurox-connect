/**
 * @brief Message framing, varous format support, including socket reading support
 *
 * @file msgframe.go
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
	case FRAME_DELIM_STOP:
		MFramingLen = 0
		ac.TpLogInfo("Stopping delimiter: %x", MDelimStop)

		if MFramingKeepHdr {
			ac.TpLogError("Invalid config: framing_keephdr not support with delimiters!")
			return errors.New("Invalid config: framing_keephdr not support with delimiters!")
		}

		break
	case FRAME_DELIM_BOTH:
		MFramingLen = 0
		ac.TpLogInfo("Start delimiter %x, Stop delimiter: %x",
			MDelimStart, MDelimStop)

		if MFramingKeepHdr {
			ac.TpLogError("Invalid config: framing_keephdr not support with delimiters!")
			return errors.New("Invalid config: framing_keephdr not support with delimiters!")
		}

		break
	default:
		ac.TpLogError("Invalid framing...")
		return errors.New("Invalid message framing...")
	}

	MFramingLenReal = MFramingLen

	if MFramingLen > 0 && MFramingOffset > 0 {
		ac.TpLogInfo("Incrementing message prefix len from %d to %d due to offset",
			MFramingLen, MFramingLen+MFramingOffset)

		MFramingLen += MFramingOffset
	}

	ac.TpLogInfo("Framing header bytes %d len bytes %d", MFramingLen, MFramingLenReal)

	return nil
}

//Read the message from connection
//@param con 	Connection handler
//@return <Binary message read>, <Error or nil>
func GetMessage(ac *atmi.ATMICtx, con *ExCon) ([]byte, error) {

	if MFramingLen > 0 {

		header := make([]byte, MFramingLen)
		headerSwapped := make([]byte, MFramingLen)
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

		//Copy off header bytes for swap manipulations
		copy(headerSwapped, header[:])

		//Swap bytes if needed
		if MFramingHalfSwap {
			ac.TpLogDump(atmi.LOG_DEBUG, "Got message prefix (before swapping)",
				headerSwapped, len(headerSwapped))
			half := MFramingLenReal / 2
			for i := 0; i < half; i++ {
				tmp := headerSwapped[MFramingOffset+i]
				headerSwapped[MFramingOffset+i] = headerSwapped[MFramingOffset+half+i]
				headerSwapped[MFramingOffset+half+i] = tmp
			}
		}

		ac.TpLogDump(atmi.LOG_DEBUG, "Got message prefix (final - for len proc)",
			headerSwapped, len(headerSwapped))

		//Decode the length now...
		if MFramingCode != FRAME_ASCII && MFramingCode != FRAME_ASCII_ILEN {

			for i := MFramingOffset; i < MFramingLen; i++ {
				//Move the current byte to front
				mlen <<= 8
				switch MFramingCode {
				case FRAME_LITTLE_ENDIAN, FRAME_LITTLE_ENDIAN_ILEN:
					//Add current byte
					mlen |= int64(headerSwapped[i])
					break
				case FRAME_BIG_ENDIAN, FRAME_BIG_ENDIAN_ILEN:
					//Add current byte, but take from older
					mlen |= int64(headerSwapped[int(MFramingLen-1)-i])
					break
				}
			}
		} else {
			mlenStr = string(headerSwapped)
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
		var data []byte
		var data_space []byte

		//Feature #531
		if MFramingKeepHdr {

			data = make([]byte, mlen+int64(MFramingLen))
			data_space = data[MFramingLen:]

			//Restore the header please...
			copy(data, header)

		} else {
			data = make([]byte, mlen)
			data_space = data[:]
		}

		n, err = io.ReadFull(con.reader, data_space)

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

		if MFramingKeepHdr {
			ac.TpLogDump(atmi.LOG_DEBUG, "FULL (incl len hdr) Message read",
				data, len(data))
		} else {
			ac.TpLogDump(atmi.LOG_DEBUG, "Message (no len hdr) read",
				data, len(data))
		}

		return data, nil

	} else {
		ac.TpLogInfo("About to read message until delimiter 0x%x", MDelimStop)

		//If we use delimiter, then read pu till that
		data, err := con.reader.ReadBytes(MDelimStop)

		if err != nil {

			ac.TpLogError("Failed to read message with %x seperator: %s",
				MDelimStop, err.Error())
			return nil, err
		}

		// Bug #103, seems like the data returned by ReadSlice is somehow shared
		// and not reallocated... Thus make a new buffer
		//data := make([]byte, len(idata))
		//copy(data, idata)

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
		}

		//Stip of flast elem
		data = data[:len(data)-1]

		ac.TpLogDump(atmi.LOG_DEBUG, "Message read", data, len(data))

		return data, nil
	}
}

//Put message on socket
func PutMessage(ac *atmi.ATMICtx, con *ExCon, data []byte) error {

	ac.TpLogInfo("Building outgoing message: len hdr bytes %d (real: %d)",
		MFramingLen, MFramingLenReal)

	ac.TpLogDump(atmi.LOG_DEBUG, "Preparing message for sending",
		data, len(data))

	if MFramingLen > 0 {
		var mlen int64 = int64(len(data))
		header := make([]byte, MFramingLenReal)

		//Test that we have a place for length bytes to be installed
		if MFramingKeepHdr && mlen < int64(MFramingLen) {
			errMsg := fmt.Sprintf("No space outoing message to install offset/len "+
				"pfx: offset: %d pfx len: %d full header: %d",
				MFramingOffset, MFramingLenReal, MFramingLen)
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		if MFamingInclPfxLen {
			if !MFramingKeepHdr {
				mlen += int64(MFramingLen)
			}
		} else {

			if MFramingKeepHdr {
				//Do not include length bytes including offset as this is already
				//filled init
				mlen -= int64(MFramingLen)
			}
		}

		ac.TpLogDebug("Message len set to %d", mlen)

		//Generate the header
		if MFramingCode != FRAME_ASCII && MFramingCode != FRAME_ASCII_ILEN {
			for i := 0; i < MFramingLenReal; i++ {
				switch MFramingCode {
				case FRAME_LITTLE_ENDIAN, FRAME_LITTLE_ENDIAN_ILEN:
					//So the least significant byte goes to end the array
					header[(MFramingLenReal-1)-i] = byte(mlen & 0xff)
					break
				case FRAME_BIG_ENDIAN, FRAME_BIG_ENDIAN_ILEN:
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

		//Swap bytes if needed
		if MFramingHalfSwap {
			ac.TpLogDump(atmi.LOG_INFO, "Built message header (before swapping)",
				header, len(header))
			half := MFramingLenReal / 2
			for i := 0; i < half; i++ {
				tmp := header[i]
				header[i] = header[half+i]
				header[half+i] = tmp
			}
		}

		// Print len
		ac.TpLogDump(atmi.LOG_INFO, "Built message header (final - len only)",
			header, len(header))

		//About to send message.
		dataToSend := []byte{}
		hdr_bytes := 0
		if MFramingKeepHdr {

			//In this case at specific offset we need to copy data from prepared
			//len bytes
			for i := 0; i < MFramingLenReal; i++ {
				data[MFramingOffset+i] = header[i]
			}

			dataToSend = data
		} else {

			var err error
			//We can sender header separetelly
			//Better two sends than... copy..
			//dataToSend = append(header[:], data...)

			ac.TpLogDump(atmi.LOG_DEBUG, "Sending header",
				header, len(header))

			hdr_bytes, err = con.con.Write(header)
			//hdr_bytes, err = con.writer.Write(header)

			if nil != err {
				errMsg := fmt.Sprintf("Failed to send header socket: %s", err)
				ac.TpLogError(errMsg)
				return errors.New(errMsg)
			}

			dataToSend = data
		}

		ac.TpLogDump(atmi.LOG_DEBUG, "Sending message, w len pfx",
			dataToSend, len(dataToSend))

		nw, err := con.con.Write(dataToSend)
		//nw, err := con.writer.Write(dataToSend)

		if nil != err {
			errMsg := fmt.Sprintf("Failed to write data to socket: %s", err)
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		ac.TpLogInfo("Written %d bytes to socket flush", nw+hdr_bytes)

	} else {

		var dataToSend []byte
		hdr_bytes := 0

		//Put STX
		if MFramingCode == FRAME_DELIM_BOTH {
			//dataToSend = append(([]byte{MDelimStart})[:], data[:]...)
			//Send start delimiter...
			var err error

			stx_data := ([]byte{MDelimStart})[:]

			ac.TpLogDump(atmi.LOG_DEBUG, "Sending STX message", stx_data, len(stx_data))

			//hdr_bytes, err = con.writer.Write(stx_data)
			hdr_bytes, err = con.con.Write(stx_data)

			if nil != err {
				errMsg := fmt.Sprintf("Failed to send STX: %s", err)
				ac.TpLogError(errMsg)
				return errors.New(errMsg)
			}
		}

		//ETX append with tx
		//dataToSend = append(data[:], ([]byte{MDelimStop})[:]...)

		ac.TpLogDump(atmi.LOG_DEBUG, "Sending message", dataToSend, len(dataToSend))

		//		nw, err := con.writer.Write(data)
		nw, err := con.con.Write(data)

		if nil != err {
			errMsg := fmt.Sprintf("Failed to write data to socket: %s", err)
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		etx_data := ([]byte{MDelimStop})[:]

		ac.TpLogDump(atmi.LOG_DEBUG, "Sending ETX message", etx_data, len(etx_data))

		//etx_bytes, err := con.writer.Write(etx_data)
		etx_bytes, err := con.con.Write(etx_data)

		if nil != err {
			errMsg := fmt.Sprintf("Failed to write to socket etx data: %s", err)
			ac.TpLogError(errMsg)
			return errors.New(errMsg)
		}

		/*
			err = con.writer.Flush()

			if nil != err {
				errMsg := fmt.Sprintf("Failed to flush socket: %s", err)
				ac.TpLogError(errMsg)
				return errors.New(errMsg)
			}
		*/
		ac.TpLogInfo("Written %d bytes to socket", nw+hdr_bytes+etx_bytes)

	}

	return nil
}

/* vim: set ts=4 sw=4 et smartindent: */
