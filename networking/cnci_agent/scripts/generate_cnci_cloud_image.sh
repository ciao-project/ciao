#!/bin/bash

#Defaults
image="clear-7370-ciao-networking.img"
certs_dir=$GOPATH/src/github.com/01org/ciao/networking/cnci_agent/scripts/certs
cnci_agent=$GOPATH/bin/cnci_agent
cnci_sysd=$GOPATH/src/github.com/01org/ciao/networking/cnci_agent/scripts/cnci-agent.service
partition="2"

#The image can be downloaded from 
#wget https://download.clearlinux.org/demos/ciao/$cnci_xz

usage="$(basename "$0") [--image clear_cnci_image_name] [-certs certificate_directory] [-agent cnci_agent_binary] [-script cnci_systemd_script] \n\n A simple script to create a CNCI Image from a clear cloud image. \n Defaults for any unspecified option are as follows \n\n --agent $cnci_agent \n --certs $certs_dir \n --image $image \n --script $cnci_sysd\n\n"

while :
do
    case "$1" in
      -a | --agent)
	  cnci_agent="$2" 
	  shift 2
	  ;;
      -c | --certs)
	  certs_dir="$2" 
	  shift 2
	  ;;
      -h | --help)
	  echo -e $usage >&2
	  exit 0
	  ;;
      -i | --image)
	  image="$2" 
	  shift 2
	  ;;
      -s | --script)
	  cnci_sysd="$2" 
	  shift 2
	  ;;
      *)
	  break
	  ;;
    esac
done

echo -e "\nMounting image: $image"
sudo mkdir -p /mnt/tmp
sudo modprobe nbd max_part=63
sudo qemu-nbd -c /dev/nbd0 "$image"
sudo mount /dev/nbd0p$partition /mnt/tmp

echo -e "Cleaning up any artifacts"
sudo rm -rf /mnt/tmp/var/lib/ciao

echo -e "Copying agent image"
sudo cp "$cnci_agent" /mnt/tmp/usr/sbin/

echo -e "Copying agent systemd service script"
sudo cp "$cnci_sysd" /mnt/tmp/usr/lib/systemd/system/

echo -e "Installing the service"
sudo mkdir -p /mnt/tmp/etc/systemd/system/default.target.wants
sudo rm /mnt/tmp/etc/systemd/system/default.target.wants/cnci-agent.service
sudo chroot /mnt/tmp /bin/bash -c "sudo ln -s /usr/lib/systemd/system/cnci-agent.service /etc/systemd/system/default.target.wants/"

echo -e "Copying CA certificates"
sudo mkdir -p /mnt/tmp/var/lib/ciao/
sudo cp "$certs_dir"/CAcert-* /mnt/tmp/var/lib/ciao/CAcert-server-localhost.pem

echo -e "Copying CNCI Agent certificate"
sudo cp "$certs_dir"/cert-CNCIAgent-* /mnt/tmp/var/lib/ciao/cert-client-localhost.pem

echo -e "Removing cloud-init traces"
sudo rm -rf /mnt/tmp/var/lib/cloud

#Umount
echo -e "Done! unmounting\n"
sudo umount /mnt/tmp
sudo qemu-nbd -d /dev/nbd0
exit 0
