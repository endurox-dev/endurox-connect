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
        -vusv1_log='${NDRX_APPHOME}/log/restin.log'

cd conf

. settest1

xadmin start -y

# Let restin to start
sleep 2

RET=0
###############################################################################
# First test, call some service with json stuff
###############################################################################
for i in {1..100}
do
        RSP=`curl -H "Content-Type: application/json" -X POST -d '{"T_CHAR_FLD":"A"}' http://localhost:8080/svc1`
        RSP_EXPECTED='{"T_CHAR_FLD":"A","T_CHAR_2_FLD":"A", "error_code": 0, "error_message": "SUCCEED"}'
        echo "Response: [$RSP]"

        if [ "X$RSP" != "X$RSP_EXPECTED" ]; then
                echo "Invalid response received, got: [$RSP], expected: [$RSP_EXPECTED]"
                RET=1
        fi
done
###############################################################################

xadmin stop -c -y

popd

exit $RET

