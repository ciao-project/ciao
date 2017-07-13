#!/bin/bash

. ~/local/demo.sh

ciao_gobin="$GOPATH"/bin
ciao_host=$(hostname)
ext_int=$(ip -o route get 8.8.8.8 | cut -d ' ' -f 5)
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sleep 2
sudo "$ciao_gobin"/ciao-launcher --alsologtostderr -v 3 --hard-reset
sudo iptables -D FORWARD -i ciao_br -j ACCEPT
sudo iptables -D FORWARD -i ciaovlan -j ACCEPT
if [ "$ciao_host" == "singlevm" ]; then
	sudo iptables -D FORWARD -i "$ext_int" -j ACCEPT
	sudo iptables -t nat -D POSTROUTING -o "$ext_int" -j MASQUERADE
fi
sudo ip link del ciao_br
sudo pkill -F /tmp/dnsmasq.ciaovlan.pid
sudo docker rm -v -f keystone
sudo docker rm -v -f ceph-demo
sudo rm /etc/ceph/*
sudo rm -rf /var/lib/ciao/ciao-image
sudo docker network rm $(sudo docker network ls --filter driver=ciao -q)
sudo rm -r ~/local/mysql/
sudo rm -f /var/lib/ciao/networking/docker_plugin.db
