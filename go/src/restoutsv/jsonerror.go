/**
 * @brief Parse the incoming json message and read the fields cotnaining json string & Number
 *   This will use Enduro/X provided Exparson package. Due to fact that there are
 *   problems with golang to dynamicall parse the string & number in one step.
 *
 * @file jsonerror.go
 */
/* -----------------------------------------------------------------------------
 * Enduro/X Middleware Platform for Distributed Transaction Processing
 * Copyright (C) 2009-2016, ATR Baltic, Ltd. All Rights Reserved.
 * Copyright (C) 2017-2018, Mavimax, Ltd. All Rights Reserved.
 * This software is released under one of the following licenses:
 * GPL or Mavimax's license for commercial use.
 * -----------------------------------------------------------------------------
 * GPL license:
 * 
 * This program is free software; you can redistribute it and/or modify it under
 * the terms of the GNU General Public License as published by the Free Software
 * Foundation; either version 3 of the License, or (at your option) any later
 * version.
 *
 * This program is distributed in the hope that it will be useful, but WITHOUT ANY
 * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A
 * PARTICULAR PURPOSE. See the GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License along with
 * this program; if not, write to the Free Software Foundation, Inc., 59 Temple
 * Place, Suite 330, Boston, MA 02111-1307 USA
 *
 * -----------------------------------------------------------------------------
 * A commercial use license is available from Mavimax, Ltd
 * contact@mavimax.com
 * -----------------------------------------------------------------------------
 */
package main

import (
	"unsafe"
	"errors"
	atmi "github.com/endurox-dev/endurox-go"
)

/*
#cgo pkg-config: atmisrvinteg

#include <stdlib.h>
#include <exparson.h>
 */
import "C"

//Get the JSON error fields (if they are present)
//@param ac	ATMI context
//@param json	JSON block recevied from network
//@return <Error code>, <Error string>, ATMIError (if set to nil, all other are present)
func JSONErrorGet(ac *atmi.ATMICtx, json *string, jcodefld string, jmessagefld string) (int, string, error) {

	c_codefld := C.CString(jcodefld)
	defer C.free(unsafe.Pointer(c_codefld))

	c_jmessagefld := C.CString(jmessagefld)
	defer C.free(unsafe.Pointer(c_jmessagefld))

	c_buffer := C.CString(*json)
	defer C.free(unsafe.Pointer(c_buffer))

	root_value := C.exjson_parse_string_with_comments(c_buffer);
	defer C.exjson_value_free(root_value);

	if nil==root_value {
		return FAIL, "", errors.New("Invalid JSON (1)!");
	}

	root_object := C.exjson_value_get_object(root_value);

	if nil==root_object {
		return FAIL, "",  errors.New("Invalid JSON (2)!");
	}

	if (nil!=root_object) {

		codeVal:=C.exjson_object_dotget_number(root_object, c_codefld)
		messageVal:=C.exjson_object_get_string(root_object, c_jmessagefld)

		errs:=C.GoString(messageVal)

		if errs=="" {
			ac.TpLogInfo("No error string in response");
		}

		return int(C.int(codeVal)), C.GoString(messageVal), nil
	}

	return FAIL, "", errors.New("Invalid JSON (3)");
}
