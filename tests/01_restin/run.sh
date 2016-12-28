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

#
# First test, call some service with json stuff
#
curl -H "Content-Type: application/json" -X POST -d '{"T_CHAR_FLD":"A"}' http://localhost:8080/svc1

xadmin stop -c -y

popd

return 0
