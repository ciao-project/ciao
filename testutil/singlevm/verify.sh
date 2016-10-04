#!/bin/bash

ciao_bin="$HOME/local"
ciao_gobin="$GOPATH"/bin

# Read cluster env variables

. $ciao_bin/demo.sh

"$ciao_gobin"/ciao-cli workload list

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list workloads"
	cleanup
	exit 1
fi

"$ciao_gobin"/ciao-cli instance add --workload=e35ed972-c46c-4aad-a1e7-ef103ae079a2 --instances=2

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to launch VMs"
	cleanup
	exit 1
fi

"$ciao_gobin"/ciao-cli instance list

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list instances"
	cleanup
	exit 1
fi

"$ciao_gobin"/ciao-cli instance add --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=2

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to launch containers"
	cleanup
	exit 1
fi

sleep 5

"$ciao_gobin"/ciao-cli instance list
if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to list instances"
	cleanup
	exit 1
fi


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
	cleanup
fi

#Check docker networking
echo "Checking Docker Networking"
sleep 30
docker_id=$(sudo docker ps -q | head -1)
sudo docker logs "$docker_id"


#Now delete all instances
"$ciao_gobin"/ciao-cli instance delete --all

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to delete instances"
	exit 1
fi

"$ciao_gobin"/ciao-cli instance list
"$ciao_gobin"/ciao-cli instance delete --all
