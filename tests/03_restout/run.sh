#!/bin/bash

#
# @(#) Test 03 - Rest-OUT interface tests...
#

TIMES=200

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
        -vaddubf=test.fd \
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

# Let restout to start
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

################################################################################
echo "JSON2UBF test case - jue (json2ubf err), OK"
################################################################################

COMMAND="ubfcall"

testcl $COMMAND JUBFJUE_OK $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 1
fi

################################################################################
echo "JSON2UBF test case - juerrors, failure"
################################################################################

COMMAND="ubfcall"

testcl $COMMAND JUBFJUE_FAIL 1
RET=$?

if [[ $RET != 11 ]]; then
	echo "testcl $COMMAND: failed (ret must be 11, but got: $RET)"
	go_out 2
fi

################################################################################
echo "JSON2UBF test case - juerrors, timeout"
################################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFJUE_TOUT 1
RET=$?

if [[ $RET != 13 ]]; then
	echo "testcl $COMMAND: failed (ret must be 13, but got: $RET)"
	go_out 3
fi

################################################################################
echo "JSON2UBF test case - juerrors, no entry"
################################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFJUE_NENT 1
RET=$?

if [[ $RET != 6 ]]; then
	echo "testcl $COMMAND: failed (ret must be 6, but got: $RET)"
	go_out 4
fi


################################################################################
echo "JSON2UBF, HTTP errors, OK"
################################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFHTE_OK $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 5
fi

###############################################################################
echo "JSON2UBF, HTTP errors, failure"
###############################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFJUE_FAIL 1
RET=$?

if [[ $RET != 11 ]]; then
	echo "testcl $COMMAND: failed (ret must be 11, but got: $RET)"
	go_out 6
fi

###############################################################################
echo "JSON2UBF, HTTP errors, timeout"
###############################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFHTE_TOUT 1
RET=$?

if [[ $RET != 13 ]]; then
	echo "testcl $COMMAND: failed (ret must be 13, but got: $RET)"
	go_out 7
fi

###############################################################################
echo "JSON2UBF, HTTP errors, NENT"
###############################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFHTE_NENT 1
RET=$?

if [[ $RET != 6 ]]; then
	echo "testcl $COMMAND: failed (ret must be 6, but got: $RET)"
	go_out 8
fi

###############################################################################
echo "JSON2UBF, HTTP errors, NENT, custom error mapping"
###############################################################################
COMMAND="ubfcall"

testcl $COMMAND JUBFHTE_NENT_13 1
RET=$?

if [[ $RET != 13 ]]; then
	echo "testcl $COMMAND: failed (ret must be 13, but got: $RET) - custom mapping fail"
	go_out 9
fi

###############################################################################
echo "STRING test case - TEXT errors, OK"
###############################################################################
COMMAND="stringcall"

testcl $COMMAND TEXTTE_OK $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 10
fi

###############################################################################
echo "STRING test case - TEXT error, failure"
###############################################################################
COMMAND="stringcall"

testcl $COMMAND TEXTTE_FAIL 1
RET=$?

if [[ $RET != 11 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 11
fi

###############################################################################
echo "JSON test case - JSON errors, OK"
###############################################################################
COMMAND="jsoncall"

testcl $COMMAND JSONJE_OK $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 13
fi


###############################################################################
echo "JSON test case - JSON error, no status in OK rsp"
###############################################################################
COMMAND="jsoncall"

testcl $COMMAND JSONJE_OKNS $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 14
fi

###############################################################################
echo "JSON test case - JSON error, no status in OK rsp"
###############################################################################
COMMAND="jsoncall"

testcl $COMMAND JSONJE_OKASYNC $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 15
fi

###############################################################################
echo "RAW test case - TEXT errors, OK"
###############################################################################
COMMAND="carraycall"

testcl $COMMAND RAWTE_OK $TIMES
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 16
fi

###############################################################################
echo "RAW test case - TEXT error, failure"
###############################################################################
COMMAND="carraycall"

testcl $COMMAND RAWTE_FAIL $TIMES
RET=$?

if [[ $RET != 11 ]]; then
	echo "testcl $COMMAND: failed"
	go_out 17
fi

###############################################################################
echo "JSON2UBF echo OK"
###############################################################################

###############################################################################
echo "JSON echo OK"
###############################################################################

###############################################################################
echo "RAW echo OK"
###############################################################################

###############################################################################
echo "ECHO FAIL, no SVC"
###############################################################################

###############################################################################
echo "Done"
###############################################################################

xadmin stop -c -y


go_out 0


