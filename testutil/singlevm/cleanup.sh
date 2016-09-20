#!/bin/bash

if [ -z "$HOSTS_FILE_BACKUP" ]; then
    echo "HOSTS_FILE_BACKUP" is not set.
    echo "Please run"
    echo ""
    echo ". ~/local/demo.sh"
    exit 1
fi
ciao_gobin="$GOPATH"/bin
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sleep 2
sudo "$ciao_gobin"/ciao-launcher --hard-reset
sudo ip link del eth10
sudo pkill -F /tmp/dnsmasq.macvlan0.pid
sudo mv $HOSTS_FILE_BACKUP /etc/hosts
