/**
 * @brief File upload handler (multi-part form)
 *
 * @file fileupload.go
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
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"ubftab"

	atmi "github.com/endurox-dev/endurox-go"
)

const (
	FILES_FLAG_KEEP   = "K" //Keep the files after the upload service finish
	FILES_FLAG_DELETE = "D" //Delete the file (RFU)
)

//This is used to strack
//additional request details
//Including list of files uploaded
type RequestContext struct {
	errSrc   string
	fileList []string
}

//Prepare file upload (request part, download & prepare the UBF buffer)
//@param ac ATMI Context
//@param bufu ATMI buffer (must be UBF) for EXT processing
//@param svc Target service
//@param req HTTP request obj
//@param rctx request context attributes
//@return ATMI error or nil
func handleFileUploadReq(ac *atmi.ATMICtx, bufu *atmi.TypedUBF, svc *ServiceMap,
	r *http.Request, rctx *RequestContext) atmi.ATMIError {

	var n int
	var err error
	var occ = 0
	// define pointers for the multipart reader and its parts
	var mr *multipart.Reader
	var part *multipart.Part

	ac.TpLogInfo("Receiving file upload...")

	if mr, err = r.MultipartReader(); err != nil {

		ac.TpLogError("Failed to open multi-part reader: %s", err.Error())
		return atmi.NewCustomATMIError(atmi.TPESYSTEM, fmt.Sprintf(
			"Failed to open multi-part reader: %s", err.Error()))
	}

	// buffer to be used for reading bytes from files
	chunk := make([]byte, 4096)

	for {
		var tempfile *os.File
		var filesize int
		var uploaded bool

		if part, err = mr.NextPart(); err != nil {
			if err != io.EOF {
				ac.TpLogError("Error while fetching next part: %s", err.Error())
				return atmi.NewCustomATMIError(atmi.TPESYSTEM,
					fmt.Sprintf("Error while fetching next part: %s", err.Error()))

			} else {
				ac.TpLogInfo("Multipart upload OK")
				return nil
			}
		}

		ac.TpLogDebug("Uploaded filename occ=%d: %s", occ, part.FileName())
		ac.TpLogDebug("Uploaded mimetype occ=%d: %s", occ, part.Header)

		//Add the file names to the buffer
		if errU := bufu.BAdd(ubftab.EX_IF_REQFILENAME, part.FileName()); nil != errU {
			ac.TpLogError("Failed to add EX_IF_REQFILENAME[%d]: %s", occ, errU.Error())
			return atmi.NewCustomATMIError(atmi.TPESYSTEM,
				fmt.Sprintf("Failed to add EX_IF_REQFILENAME[%d]: %s", occ, errU.Error()))
		}

		if errU := bufu.BAdd(ubftab.EX_IF_REQFILEFORM, part.FormName()); nil != errU {
			ac.TpLogError("Failed to add EX_IF_REQFILEFORM[%d]: %s", occ, errU.Error())
			return atmi.NewCustomATMIError(atmi.TPESYSTEM,
				fmt.Sprintf("Failed to add EX_IF_REQFILEFORM[%d]: %s", occ, errU.Error()))
		}

		if errU := bufu.BAdd(ubftab.EX_IF_REQFILEMIME, part.Header.Get("Content-Type")); nil != errU {
			ac.TpLogError("Failed to add EX_IF_REQFILEMIME[%d]: %s", occ, errU.Error())
			return atmi.NewCustomATMIError(atmi.TPESYSTEM,
				fmt.Sprintf("Failed to add EX_IF_REQFILEMIME[%d]: %s", occ, errU.Error()))
		}

		//Add the file name to received rctx

		tempfile, err = ioutil.TempFile(svc.Tempdir, fmt.Sprintf("%s-%s", progsection, M_cctag))
		if err != nil {
			return atmi.NewCustomATMIError(atmi.TPEOS,
				fmt.Sprintf("Error while creating temp file: %s", err.Error()))
		}

		ac.TpLogInfo("Got file name for file occ %d: %s", occ, tempfile.Name())
		rctx.fileList = append(rctx.fileList, tempfile.Name())

		if errU := bufu.BAdd(ubftab.EX_IF_REQFILEDISK, tempfile.Name()); nil != errU {
			ac.TpLogError("Failed to add EX_IF_REQFILEDISK[%d]: %s", occ, errU.Error())
			return atmi.NewCustomATMIError(atmi.TPESYSTEM,
				fmt.Sprintf("Failed to add EX_IF_REQFILEDISK[%d]: %s", occ, errU.Error()))
		}

		defer tempfile.Close()

		// Read all parts of the file & write off to disk...
		for !uploaded {
			if n, err = part.Read(chunk); err != nil {
				if err != io.EOF {
					ac.TpLogError("Error reading chunk: %s", err.Error())
					return atmi.NewCustomATMIError(atmi.TPESYSTEM,
						fmt.Sprintf("Error reading chunk: %s", err.Error()))
				}
				uploaded = true
			}

			if n, err = tempfile.Write(chunk[:n]); err != nil {
				ac.TpLogError("Error writing chunk to [%s]: %s", tempfile.Name(), err.Error())
				return atmi.NewCustomATMIError(atmi.TPEOS,
					fmt.Sprintf("Error writing chunk [%s] to: %s", tempfile.Name(), err.Error()))
			}
			filesize += n
		}

		ac.TpLogInfo("Uploaded file [%s] size: %d bytes", tempfile.Name(), filesize)

		occ++
	}

}

//Handle response after the file processed
//@param ac ATMI Context
//@param ubfu UBF buffer used for request handling
//@param service definition
//@param rctx request boject
//@return ATMI error if any
func handleFileUploadRsp(ac *atmi.ATMICtx, bufu *atmi.TypedUBF, svc *ServiceMap,
	rctx *RequestContext) atmi.ATMIError {

	for i, s := range rctx.fileList {

		//Check the action by index

		action, _ := bufu.BGetString(ubftab.EX_IF_RSPFILEACTION, i)

		if strings.Contains(action, FILES_FLAG_KEEP) {
			ac.TpLogInfo("Keeping file [%s] (occ %d)", s, i)
		} else {
			ac.TpLogInfo("Removing file [%s] (occ %d)", s, i)
			os.Remove(s)
		}
	}

	return nil
}

/* vim: set ts=4 sw=4 et smartindent: */
