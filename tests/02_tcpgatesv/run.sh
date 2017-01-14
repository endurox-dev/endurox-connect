#!/bin/bash

#
# @(#) Test 01 - Rest-IN interface tests...
#

pushd .

rm runtime/log/* 2>/dev/null

cd runtime

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
xadmin provision -d -vaddubf=test.fd,Exfields

cd conf
. settest1
cd ..

# Start the system
xadmin start -y

# Let connections to establish
sleep 2

################################################################################
# Run async calls
################################################################################
NROFCALLS=5000
testcl async_call $NROFCALLS TCP_P_ASYNC_A
RET=$?

if [[ $RET != 0 ]]; then
	echo "testcl async_call $NROFCALLS TCP_P_ASYNC_A failed"
	go_out 1
fi

# Let connections to complete as we go async!
sleep 2

# Check that given count of reponses are generated...
CNT=`grep "Test case 11 OK" log/testsv.log | wc | awk '{print $1}'`

if [[ $CNT !=  $NROFCALLS ]]; then
	echo "Expected $NROFCALLS but got $CNT from server traces!"
	go_out 2
fi


xadmin stop -c -y


go_out 0


