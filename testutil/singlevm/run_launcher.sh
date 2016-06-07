#!/bin/bash

ciao_host=$(hostname)

#Obtain the subnet of the primary interface
default_if=$(ip route list | awk '/^default/ {print $5}')
default_subnet=$(ip -o -f inet addr show $default_if | awk '{print $4}')

#Cleanup artifacts
sudo "$GOPATH"/bin/ciao-launcher --cacert=./CAcert-"$ciao_host".pem --cert=./cert-CNAgent-NetworkingAgent-"$ciao_host".pem --network=dual --compute-net "$default_subnet" --mgmt-net "$default_subnet" --alsologtostderr -v 3 --disk-limit=false --hard-reset

#Cleanup any prior docker instances and networks
sudo docker rm $(sudo docker ps -qa)
sudo docker network rm $(sudo docker network ls -q -f "type=custom")
sudo rm -f /var/lib/ciao/networking/docker_plugin.db

#Run launcher
sudo "$GOPATH"/bin/ciao-launcher --cacert=./CAcert-"$ciao_host".pem --cert=./cert-CNAgent-NetworkingAgent-"$ciao_host".pem --network=dual --compute-net "$default_subnet" --mgmt-net "$default_subnet" -v 3 --disk-limit=false &
