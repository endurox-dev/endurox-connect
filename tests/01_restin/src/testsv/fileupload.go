/**
 * File upload handler
 */
package main

import (
	"os/exec"
	"strings"
	u "ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	SUCCEED = 0
	FAIL    = -1
)

//Check if the failure is due to internal workings
//Check the error code that it matches
//If so return message "DIS"
func UPLDERR(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED

	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)
	ac.TpLogSetReqFile(ub, "", "")
	//Return to the caller
	defer func() {

		ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Returning:")

		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	tperrno, _ := ub.BGetInt(u.EX_IF_ECODE, 0)
	tperrsrc, _ := ub.BGetString(u.EX_IF_ERRSRC, 0)
	tpstrerror, _ := ub.BGetString(u.EX_IF_EMSG, 0)

	ac.TpLogDebug("Got error tperrno: %d src: %s msg: %s", tperrno, tperrsrc, tpstrerror)

	if tperrno == atmi.TPEOS && tperrsrc == "R" {
		//This is OK condition, set the message
		ub.BChg(u.EX_IF_RSPDATA, 0, "DISK FAILURE")
	} else {
		ub.BChg(u.EX_IF_RSPDATA, 0, "OTHER FAILURE: "+tperrsrc)
	}

	//Set return code OK

	ub.BChg(u.EX_NETRCODE, 0, "200")

}

//This will receive the files on disk, perform the checksum
//And will return <file name:chksum>\n on reply body
func FILEUPLOAD(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	ret := SUCCEED
	var reply string
	//Get UBF Handler
	ub, _ := ac.CastToUBF(&svc.Data)

	ac.TpLogSetReqFile(ub, "", "")

	//Return to the caller
	defer func() {

		ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Response buf:")
		ac.TpLogCloseReqFile()
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, ub, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, ub, 0)
		}
	}()

	//Print the buffer to stdout
	ub.TpLogPrintUBF(atmi.LOG_DEBUG, "Incoming request:")

	//Loop over the buffer... we will mark file No 1.ac

	occs, errU := ub.BOccur(u.EX_IF_REQFILEDISK)

	if nil != errU {
		ac.TpLogError("TESTERROR: Error reading EX_IF_REQFILEDISK")
		ret = FAIL
		return
	}

	if occs < 3 {
		ac.TpLogError("TESTERROR: Expected atleast 3 occs (got %d)", occs)
		ret = FAIL
		return
	}

	//Resize buffer, to have some more space
	used, _ := ub.BUsed()
	if err := ub.TpRealloc(used + 1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	//If occs==3, the we leave first file on disk
	//In other cases all files from disk must be deleted

	//Process each of the files...
	for i := 0; i < occs; i++ {

		if 3 == occs {
			//Set the keep flag
			if 2 == i {
				ub.BChg(u.EX_IF_RSPFILEACTION, i, "K")
			} else {
				ub.BChg(u.EX_IF_RSPFILEACTION, i, "D")
			}
		} else {
			//Having other occs, just delete (default)
		}

		//Get the file on disk
		diskname, errU := ub.BGetString(u.EX_IF_REQFILEDISK, i)
		if nil != errU {
			ac.TpLogError("Failed to get EX_IF_REQFILEDISK[%d]: %s", i, errU.Error())
			ret = FAIL
			return
		}
		ac.TpLogInfo("File name on disk is: [%s]", diskname)

		//Get the file name at user side
		fname, errU := ub.BGetString(u.EX_IF_REQFILENAME, i)
		if nil != errU {
			ac.TpLogError("Failed to get EX_IF_REQFILENAME[%d]: %s", i, errU.Error())
			ret = FAIL
			return
		}
		ac.TpLogInfo("Form file name (user side): [%s]", fname)

		//Get the mime
		mime, errU := ub.BGetString(u.EX_IF_REQFILEMIME, i)
		if nil != errU {
			ac.TpLogError("Failed to get EX_IF_REQFILEMIME[%d]: %s", i, errU.Error())
			ret = FAIL
			return
		}
		ac.TpLogInfo("Mime: [%s]", mime)

		//Get the form field name
		formname, errU := ub.BGetString(u.EX_IF_REQFILEFORM, i)
		if nil != errU {
			ac.TpLogError("Failed to get EX_IF_REQFILEFORM[%d]: %s", i, errU.Error())
			ret = FAIL
			return
		}
		ac.TpLogInfo("Form field: [%s]", formname)

		//Disk file name shall be different that user
		//this is nature of the test
		if diskname == fname {
			ac.TpLogError("Disk file name [%s] shall not match the logical upload file name [%s] at occ %d",
				diskname, fname, i)
			ret = FAIL
			return
		}

		cmd := exec.Command("cksum", diskname)
		out, err := cmd.Output()

		if err != nil {
			ac.TpLogError("Failed to get checksum for [%s]: %s", diskname, err.Error())
			ret = FAIL
			return
		}

		strout := strings.Replace(string(out), diskname, fname, -1)
		reply += (strout + "\n")

	}

	//Get the chekcsum
	ac.TpLogInfo("Reply: [%s]", reply)
	//Load the reply
	if errU := ub.BChg(u.EX_IF_RSPDATA, 0, reply); nil != errU {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]", errU.Code(), errU.Message())
		ret = FAIL
		return
	}

	return
}
