#!/bin/bash

ciao_bin="$HOME/local"
ciao_gobin="$GOPATH"/bin

# Read cluster env variables

. $ciao_bin/demo.sh

"$ciao_gobin"/ciao-cli workload list

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list workloads"
	exit 1
fi

"$ciao_gobin"/ciao-cli instance add --workload=e35ed972-c46c-4aad-a1e7-ef103ae079a2 --instances=2

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to launch VMs"
	exit 1
fi

"$ciao_gobin"/ciao-cli instance list

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list instances"
	exit 1
fi

#Launch containers

#Pre-cache the image to reduce the start latency
sudo docker pull debian
"$ciao_gobin"/ciao-cli instance add --workload=ca957444-fa46-11e5-94f9-38607786d9ec --instances=1

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to launch containers"
	exit 1
fi

sleep 5

"$ciao_gobin"/ciao-cli instance list
if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list instances"
	exit 1
fi

container_1=`sudo docker ps -q -l`
container_1_ip=`sudo docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $container_1`

"$ciao_gobin"/ciao-cli instance add --workload=ca957444-fa46-11e5-94f9-38607786d9ec --instances=1

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to launch containers"
	exit 1
fi

sleep 5

"$ciao_gobin"/ciao-cli instance list
if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list instances"
	exit 1
fi

container_2=`sudo docker ps -q -l`

#Check SSH connectivity
"$ciao_gobin"/ciao-cli instance list

#The VM takes time to boot as you are running on two
#layers of virtualization. Hence wait a bit
retry=0
until [ $retry -ge 6 ]
do
	ssh_ip=$("$ciao_gobin"/ciao-cli instance list --workload=e35ed972-c46c-4aad-a1e7-ef103ae079a2 --detail |  grep "SSH IP:" | sed 's/^.*SSH IP: //' | head -1)

	if [ "$ssh_ip" == "" ] 
	then
		echo "Waiting for instance to boot"
		let retry=retry+1
		sleep 30
		continue
	fi

	ssh_check=$(head -1 < /dev/tcp/"$ssh_ip"/33002)
	echo "$ssh_check"

	echo "Attempting to ssh to: $ssh_ip"

	if [[ "$ssh_check" == *SSH-2.0-OpenSSH* ]]
	then
		echo "SSH connectivity verified"
		break
	else
		let retry=retry+1
		echo "Retrying ssh connection $retry"
	fi
	sleep 30
done

if [ $retry -ge 6 ]
then
	echo "Unable check ssh connectivity into VM"
	exit 1
fi

#Check docker networking
echo "Checking Docker Networking"
sudo docker exec $container_2 /bin/ping -c 3 $container_1_ip

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to ping across containers"
	exit 1
else
	echo "Container connectivity verified"
fi

#Clear out all prior events
"$ciao_gobin"/ciao-cli event delete

#Wait for the event count to drop to 0
retry=0
ciao_events=0

until [ $retry -ge 6 ]
do
	ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

	if [ $ciao_events -eq 0 ]
	then
		break
	fi

	let retry=retry+1
	sleep 1
done

if [ $ciao_events -ne 0 ]
then
	echo "FATAL ERROR: ciao events not deleted properly"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

#Test External IP Assignment support
#Pick the first instance which is a VM, as we can even SSH into it
#We have already checked that the VM is up.
"$ciao_gobin"/ciao-cli pool create -name test && ciao-cli pool add -subnet 203.0.113.0/24 -name test

testinstance=`"$ciao_gobin"/ciao-cli instance list -f '{{with index . 0}}{{.ID}}{{end}}'`

"$ciao_gobin"/ciao-cli external-ip map -instance $testinstance -pool test

#Wait for the CNCI to report successful map
retry=0
ciao_events=0

until [ $retry -ge 6 ]
do
	ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

	if [ $ciao_events -eq 1 ]
	then
		break
	fi

	let retry=retry+1
	sleep 1
done

if [ $ciao_events -ne 1 ]
then
	echo "FATAL ERROR: ciao external IP event not reported"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

mapped=`ciao-cli event list -f '{{if eq (len .) 1}}{{(index . 0).Message}}{{end}}' | cut -d " " -f 1`
if [ $mapped != "Mapped" ]
then
	echo "FATAL ERROR: Unknown event"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

"$ciao_gobin"/ciao-cli event list
"$ciao_gobin"/ciao-cli external-ip list

#We checked the event, so the mapping should exist
testip=`"$ciao_gobin"/ciao-cli external-ip list -f '{{with index . 0}}{{.ExternalIP}}{{end}}'`

sudo ip route add 203.0.113.0/24 dev ciaovlan
ping -c 10 $testip
ping_result=$?
sudo ip route del 203.0.113.0/24 dev ciaovlan

if [ $ping_result -ne 0 ]
then
	echo "FATAL ERROR: Unable to ping external IP"
	exit 1
else
	echo "Container external connectivity verified"
fi


"$ciao_gobin"/ciao-cli external-ip unmap -address $testip

#Wait for the CNCI to report successful unmap
retry=0
ciao_events=0

until [ $retry -ge 6 ]
do
	ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

	if [ $ciao_events -eq 2 ]
	then
		break
	fi

	let retry=retry+1
	sleep 1
done

if [ $ciao_events -ne 2 ]
then
	echo "FATAL ERROR: ciao external IP event not reported"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

unmapped=`"$ciao_gobin"/ciao-cli event list -f '{{if eq (len .) 2}}{{(index . 1).Message}}{{end}}' | cut -d " " -f 1`
if [ $unmapped != "Unmapped" ]
then
	echo "FATAL ERROR: Unknown event"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

"$ciao_gobin"/ciao-cli event list
"$ciao_gobin"/ciao-cli external-ip list


#Now delete all instances
"$ciao_gobin"/ciao-cli instance delete --all

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to delete instances"
	exit 1
fi

"$ciao_gobin"/ciao-cli instance list

#Wait for all the instance deletions to be reported back
retry=0
ciao_events=0

until [ $retry -ge 6 ]
do
	ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

	if [ $ciao_events -eq 6 ]
	then
		break
	fi

	let retry=retry+1
	sleep 1
done

if [ $ciao_events -ne 6 ]
then
	echo "FATAL ERROR: ciao instances not deleted properly"
	"$ciao_gobin"/ciao-cli event list
	exit 1
fi

#Wait around a bit as instance delete is asynchronous
retry=0
ciao_networks=0
until [ $retry -ge 6 ]
do
	#Verify that there are no ciao related artifacts left behind
	ciao_networks=`sudo docker network ls --filter driver=ciao -q | wc -l`

	if [ $ciao_networks -eq 0 ]
	then
		break
	fi
	let retry=retry+1
	sleep 1
done

if [ $ciao_networks -ne 0 ]
then
	echo "FATAL ERROR: ciao docker networks not cleaned up"
	sudo docker network ls --filter driver=ciao
	exit 1
fi


#The only ciao interfaces left behind should be CNCI VNICs
#Once we can delete tenants we should not even have them around
cnci_vnics=`ip -d link | grep alias | grep cnci | wc -l`
ciao_vnics=`ip -d link | grep alias | wc -l`

if [ $cnci_vnics -ne $ciao_vnics ]
then
	echo "FATAL ERROR: ciao network interfaces not cleaned up"
	ip -d link | grep alias
	exit 1
fi
