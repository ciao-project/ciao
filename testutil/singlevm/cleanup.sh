#!/bin/bash

if [ ! -z $1 ]; then
    hosts_file_backup=$1
else
    . ~/local/demo.sh
    hosts_file_backup=$HOSTS_FILE_BACKUP
fi

ciao_gobin="$GOPATH"/bin
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sleep 2
sudo "$ciao_gobin"/ciao-launcher --alsologtostderr -v 3 --hard-reset
sudo ip link del eth10
sudo pkill -F /tmp/dnsmasq.macvlan0.pid
sudo mv $hosts_file_backup /etc/hosts
sudo docker rm -v -f ceph-demo
sudo rm /etc/ceph/*
