package main

import (
	"encoding/json"

	atmi "github.com/endurox-dev/endurox-go"
)

type TestJSONMsg struct {
	StringField  string `json:"StringField"`
	StringField2 string `json:"StringField2"`
	NumField     int    `json:"NumField"`
	NumField2    int    `json:"NumField2"`
	BoolField    bool   `json:"BoolField"`
	BoolField2   bool   `json:"BoolField2"`
}

//JSON Service, we will receive JSON block
//@param ac ATMI Context
//@param svc Service call information
func JSONSV(ac *atmi.ATMICtx, svc *atmi.TPSVCINFO) {

	var msg TestJSONMsg
	ret := SUCCEED

	//Get UBF Handler
	jb, _ := ac.CastToJSON(&svc.Data)

	//Return to the caller
	defer func() {
		if SUCCEED == ret {
			ac.TpReturn(atmi.TPSUCCESS, 0, jb, 0)
		} else {
			ac.TpReturn(atmi.TPFAIL, 0, jb, 0)
		}
	}()

	ac.TpLogWarn("Got json request...")

	//Resize buffer, to have some more space to return data in
	if err := jb.TpRealloc(1024); err != nil {
		ac.TpLogError("TpRealloc() Got error: %d:[%s]\n", err.Code(), err.Message())
		ret = FAIL
		return
	}

	ac.TpLogDump(atmi.LOG_INFO, "Got request buffer", jb.GetJSON(), len(jb.GetJSON()))
	//Umarshal the data, copy to *2 and marshal back to buffer
	jerr := json.Unmarshal(jb.GetJSON(), &msg)
	if jerr != nil {
		ac.TpLogError("Unmarshal: %s", jerr)
		ret = FAIL
		return
	}
	msg.StringField2 = msg.StringField
	msg.BoolField2 = msg.BoolField
	msg.NumField2 = msg.NumField

	val, jerr := json.Marshal(msg)
	if jerr != nil {
		ac.TpLogError("Marshal: %s", jerr)
		ret = FAIL
		return
	}

	ac.TpLogDump(atmi.LOG_INFO, "Built response", val, len(val))

	//Set the data in return buffer
	if err := jb.SetJSON(val); err != nil {
		ac.TpLogError("Failed to return json buffer %s", err.Message())
		ret = FAIL
		return
	}

	ac.TpLogDump(atmi.LOG_INFO, "Responding with buffer", jb.GetJSON(), len(jb.GetJSON()))

	return
}
