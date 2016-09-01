#!/bin/bash
ciao_host=$(hostname)
ciao_ip=$(ip route get 8.8.8.8 | head -1 | cut -d' ' -f8)
ciao_bin="$HOME/local"
ciao_cert="$ciao_bin""/cert-Scheduler-""$ciao_host"".pem"
export no_proxy=$no_proxy,$ciao_host

ciao_email="ciao-devel@lists.clearlinux.org"
ciao_org="Intel"
ciao_src="$GOPATH"/src/github.com/01org/ciao
ciao_gobin="$GOPATH"/bin
ciao_scripts="$GOPATH"/src/github.com/01org/ciao/testutil/singlevm
ciao_env="$ciao_bin/demo.sh"
ciao_ctl_log="/var/lib/ciao/logs/controller/ciao-controller.ERROR"
ciao_cnci_image="clear-8260-ciao-networking.img"
download=0
hosts_file_backup="/etc/hosts.orig.$RANDOM"

cleanup()
{
	echo "Performing cleanup"
	"$ciao_gobin"/ciao-cli instance delete --all
	#Also kill the CNCI (as there is no other way to delete it today)
	sudo killall qemu-system-x86_64
	sudo rm -rf /var/lib/ciao/instances
	sudo ip link del eth10
	sudo pkill -F /tmp/dnsmasq.macvlan0.pid
	sudo mv $hosts_file_backup /etc/hosts
}

usage="$(basename "$0") [--download] The script will download dependencies if needed. Specifing --download will force download the dependencies even if they are cached locally"

while :
do
    case "$1" in
      -d | --download)
          download=1
          shift 1
          ;;
      -h | --help)
          echo -e "$usage" >&2
          exit 0
          ;;
      *)
          break
          ;;
    esac
done

set -o nounset

sudo mkdir -p /var/lib/ciao/images
if [ ! -d /var/lib/ciao/images ]
then
	echo "FATAL ERROR: Unable to create /var/lib/ciao/images"
	exit 1

fi

sudo mkdir -p /etc/pki/ciao
if [ ! -d /etc/pki/ciao ]
then
	echo "FATAL ERROR: Unable to create /etc/pki/ciao"
	exit 1
fi

sudo mkdir -p /etc/ciao/
if [ ! -d /etc/ciao ]
then
	echo "FATAL ERROR: Unable to create /etc/ciao"
	exit 1
fi
sudo cp -f "$ciao_scripts"/configuration.yaml /etc/ciao

#Stop any running agents and CNCIs
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sudo killall qemu-system-x86_64
echo "Original /etc/hosts is temporarily move to $hosts_file_backup"
sudo mv /etc/hosts $hosts_file_backup
echo "$ciao_ip $ciao_host" > hosts
sudo mv hosts /etc/hosts
sudo rm -rf /var/lib/ciao/instances
echo "Deleting docker containers. This may take time"
sudo docker rm -f $(sudo docker ps -a -q)

#Create a directory where all the certificates, binaries and other
#dependencies are placed
mkdir "$ciao_bin"

if [ ! -d  "$ciao_bin" ]
then
	echo "FATAL ERROR: Unable to create $ciao_bin"
	exit 1
fi

cd "$ciao_bin"

#Cleanup any old artifcats
rm -f "$ciao_bin"/*.pem
sudo rm -f "$ciao_bin"/ciao-controller.db-shm
sudo rm -f "$ciao_bin"/ciao-controller.db-wal
sudo rm -f "$ciao_bin"/ciao-controller.db
sudo rm -f /tmp/ciao-controller-stats.db
rm -rf "$ciao_bin"/tables
rm -rf "$ciao_bin"/workloads

#Build ciao
rm -f "$ciao_gobin"/ciao*
cd "$ciao_src"
go install -v --tags 'debug' ./...

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to build ciao"
	exit 1
fi

cd "$ciao_bin"

#Check if the build was a sucess
if [ ! -f "$ciao_gobin"/ciao-cli ]
then
	echo "FATAL ERROR: build failed"
	exit 1
fi

#Generate Certificates
"$GOPATH"/bin/ciao-cert -server -role scheduler -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -verify 

"$GOPATH"/bin/ciao-cert -role cnciagent -server-cert "$ciao_cert" -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -verify 

"$GOPATH"/bin/ciao-cert -role controller -server-cert "$ciao_cert" -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -verify 

"$GOPATH"/bin/ciao-cert -role agent,netagent -server-cert "$ciao_cert" -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -verify

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout controller_key.pem -out controller_cert.pem -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=ciao.example.com"

#Copy the certs
sudo cp -f controller_key.pem /etc/pki/ciao
sudo cp -f controller_cert.pem /etc/pki/ciao


#Copy the configuration
cd "$ciao_bin"
cp -a "$ciao_src"/ciao-controller/tables "$ciao_bin"
cp -a "$ciao_src"/ciao-controller/workloads "$ciao_bin"

#Over ride the configuration with test specific defaults
cp -f "$ciao_scripts"/workloads/* "$ciao_bin"/workloads
cp -f "$ciao_scripts"/tables/* "$ciao_bin"/tables


#Copy the launch scripts
cp "$ciao_scripts"/run_scheduler.sh "$ciao_bin"
cp "$ciao_scripts"/run_controller.sh "$ciao_bin"
cp "$ciao_scripts"/run_launcher.sh "$ciao_bin"

#Download the firmware
cd "$ciao_bin"
if [ $download -eq 1 ] || [ ! -f OVMF.fd ]
then
	rm -f OVMF.fd
	curl -O https://download.clearlinux.org/image/OVMF.fd
fi

if [ ! -f OVMF.fd ]
then
	echo "FATAL ERROR: unable to download firmware"
	exit 1
fi

sudo cp -f OVMF.fd  /usr/share/qemu/OVMF.fd

#Generate the CNCI VM and seed the image and populate the image cache
cd "$ciao_bin"
rm -f "$ciao_cnci_image".qcow

if [ $download -eq 1 ] || [ ! -f "$ciao_cnci_image" ] 
then
	rm -f "$ciao_cnci_image"
	"$GOPATH"/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/generate_cnci_cloud_image.sh -c "$ciao_bin" -i "$ciao_cnci_image" -d
else
	"$GOPATH"/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/generate_cnci_cloud_image.sh -c "$ciao_bin" -i "$ciao_cnci_image"
fi

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to mount CNCI Image"
	exit 1
fi

if [ ! -f "$ciao_cnci_image" ]
then
	echo "FATAL ERROR: unable to download CNCI Image"
	exit 1
fi

qemu-img convert -f raw -O qcow2 "$ciao_cnci_image" "$ciao_cnci_image".qcow
sudo cp -f "$ciao_cnci_image".qcow /var/lib/ciao/images
cd /var/lib/ciao/images
sudo ln -sf "$ciao_cnci_image".qcow 4e16e743-265a-4bf2-9fd1-57ada0b28904

#Clear
cd "$ciao_bin"
LATEST=$(curl https://download.clearlinux.org/latest)

if [ $download -eq 1 ] || [ ! -f clear-"${LATEST}"-cloud.img ] 
then
	rm -f clear-"${LATEST}"-cloud.img.xz
	rm -f clear-"${LATEST}"-cloud.img
	curl -O https://download.clearlinux.org/releases/"$LATEST"/clear/clear-"$LATEST"-cloud.img.xz
	xz -T0 --decompress clear-"${LATEST}"-cloud.img.xz
fi


if [ ! -f clear-"${LATEST}"-cloud.img ]
then
	echo "FATAL ERROR: unable to download clear cloud Image"
	exit 1
fi

sudo cp -f clear-"${LATEST}"-cloud.img /var/lib/ciao/images
cd /var/lib/ciao/images
sudo ln -sf clear-"${LATEST}"-cloud.img df3768da-31f5-4ba6-82f0-127a1a705169

# Set macvlan interface
if [ -x "$(command -v ip)" ]; then
    sudo ip link del eth10
    sudo ip link add name eth10 type bridge
    sudo ip link add link eth10 name macvlan0 type macvlan mode bridge
    sudo ip addr add 198.51.100.1/24 brd 198.51.100.255 dev macvlan0
    sudo ip link set dev macvlan0 up
    sudo ip -d link show macvlan0
    sudo ip link set dev eth10 up
    sudo ip -d link show eth10
else
    echo 'ip command is not supported'
fi

# Set DHCP server with dnsmasq
sudo mkdir -p /var/lib/misc
if [ -x "$(command -v ip)" ]; then
    sudo dnsmasq -C $ciao_scripts/dnsmasq.conf.macvlan0 \
	 --pid-file=/tmp/dnsmasq.macvlan0.pid
else
    echo 'dnsmasq command is not supported'
fi

#Kick off the agents
cd "$ciao_bin"
"$ciao_bin"/run_scheduler.sh  &> /dev/null
"$ciao_bin"/run_launcher.sh &> /dev/null
"$ciao_bin"/run_controller.sh &> /dev/null

echo "export CIAO_CONTROLLER=""$ciao_host" > "$ciao_env"
echo "export CIAO_USERNAME=admin" >> "$ciao_env"
echo "export CIAO_PASSWORD=giveciaoatry" >> "$ciao_env"
sleep 5
cat "$ciao_ctl_log"
identity=$(grep CIAO_IDENTITY $ciao_ctl_log | sed 's/^.*export/export/')
echo "$identity" >> "$ciao_env"
export CIAO_CONTROLLER="$ciao_host"
export CIAO_USERNAME=admin
export CIAO_PASSWORD=giveciaoatry

eval "$identity"
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
cleanup
