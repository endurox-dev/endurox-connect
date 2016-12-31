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

        if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
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
\"error_code\":0,\
\"error_message\":\"SUCCEED\"}"

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


