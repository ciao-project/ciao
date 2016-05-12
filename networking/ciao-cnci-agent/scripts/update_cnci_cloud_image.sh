#!/bin/bash

#This script will be deprecated and removed in the future

#Note: the clear image should have dnsmasq and iptables installed
#swupd bundle-add sysadmin-hostmgmt/kvm-host network-advanced
#You can take any clear image that supports GRE tunneling (6000 and beyond)

#TODO: Add code here to clear out any cloud init that was carried out

if [ -z "$1" ]; then
        IMAGE="clear-7360-ciao-networking.img"
else
        IMAGE="$1"
fi

if [ -z "$2" ]; then
	CERTS_DIR=$GOPATH/src/github.com/01org/ciao/networking/cnci_agent/scripts/certs
else
	CERTS_DIR=$2
fi

if [ -z "$3" ]; then
	CNCI_AGENT=$GOPATH/bin/cnci_agent
else
	CNCI_AGENT=$3
fi

if [ -z "$4" ]; then
	CNCI_SYSD=$GOPATH/src/github.com/01org/ciao/networking/cnci_agent/scripts/cnci-agent.service
else
	CNCI_SYSD=$4
fi

if [ -z "$5" ]; then
	PARTITION="2"
else
	PARITION=$5
fi



echo "mounting image" 
echo $IMAGE
sudo mkdir -p /mnt/tmp
sudo modprobe nbd max_part=63
sudo qemu-nbd -c /dev/nbd0 $IMAGE
sudo mount /dev/nbd0p$PARTITION /mnt/tmp

echo "Cleanup artifacts"
sudo ls -alp /mnt/tmp/var/lib/ciao
sudo rm -rf /mnt/tmp/var/lib/ciao
#echo "Checking cleanup"
#sudo ls -alp /mnt/tmp/var/lib/ciao

#Copy the cnci_agent image
echo "copying agent image"
sudo cp $CNCI_AGENT /mnt/tmp/usr/sbin/

sudo ls -alp /mnt/tmp/usr/sbin/cnci_agent
sudo ls -alp $CNCI_AGENT
sudo diff $CNCI_AGENT /mnt/tmp/usr/sbin/cnci_agent

#Copy the cnci_agent systemd service script
echo "copying agent systemd service script"
sudo cp $CNCI_SYSD /mnt/tmp/usr/lib/systemd/system/

sudo ls -alp /mnt/tmp/usr/lib/systemd/system/cnci-agent.service
sudo ls -alp $CNCI_SYSD
sudo diff $CNCI_SYSD /mnt/tmp/usr/lib/systemd/system/cnci-agent.service

#Install the systemd service
#Hacking it. Ideally do it with chroot
echo "installing the service"
sudo mkdir -p /mnt/tmp/etc/systemd/system/default.target.wants
sudo rm /mnt/tmp/etc/systemd/system/default.target.wants/cnci-agent.service
sudo chroot /mnt/tmp /bin/bash -c "sudo ln -s /usr/lib/systemd/system/cnci-agent.service /etc/systemd/system/default.target.wants/"
sudo ls -alp /mnt/tmp/etc/systemd/system/default.target.wants

#Copy the certs
echo "Copying the certs"
sudo mkdir -p /mnt/tmp/var/lib/ciao/

echo "Copying CA certificates..."
sudo cp $CERTS_DIR/CAcert-* /mnt/tmp/var/lib/ciao/CAcert-server-localhost.pem

echo -e "Copying CNCI Agent certificate..."
sudo cp $CERTS_DIR/cert-CNCIAgent-* /mnt/tmp/var/lib/ciao/cert-client-localhost.pem

ls -alp /mnt/tmp/var/lib/ciao/

#Remove cloud-init traces (hack)
#echo "Checking cleanup"
#sudo ls -alp /mnt/tmp/var/lib/cloud
sudo rm -rf /mnt/tmp/var/lib/cloud
#sudo ls -alp /mnt/tmp/var/lib/cloud

#Umount
echo "done unmounting"
sudo umount /mnt/tmp
sudo qemu-nbd -d /dev/nbd0
