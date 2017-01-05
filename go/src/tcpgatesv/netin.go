/*
** Network -> Enduro/X
**
** @file atmiout.go
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

//We have recieved new call from Network
//So shall wait for new ATMI context & send the message in
//This should be run on go routine.
//@param data 	Data received from Network
//@param bool	set to false if do not need to continue (i.e. close conn)
func NetDispatchCall(con *ExCon, data []byte, corr string) {
	//TODO: Setup UBF buffer, load the fields

	buf, err := ac.NewUBF(len(data) + 1024)
	if nil != err {
		ac.TpLogError("Failed to allocate buffer: [%s] - dropping incoming message",
			err.Error())
		return
	}

	/*
		TODO: Load the fields
		buf.BChg(u.EX_CC_CMD, 0, "g")
		buf.BChg(u.EX_CC_LOOKUPSECTION, 0, fmt.Sprintf("%s/%s", PROGSECTION, os.Getenv("NDRX_CCTAG")))
	*/

}

//Dispatch connection answer
//@param call 	Call data block (what caller thread actually made)
//@param data	Data block received from network
//@param bool	ptr for finish off parameter
func NetDispatchConAnswer(call *DataBlock, data []byte, doContinue *bool) {
	call.atmi_chan <- data
	*doContinue = false //Do not continue - close thread

	//Remove from corelator lists
	RemoveFromCallLists(call)
}

//Dispatch connection answer
//@param call 	Call data block (what caller thread actually made)
//@param data	Data block received from network
//@param bool	ptr for finish off parameter
func NetDispatchCorAnswer(call *DataBlock, data []byte, doContinue *bool) {
	call.atmi_chan <- data //Send the data to caller
	//Remove from corelator lists
	RemoveFromCallLists(call)
}

//Get correlator id
func NetGetCorID(data []byte) (string, error) {

	return "", nil
}
