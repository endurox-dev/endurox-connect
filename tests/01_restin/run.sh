#!/bin/bash

#
# @(#) Test 01 - Rest-IN interface tests...
#

pushd .

rm runtime/log/* 2>/dev/null

cd runtime

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

. settest1

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

# Test the buffer:
# curl -s -H "Content-Type: text/plain" -X POST --data-binary "@binary.test" http://localhost:8080/binary/ok 

###############################################################################
echo "Text buffer, async, echo"
###############################################################################
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

###############################################################################
echo "Text buffer, async, no echo"
###############################################################################
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


###############################################################################
echo "Text buffer, call fail"
###############################################################################
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


###############################################################################
echo "Text buffer, call ok"
###############################################################################
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

###############################################################################
echo "JSON buffer, call async"
###############################################################################
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
###############################################################################
echo "JSON buffer, call error"
###############################################################################
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

###############################################################################
echo "JSON buffer, no status"
###############################################################################
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

###############################################################################
echo "JSON buffer"
###############################################################################
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

###############################################################################
echo "Http error hanlding, fail case, timeout mapped to 404"
###############################################################################
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


###############################################################################
echo "Http error hanlding, fail case, timeout 504"
###############################################################################
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

###############################################################################
echo "Http error hanlding, fail case, 500"
###############################################################################
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


###############################################################################
echo "Http error hanlding, ok case"
###############################################################################
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

###############################################################################
echo "JSON2UBF errors handling"
###############################################################################
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
###############################################################################
echo "First test, call some service with json stuff"
###############################################################################
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
###############################################################################
echo "Echo test"
###############################################################################
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

###############################################################################
echo "Async test"
###############################################################################
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

###############################################################################
echo "Timeout test"
###############################################################################
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

###############################################################################
echo "Notime"
###############################################################################
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


xadmin stop -c -y


go_out 0


