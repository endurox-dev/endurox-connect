#!/bin/bash

#
# @(#) Test 03 - Rest-OUT interface tests...
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

###############################################################################
echo "JSON2UBF test case - juerrors"
###############################################################################

COMMAND="juerrors"

testcl $COMMAND 
RET=$?

if [[ $RET != 0 ]]; then
        echo "testcl $COMMAND: failed"
        go_out 1
fi


###############################################################################
echo "Done"
###############################################################################

xadmin stop -c -y


go_out 0


