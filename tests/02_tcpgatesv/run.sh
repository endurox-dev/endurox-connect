#!/bin/bash

#
# @(#) Test 01 - Rest-IN interface tests...
#

pushd .

rm runtime/log/* 2>/dev/null

cd runtime


NUMCALL=100
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

#
# So we need to add some demo server
# We need to add server process here + we need to register ubftab (test.fd)
#
xadmin provision -d -vaddubf=test.fd

cd conf
. settest1
# Fix some config (logging for testcl to file)
echo "[@debug]" >> app.ini
echo 'testcl= ndrx=5 ubf=1 tp=5 file=${NDRX_APPHOME}/log/testcl.log' >> app.ini

cd ..

# Start the system
xadmin start -y

# Let connections to establish
sleep 60

################################################################################
echo ">>> Running sequence tests"
################################################################################
NROFCALLS=40000
COMMAND="seq"
testcl $COMMAND $NROFCALLS TCP_P_SEQ_A
RET=$?

if [[ $RET != 0 ]]; then
    echo "testcl $COMMAND $NROFCALLS TCP_P_SEQ_A failed"
    go_out 53
fi

echo ">>> Running sequence tests (2)"

testcl $COMMAND $NROFCALLS TCP_P_SEQ_A
RET=$?

if [[ $RET != 0 ]]; then
    echo "testcl $COMMAND $NROFCALLS TCP_P_SEQ_A failed (2)"
    go_out 53
fi

echo "Sleep 90 - let async calls to complete..."
sleep 90

################################################################################
echo ">>> Run offset tests, len not included"
################################################################################
NROFCALLS=$(($NUMCALL+5))
NROFCALLS_CMP=$NUMCALL
COMMAND="offsetsync"
testcl $COMMAND $NROFCALLS TCP_P_SYNCOFF_A 0
RET=$?

if [[ $RET != 0 ]]; then
    echo "testcl $COMMAND $NROFCALLS TCP_P_SYNCOFF_A failed"
    go_out 51
fi

################################################################################
echo ">>> Run offset tests, swap bytes, len included"
################################################################################
NROFCALLS=$(($NUMCALL+5))
NROFCALLS_CMP=$NUMCALL
COMMAND="offsetsync"
testcl $COMMAND $NROFCALLS TCP_P_SYNCOFFI_A 1

RET=$?

if [[ $RET != 0 ]]; then
    echo "testcl $COMMAND $NROFCALLS TCP_P_SYNCOFFI_A failed"
    go_out 50
fi


################################################################################
echo ">>> Run async calls, sync invocation"
################################################################################
NROFCALLS=$(($NUMCALL+5))
NROFCALLS_CMP=$NUMCALL
COMMAND="corr"
testcl $COMMAND $NROFCALLS TCP_P_ASYSY_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_ASYSY_A failed"
	go_out 1
fi

################################################################################
echo ">>> Run async calls"
################################################################################
NROFCALLS=$(($NUMCALL+5))
NROFCALLS_CMP=$NUMCALL
COMMAND="async_call"

 > log/testsv.log
testcl async_call $NROFCALLS TCP_P_ASYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_ASYNC_A failed"
	go_out 1
fi

# Let connections to complete as we go async!
sleep 10

# Check that given count of reponses are generated...
CNT=`grep "Test case 11 OK" log/testsv.log | wc | awk '{print $1}'`

# We might have some buffered logs, thus have some more calls
# This give +5 for logs to flush
if [[ $CNT -lt  $NROFCALLS_CMP ]]; then
	echo "Expected $NROFCALLS but got $CNT from server traces!"
	go_out 2
fi

################################################################################
echo ">>> No connection"
################################################################################
NROFCALLS=$NUMCALL
COMMAND="nocon"
xadmin stop -i 210
xadmin stop -i 230

# Flush connections
sleep 1
testcl $COMMAND $NROFCALLS TCP_P_ASYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_ASYNC_A failed"
	go_out 3
fi


# Try also Persistent-sync channel, should be the same error.

xadmin start -i 210
xadmin start -i 230
sleep 1

################################################################################
echo " >>> Run Correlation..."
################################################################################
NROFCALLS=$NUMCALL
COMMAND="corr"

# Flush connections
# This time will start from Passive side...
testcl $COMMAND $NROFCALLS TCP_P_ASYNC_P
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_ASYNC_P failed"
	go_out 4
fi

################################################################################
echo ">>>Run Correlation, timeout"
################################################################################
COMMAND="corrtot"

# Flush connections
# This time will start from Passive side...
testcl $COMMAND TCP_P_ASYNC_P
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND TCP_P_ASYNC_P failed"
	go_out 5
fi

################################################################################
echo ">>> Persistent, Sync connection, call"
################################################################################
NROFCALLS=$NUMCALL
# We can reuse same test case, it will return some data (but tcpgates will match with
# connection id
COMMAND="corr"

testcl $COMMAND $NROFCALLS TCP_P_SYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_SYNC_A failed"
	go_out 6
fi

################################################################################
echo ">>> Persistent, Sync connection, call, timeout"
################################################################################
COMMAND="corrtot"

testcl $COMMAND TCP_P_SYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND TCP_P_SYNC_A failed"
	go_out 7
fi


################################################################################
echo ">>> Persistent, Sync connection, call, no-connection, try from Passive end."
################################################################################
NROFCALLS=$NUMCALL
COMMAND="nocon"

testcl $COMMAND $NROFCALLS TCP_P_SYNC_P
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_SYNC_P failed"
	go_out 9
fi

################################################################################
echo ">>> Nonpersistent, normal call"
################################################################################
NROFCALLS=$NUMCALL
# We can reuse same test case, it will return some data (but tcpgates will match with
# connection id
COMMAND="corr"

# Flush connections
# This time will start from Passive side...
testcl $COMMAND $NROFCALLS TCP_NP_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_NP_A failed"
	go_out 10
fi


################################################################################
echo ">>> Nonpersistent, timeout"
################################################################################
COMMAND="corrtot"

# Flush connections
# This time will start from Passive side...
testcl $COMMAND TCP_NP_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND TCP_NP_A failed"
	go_out 11
fi

################################################################################
echo ">>> Nonpersistent, cannot connect"
################################################################################
NROFCALLS=$NUMCALL
COMMAND="nocon"

testcl $COMMAND $NROFCALLS TCP_NP_P
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_NP_P failed"
	go_out 12
fi

################################################################################
echo ">>> Have some test where we overload the channel - i.e."
# Send multiple requests to desitnation host, the all messages must be cleared ok
# i.e. wait the connection from queue...
################################################################################
# Number of calls depends on internal modulus, now 40... over the ascii table from A
NROFCALLS=40
COMMAND="corrsim"

testcl $COMMAND $NROFCALLS TCP_P_ASYNC_P
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_ASYNC_P failed"
	go_out 13
fi

################################################################################
echo ">>> have some batch callers to non persistant connections they all should complete ok."
################################################################################
# Number of calls depends on internal modulus, now 40... over the ascii table from A
NROFCALLS=40
COMMAND="corrsim"

testcl $COMMAND $NROFCALLS TCP_NP_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_NP_A failed"
	go_out 14
fi

################################################################################
echo ">>> echo same test for sync mode"
################################################################################
# Number of calls depends on internal modulus, now 40... over the ascii table from A
NROFCALLS=40
COMMAND="corrsim"

#testcl $COMMAND $NROFCALLS TCP_P_SYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl $COMMAND $NROFCALLS TCP_P_SYNC_A failed"
	go_out 15
fi


################################################################################
echo ">>> Test connection reset tests in_idle_max/in_idle_check params..."
################################################################################
# This assumes that test have already run for 20 seconds to have idle time enough

sleep 30

FILE="tcpgatesv-async-active-idlerst"
RESET_OUT=`grep RESET $NDRX_APPHOME/log/$FILE*`

echo "RESET $FILE out: [$RESET_OUT]"

if [ "X$RESET_OUT" == "X" ]; then
	echo "Testing of in_idle_max/in_idle_check fail - NO 'RESET' found in $FILE"
	go_out 16
fi


FILE="tcpgatesv-async-active-idlerst"
RESET_OUT=`grep RESET $NDRX_APPHOME/log/$FILE*`

echo "RESET $FILE out: [$RESET_OUT]"

if [ "X$RESET_OUT" == "X" ]; then
	echo "Testing of in_idle_max/in_idle_check fail - NO 'RESET' found in $FILE"
	go_out 17
fi

FILE="tcpgatesv-async-passive."
RESET_OUT=`grep RESET $NDRX_APPHOME/log/$FILE*`

echo "RESET $FILE out: [$RESET_OUT]"

if [ "X$RESET_OUT" != "X" ]; then
	echo "Testing of in_idle_max/in_idle_check fail - 'RESET' MUST NOT be found in $FILE"
	go_out 18
fi

FILE="tcpgatesv-async-active."
RESET_OUT=`grep RESET $NDRX_APPHOME/log/$FILE*`

echo "RESET $FILE out: [$RESET_OUT]"

if [ "X$RESET_OUT" != "X" ]; then
	echo "Testing of in_idle_max/in_idle_check fail - 'RESET' MUST NOT be found in $FILE"
	go_out 19
fi

################################################################################
echo ">>> Test for any log errors"
################################################################################

# Catch is there is test error!!!
if [ "X`grep TESTERROR log/*.log`" != "X" ]; then
        echo "Test error detected!"
        go_out 20
fi

xadmin stop -c -y

go_out 0

