#!/bin/bash

ciao_host=$(hostname)

sudo -E "$GOPATH"/bin/ciao-scheduler --cacert=./CAcert-"$ciao_host".pem --cert=./cert-Scheduler-"$ciao_host".pem -v 3 &
