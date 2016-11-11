#!/bin/bash
ciao_host=$(hostname)

sudo rm ciao-controller.db-shm ciao-controller.db-wal ciao-controller.db /tmp/ciao-controller-stats.db

sudo "$GOPATH"/bin/ciao-controller --cacert=./CAcert-"$ciao_host".pem --cert=./cert-Controller-"$ciao_host".pem --v 3 &
