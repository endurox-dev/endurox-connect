#!/bin/bash

#
# @(#) Test 01 - Rest-IN interface tests...
#

pushd .

rm -rf runtime/log 2>/dev/null
mkdir runtime/log 

cd runtime

LOGFILE=log/shell_out.log
# truncate the file
> $LOGFILE

#
# Create the env..
#

#
# So we need to add some demo server
# We need to add server process here + we need to register ubftab (test.fd)
#
xadmin provision -d \
        -vusv1_name=testsv \
        -vusv1=y \
        -vusv1_sysopt='-e ${NDRX_APPHOME}/log/testsv.log -r' \
        -vaddubf=test.fd \
        -vucl1=y \
        -vusv1_cmdline=restincl \
        -vusv1_tag=RESTIN \
        -vusv1_log='${NDRX_APPHOME}/log/restin.log' \
        -vtimeout=2

cd conf

# Remove certificate files
rm localhost* 2>/dev/null

# Generate new ceritificate
./gencert.sh localhost 

. settest1

# So we are in runtime directory
cd ..
# Be on safe side...
unset NDRX_CCTAG 
xadmin start -y

# Let restin to start
sleep 2

#
# Generic exit function
#
function go_out {
    echo "Test exiting with: $1"
    xadmin stop -y
    xadmin down -y

    popd 2>/dev/null
    exit $1
}

###############################################################################
echo "VIEW TEST - normal call, view errors"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok 2>&1 )`

	RSP_EXPECTED="{\"REQUEST1\":{\"tshort1\":8,\
\"tlong1\":11111,\
\"tstring1\":[\"HELLO RESPONSE\",\"INCOMING TEST\",\"\"],\
\"rspcode\":\"0\",\
\"rspmessage\":\"\"}}"
        echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 23
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "VIEW TEST - normal call, error code in response"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok/errsucc 2>&1 )`

	RSP_EXPECTED="{\"REQUEST1\":{\"tshort1\":8,\
\"tlong1\":11111,\
\"tstring1\":[\"HELLO RESPONSE\",\"INCOMING TEST\",\"\"],\
\"rspcode\":\"0\",\
\"rspmessage\":\"SUCCEED\"}}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 24
	fi
done

} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW TEST - invalid json - expected response object to be returned"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
	RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort: 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok/errsucc 2>&1 )`

	RSP_EXPECTED="{\"RSPV\":{\"rspcode\":\"4\",\"rspmessage\":\"4:T\"}}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 25
	fi
done

} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW TEST return different object"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok/diffbuff 2>&1 )`

	RSP_EXPECTED="{\"REQUEST2\":{\"tshort2\":5,\
\"tlong2\":77777,\
\"tstring2\":\"INCOMING TEST\"}}"
	echo "Response: [$RSP]"
	
	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 26
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW TEST return different object"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok/diffbuff 2>&1 )`

	RSP_EXPECTED="{\"REQUEST2\":{\"tshort2\":5,\
\"tlong2\":77777,\
\"tstring2\":\"INCOMING TEST\"}}"
	echo "Response: [$RSP]"
	
	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 27
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "VIEW error not install, as succeed and no fields available..."
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/ok/noerr/errosucc 2>&1 )`

	RSP_EXPECTED="{\"REQUEST2\":{\"tshort2\":5,\
\"tlong2\":77777,\
\"tstring2\":\"INCOMING TEST\"}}"
	echo "Response: [$RSP]"
	
	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 28
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW error service failure"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/fail 2>&1 )`

	RSP_EXPECTED="{\"RSPV\":{\"rspcode\":\"11\",\"rspmessage\":\"11:\"}}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 29
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW error service failure"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/fail/norsp 2>&1 )`

	RSP_EXPECTED="{}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 30
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW error, async no nulls in rsp"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/async 2>&1 )`

	RSP_EXPECTED="{\"RSPV\":{\"rspcode\":\"0\",\"rspmessage\":\"SUC\"}}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 31
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW errors, stipped NULLs, async + echo"
###############################################################################
{
for i in {1..100}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
http://localhost:8080/view/async/echo 2>&1 )`

	RSP_EXPECTED="{\"REQUEST1\":{\"tshort1\":5,\
\"tlong1\":77777,\
\"tstring1\":[\"\",\"INCOMING TEST\"]}}"
        echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 32
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "Views HTTP case, ok"
###############################################################################
{
for i in {1..100}
do

	RSP=`curl -i -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
	http://localhost:8080/view/httpe/ok 2>&1`

	if [[ "$RSP" != *"HELLO RESPONSE"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [HELLO RESPONSE] to appear"
		go_out 33
	fi
	
	RSP_EXPECTED="200"
	
	echo "Response: [$RSP]"

	if [[ "$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 34
	fi
	
done

} >> $LOGFILE 2>&1
###############################################################################
echo "Views HTTP case, svc fail"
###############################################################################
{
for i in {1..100}
do

	RSP=`curl -i -H "Content-Type: application/json" -X POST -d \
"{ \
	\"REQUEST1\": { \
		\"tshort1\": 5, \
		\"tlong1\": 77777, \
		\"tstring1\": [\"\", \"INCOMING TEST\"] \
	} \
}" \
	http://localhost:8080/view/httpe/fail 2>&1`

	if [[ "$RSP" != *"REQUEST2"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [REQUEST2] to appear"
		go_out 35
	fi

	if [[ "$RSP" != *"INCOMING"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [INCOMING] to appear"
		go_out 36
	fi
	
	RSP_EXPECTED="500"
	
	echo "Response: [$RSP]"

	if [[ "$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 37
	fi
	
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW TEST - view errors, line in req buffer"
###############################################################################
{
for i in {1..100}
do

    # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
    \"REQUEST1\": { \
        \"tshort1\": 5, \
        \"tlong1\": 77777, \
        \"tstring1\": [\"\", \"INCOMING TEST\"] \
    } \
}" \
http://localhost:8080/view/fail/einline 2>&1 )`

    RSP_EXPECTED="{\"REQUEST1\":{\"tshort1\":8,\
\"tlong1\":11111,\
\"tstring1\":[\"HELLO RESPONSE\",\"INCOMING TEST\",\"\"],\
\"rspcode\":\"11"
        echo "Response: [$RSP]"

    if [[ "X$RSP" != "X$RSP_EXPECTED"* ]]; then
        echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
        go_out 38
    fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "VIEW TEST - view errors, error object view first"
###############################################################################
{
for i in {1..100}
do

    # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{ \
    \"REQUEST1\": { \
        \"tshort1\": 5, \
        \"tlong1\": 77777, \
        \"tstring1\": [\"\", \"INCOMING TEST\"] \
    } \
}" \
http://localhost:8080/view/fail/efirst 2>&1 )`

    RSP_EXPECTED="{\"RSPV\":{\"rspcode\":\"11\",\"rspmessage\":\"11:\"}}"
        echo "Response: [$RSP]"

    if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
        echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
        go_out 39
    fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "TLS Test"
###############################################################################

# Stop the non-ssl client
xadmin sc -t RESTIN
export NDRX_CCTAG="TLS"

restincl > ./log/restin-tls.log 2>&1 & 
RPID=$!

# Let it start
sleep 1
{
for i in {1..1000}
do

        RSP=`curl --insecure -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
https://localhost:8080/svc1`


        RSP_EXPECTED="{\"T_SHORT_FLD\":123,\
\"T_SHORT_2_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_LONG_2_FLD\":444444444,\
\"T_CHAR_FLD\":\"A\",\
\"T_CHAR_2_FLD\":\"A\",\
\"T_FLOAT_FLD\":1.330000,\
\"T_FLOAT_2_FLD\":1.330000,\
\"T_DOUBLE_FLD\":4444.333300,\
\"T_DOUBLE_2_FLD\":4444.333300,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_STRING_2_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\",\
\"T_CARRAY_2_FLD\":\"SGVsbG8=\",\
\"error_code1\":0,\
\"error_message1\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 22
	fi
done
} >> $LOGFILE 2>&1
unset NDRX_CCTAG
kill -2 $RPID
sleep 1

# Start the non ssl version
xadmin bc -t RESTIN

# Let it open connection..
sleep 2

###############################################################################
echo "Binary buffer, async, echo"
###############################################################################
{
for i in {1..1000}
do
	rm tmp.out 2>/dev/null
	# Having a -i means to print the headers
	RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
	--data-binary "@../binary.test.request" http://localhost:8080/binary/ok/async/echo > tmp.out 2>&1)`

	DIFF=`diff tmp.out ../binary.test.request`
	echo "Response: [$DIFF]"

	if [[ "X$DIFF" != "X" ]]; then
		echo "The response [tmp.out] does not match binary.test.request!"
		go_out 21
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Binary buffer, async"
###############################################################################
{
for i in {1..1000}
do
	rm tmp.out 2>/dev/null
        # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
        --data-binary "@../binary.test.request" http://localhost:8080/binary/ok/async )`

        RSP_EXPECTED="0: SUCCEED"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 20
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Binary buffer, fail"
###############################################################################
{
for i in {1..1000}
do
	rm tmp.out 2>/dev/null
        # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
        --data-binary "@../binary.test.request" http://localhost:8080/binary/fail )`

        RSP_EXPECTED="11: 11:TPESVCFAIL (last error 11: Service returned 1)"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 19
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Binary buffer, ok"
###############################################################################
{
for i in {1..1000}
do
	rm tmp.out 2>/dev/null
	# Having a -i means to print the headers
	RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
	--data-binary "@../binary.test.request" http://localhost:8080/binary/ok  > tmp.out )`

	DIFF=`diff tmp.out ../binary.test.response`
	echo "Response: [$DIFF]"

	if [[ "X$DIFF" != "X" ]]; then
		echo "The response [tmp.out] does not match binary.test.response!"
		go_out 18
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Text buffer, async, echo"
###############################################################################
{
for i in {1..1000}
do
        # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
		-d "Hello from curl" http://localhost:8080/text/ok/async/echo 2>&1 )`

        RSP_EXPECTED="Hello from curl"
        echo "Response: [$RSP]"

        if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
                echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
                go_out 17
        fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Text buffer, async, no echo"
###############################################################################
{
for i in {1..1000}
do
	# Having a -i means to print the headers
	RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
		-d "Hello from curl" http://localhost:8080/text/ok/async 2>&1 )`

	RSP_EXPECTED="0: SUCCEED"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 16
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Text buffer, call fail"
###############################################################################
{
for i in {1..1000}
do
        # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
		-d "Hello from curl" http://localhost:8080/text/fail 2>&1 )`

	RSP_EXPECTED="11: 11:TPESVCFAIL (last error 11: Service returned 1)"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 16
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "Text buffer, call ok"
###############################################################################
{
for i in {1..1000}
do
	# Having a -i means to print the headers
	RSP=`(curl -s -H "Content-Type: text/plain" -X POST\
		-d "Hello from curl" http://localhost:8080/text/ok 2>&1 )`

	RSP_EXPECTED="Hello from EnduroX"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 15
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "JSON buffer, call async"
###############################################################################
{
for i in {1..1000}
do
        # Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{\"StringField\":\"Hello\",\
\"NumField\":12345,\
\"BoolField\":true}" \
http://localhost:8080/jsonbuf/ok/async 2>&1 )`

        RSP_EXPECTED="{\
\"error_code\":0\
,\"error_message\":\"SUCCEED\"\
}"
        echo "Response: [$RSP]"

        if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
                echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
                go_out 14
        fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "JSON buffer, call error"
###############################################################################
{
for i in {1..1000}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{\"StringField\":\"Hello\",\
\"NumField\":12345,\
\"BoolField\":true}" \
http://localhost:8080/jsonbuf/fail 2>&1 )`

        RSP_EXPECTED="{\
\"StringField\":\"Hello\"\
,\"NumField\":12345\
,\"BoolField\":true\
,\"error_code\":11\
,\"error_message\":\"11:TPESVCFAIL (last error 11: Service returned 1)\"\
}"
        echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 13
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "JSON buffer, no status"
###############################################################################
{
for i in {1..1000}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{\
\"StringField\":\"Hello\"\
,\"NumField\":12345\
,\"BoolField\":true\
}" \
http://localhost:8080/jsonbuf/ok/no/status 2>&1 )`

        RSP_EXPECTED="{\
\"StringField\":\"Hello\"\
,\"StringField2\":\"Hello\"\
,\"NumField\":12345\
,\"NumField2\":12345\
,\"BoolField\":true\
,\"BoolField2\":true\
}"
        echo "Response: [$RSP]"

        if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
                echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
                go_out 12
        fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "JSON buffer"
###############################################################################
{
for i in {1..1000}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -H "Content-Type: application/json" -X POST -d \
"{\"StringField\":\"Hello\",\
\"NumField\":12345,\
\"BoolField\":true}" \
http://localhost:8080/jsonbuf/ok 2>&1 )`

        RSP_EXPECTED="{\
\"StringField\":\"Hello\"\
,\"StringField2\":\"Hello\"\
,\"NumField\":12345\
,\"NumField2\":12345\
,\"BoolField\":true\
,\"BoolField2\":true\
,\"error_code\":0\
,\"error_message\":\"SUCCEED\"\
}"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 11
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Http error hanlding, fail case, timeout mapped to 404"
###############################################################################
{
for i in {1..1}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -i -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/httpe/tout/mapped 2>&1 )`


	RSP_EXPECTED="404"
	echo "Response: [$RSP]"

	if [[ "X$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 10
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "Http error hanlding, fail case, timeout 504"
###############################################################################
{
for i in {1..1}
do

	# Having a -i means to print the headers
        RSP=`(curl -s -i -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/httpe/tout 2>&1 )`


	RSP_EXPECTED="504"

	echo "Response: [$RSP]"

	if [[ "X$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 9
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Http error hanlding, fail case, 500"
###############################################################################
{
for i in {1..1000}
do

	# Having a -i means to print the headers
        RSP=`curl -s -i -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/httpe/fail 2>&1`


	RSP_EXPECTED="500"

	echo "Response: [$RSP]"

	if [[ "X$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 8
	fi
done
} >> $LOGFILE 2>&1

###############################################################################
echo "Http error hanlding, ok case"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -i -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/httpe/ok 2>&1`


	RSP_EXPECTED="200"

	echo "Response: [$RSP]"

	if [[ "X$RSP" != *"$RSP_EXPECTED"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 7
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "JSON2UBF errors handling"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/juerrors`


        RSP_EXPECTED="{\"EX_IF_ECODE\":0\
,\"T_SHORT_FLD\":123\
,\"T_SHORT_2_FLD\":123\
,\"T_LONG_FLD\":444444444\
,\"T_LONG_2_FLD\":444444444\
,\"T_CHAR_FLD\":\"A\"\
,\"T_CHAR_2_FLD\":\"A\"\
,\"T_FLOAT_FLD\":1.330000\
,\"T_FLOAT_2_FLD\":1.330000\
,\"T_DOUBLE_FLD\":4444.333300\
,\"T_DOUBLE_2_FLD\":4444.333300\
,\"EX_IF_EMSG\":\"SUCCEED\"\
,\"T_STRING_FLD\":\"HELLO\"\
,\"T_STRING_2_FLD\":\"HELLO\"\
,\"T_CARRAY_FLD\":\"SGVsbG8=\"\
,\"T_CARRAY_2_FLD\":\"SGVsbG8=\"\
}"

	echo "Response: [$RSP]"

	if [[ "X$RSP" != "X$RSP_EXPECTED" ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 6
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "First test, call some service with json stuff"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/svc1`


        RSP_EXPECTED="{\"T_SHORT_FLD\":123,\
\"T_SHORT_2_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_LONG_2_FLD\":444444444,\
\"T_CHAR_FLD\":\"A\",\
\"T_CHAR_2_FLD\":\"A\",\
\"T_FLOAT_FLD\":1.330000,\
\"T_FLOAT_2_FLD\":1.330000,\
\"T_DOUBLE_FLD\":4444.333300,\
\"T_DOUBLE_2_FLD\":4444.333300,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_STRING_2_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\",\
\"T_CARRAY_2_FLD\":\"SGVsbG8=\",\
\"error_code1\":0,\
\"error_message1\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 5
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Echo test"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/echo`


        RSP_EXPECTED="{\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_CHAR_FLD\":\"A\",\
\"T_FLOAT_FLD\":1.330000,\
\"T_DOUBLE_FLD\":4444.333300,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\",\
\"error_code\":0,\
\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 4
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Async test"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_CHAR_FLD\":\"A\",\
\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_FLOAT_FLD\":1.33,\
\"T_DOUBLE_FLD\":4444.3333,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\"}" \
http://localhost:8080/svc1/async`


        RSP_EXPECTED="{\"T_SHORT_FLD\":123,\
\"T_LONG_FLD\":444444444,\
\"T_CHAR_FLD\":\"A\",\
\"T_FLOAT_FLD\":1.330000,\
\"T_DOUBLE_FLD\":4444.333300,\
\"T_STRING_FLD\":\"HELLO\",\
\"T_CARRAY_FLD\":\"SGVsbG8=\",\
\"error_code\":0,\
\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 3
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Timeout test"
###############################################################################
{
for i in {1..1}
do

	RSP=`curl -H "Content-Type: application/json" -X POST -d "{\"T_CHAR_FLD\":\"A\"}" \
http://localhost:8080/longop/tout`

	#RSP_EXPECTED="{\"T_CHAR_FLD\":\"A\",\"error_code\":13,\"error_message\":\"13:TPETIME (last error 13: ndrx_mq_receive failed: Connection timed out)\"}"
	RSP_EXPECTED="TPETIME"

	echo "Response: [$RSP]"

	if [[ "X$RSP" != *"$RSP_EXPECTE"* ]]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 2
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Notime"
###############################################################################
{
for i in {1..1}
do
	RSP=`curl -H "Content-Type: application/json" -X POST -d "{\"T_CHAR_FLD\":\"A\"}" \
http://localhost:8080/longop/ok`

	RSP_EXPECTED="{\"T_CHAR_FLD\":\"A\",\"T_CHAR_2_FLD\":\"A\",\"error_code\":0,\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 1
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Empty regexp test"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_STRING_FLD\":\"REGEXP\"}" \
http://localhost:8080/regexp/empty`


        RSP_EXPECTED="{\"EX_IF_URL\":\"\/regexp\/empty\",\"T_STRING_FLD\":\"REGEXP\",\
\"error_code\":0,\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 4
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Valid UBF test with Regexp"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_STRING_FLD\":\"REGEXP\"}" \
http://localhost:8080/regexp/valid/ubf_test`


        RSP_EXPECTED="{\"EX_NETGATEWAY\":\"\/regexp\/valid\/ubf_test\",\
\"T_STRING_FLD\":\"REGEXP\",\"error_code\":0,\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 4
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Valid JSON test with Regexp"
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"string\":\"REGEXP\"}" \
http://localhost:8080/regexp/valid/json_test`


        RSP_EXPECTED="{\"Url\":\"/regexp/valid/json_test\",\"string\":\"REGEXP\",\
\"error_code\":0,\"error_message\":\"SUCCEED\"}"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 4
	fi
done
} >> $LOGFILE 2>&1
###############################################################################
echo "Invalid regexp test" 
###############################################################################
{
for i in {1..1000}
do

        RSP=`curl -H "Content-Type: application/json" -X POST -d \
"{\"T_STRING_FLD\":\"REGEXP\"}" \
http://localhost:8080/regexp/invalid/test`


        RSP_EXPECTED="404 page not found"

	echo "Response: [$RSP]"

	if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
		echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
		go_out 4
	fi
done
} >> $LOGFILE 2>&1

# go_out alreay doing stop
#xadmin stop -c -y


go_out 0


