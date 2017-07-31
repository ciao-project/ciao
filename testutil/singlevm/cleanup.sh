#!/bin/bash

. ~/local/demo.sh

ciao_gobin="$GOPATH"/bin
ciao_host=$(hostname)
ext_int=$(ip -o route get 8.8.8.8 | cut -d ' ' -f 5)

ciao_dir=/var/lib/ciao
ciao_pki_path=/etc/pki/ciao

sudo systemctl stop ciao-scheduler.service
sudo systemctl stop ciao-launcher.service
sudo systemctl stop ciao-controller.service
sudo "$ciao_gobin"/ciao-launcher --alsologtostderr -v 3 --hard-reset
sudo rm -rf ${ciao_dir}
sudo rm -rf ${ciao_pki_path}
sudo rm -rf /etc/ciao
sudo rm -f /etc/systemd/system/ciao*.service
sudo systemctl daemon-reload
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
sudo rm -f /etc/ceph/*
sudo rm -rf /var/lib/ciao/ciao-image
sudo docker network rm $(sudo docker network ls --filter driver=ciao -q)
sudo rm -r ~/local/mysql/
sudo rm -f /var/lib/ciao/networking/docker_plugin.db
