/**
 * @brief CA Root handler
 *
 * @file caroots.go
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
package exutil

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"

	atmi "github.com/endurox-dev/endurox-go"
)

var MRootCAs *x509.CertPool = nil //Loaded root cer

//Load root certificate authorities from configured string
func LoadRootCAs(ac *atmi.ATMICtx, carootsfiles string) error {

	//Split by ;
	certs := strings.Replace(carootsfiles, " ", "", -1)
	certs = strings.Replace(carootsfiles, "\t", "", -1)

	crt_arr := strings.Split(certs, ";")
	crt_num := len(crt_arr)

	MRootCAs = x509.NewCertPool()

	for i := 0; i < crt_num; i++ {

		ac.TpLogInfo("Loading root CA: %s", crt_arr[i])

		caCert, err := ioutil.ReadFile(crt_arr[i])
		if err != nil {
			ac.TpLogError("Failed to read CA root cert [%s]: %s", crt_arr[i], err)
			return fmt.Errorf("Failed to read CA root cert [%s]: %s", crt_arr[i], err)
		}
		if !MRootCAs.AppendCertsFromPEM(caCert) {
			ac.TpLogError("Failed to load/parse CA root cert [%s]", crt_arr[i])
			return fmt.Errorf("Failed to load/parse CA root cert [%s]", crt_arr[i])
        }
	}

	ac.TpLogInfo("Roots loaded OK")
	return nil

}

/* vim: set ts=4 sw=4 et smartindent: */
