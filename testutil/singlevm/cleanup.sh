#!/bin/bash

. ~/local/demo.sh

ciao_gobin="$GOPATH"/bin
ciao_data="$HOME/local/data"
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
sudo rm -rf "$ciao_data"/*
sudo rm /var/lib/ciao/images/4e16e743-265a-4bf2-9fd1-57ada0b28904
sudo rm /var/lib/ciao/images/df3768da-31f5-4ba6-82f0-127a1a705169
sudo rm /var/lib/ciao/images/73a86d7e-93c0-480e-9c41-ab42f69b7799
sudo docker network rm $(sudo docker network ls --filter driver=ciao -q)
