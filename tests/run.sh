#!/bin/bash

#
# @(#) Integration tests
#

# have some limits... 
ulimit -n 50000
ulimit -c unlimited

# Load other config... stack size, etc
source ~/ndrx_home

> ./test.out
# Have some terminal output...
tail -f test.out &

(
M_tests=0
M_ok=0
M_fail=0

run_test () {

        test=$1
        M_tests=$((M_tests + 1))
        echo "*** RUNNING [$test]"

        pushd .
        cd $test
        ./run.sh
        ret=$?
        popd
        
        echo "*** RESULT [$test] $ret"
        
        if [[ $ret -eq 0 ]]; then
                M_ok=$((M_ok + 1))
        else
                M_fail=$((M_fail + 1))
        fi
}

run_test "01_restin"
run_test "02_tcpgatesv"
run_test "03_restout"

################################################################################
# Print results in out file
# use temp file to avoid plotting to the same file which are are grepping...
################################################################################
grep RESULT test.out >test.out.tmp
cat test.out.tmp
rm test.out.tmp
################################################################################ 

echo "*** SUMMARY $M_tests tests executed. $M_ok passes, $M_fail failures"

# wait for tail to pick the change
sleep 5
xadmin killall tail

exit $M_fail

) > test.out 2>&1

