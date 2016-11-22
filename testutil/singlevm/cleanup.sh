#!/bin/bash

. ~/local/demo.sh

ciao_gobin="$GOPATH"/bin
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sleep 2
sudo "$ciao_gobin"/ciao-launcher --alsologtostderr -v 3 --hard-reset
sudo ip link del ciao_eth
sudo pkill -F /tmp/dnsmasq.ciaovlan.pid
sudo docker rm -v -f ceph-demo
sudo rm /etc/ceph/*
sudo rm -rf /var/lib/ciao/ciao-image
sudo docker network rm $(sudo docker network ls --filter driver=ciao -q)
