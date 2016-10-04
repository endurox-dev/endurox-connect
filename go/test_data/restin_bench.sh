#!/bin/bash

ab  -n 100000 -c20 -p restin.data http://localhost:8080/svc2/hello

