#!/bin/bash
ciao_host=$(hostname)
ciao_ip=$(ip route get 8.8.8.8 | head -1 | cut -d' ' -f8)
ciao_subnet=$(echo $ciao_ip | sed -e 's/\([0-9]\+\).\([0-9]\+\).\([0-9]\+\).\([0-9]\+\)/\1.\2\.\3.0\/24/')
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
fedora_cloud_image="Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
fedora_cloud_url="https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
download=0
hosts_file_backup="/etc/hosts.orig.$RANDOM"

#Ensure that the local cache exists
mkdir -p "$ciao_bin"

# Copy the cleanup scripts
cp "$ciao_scripts"/cleanup.sh "$ciao_bin"

cleanup()
{
    echo "Performing cleanup"
    "$ciao_bin"/cleanup.sh $hosts_file_backup
}

# Ctrl-C Trapper
trap ctrl_c INT

function ctrl_c() {
    echo "Trapped CTRL-C, performing cleanup"
    cleanup
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

openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout controller_key.pem -out controller_cert.pem -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=$ciao_host"

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
cp "$ciao_scripts"/verify.sh "$ciao_bin"

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

#Fedora, needed for BAT tests
cd "$ciao_bin"
if [ $download -eq 1 ] || [ ! -f $fedora_cloud_image ]
then
    rm -f $fedora_cloud_image
    curl -L -O $fedora_cloud_url
fi

if [ ! -f $fedora_cloud_image ]
then
	echo "FATAL ERROR: unable to download fedora cloud Image"
	exit 1
fi

sudo cp -f $fedora_cloud_image /var/lib/ciao/images
cd /var/lib/ciao/images
sudo ln -sf $fedora_cloud_image 73a86d7e-93c0-480e-9c41-ab42f69b7799

# Install ceph

sudo docker run --name ceph-demo -d --net=host -v /etc/ceph:/etc/ceph -e MON_IP=$ciao_ip -e CEPH_PUBLIC_NETWORK=$ciao_subnet ceph/demo
sudo ceph auth get-or-create client.ciao -o /etc/ceph/ceph.client.ciao.keyring mon 'allow *' osd 'allow *' mds 'allow'


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

echo "export HOSTS_FILE_BACKUP=""$hosts_file_backup" > "$ciao_env"
echo "export CIAO_CONTROLLER=""$ciao_host" >> "$ciao_env"
echo "export CIAO_USERNAME=admin" >> "$ciao_env"
echo "export CIAO_PASSWORD=giveciaoatry" >> "$ciao_env"
echo "export CIAO_ADMIN_USERNAME=admin" >> "$ciao_env"
echo "export CIAO_ADMIN_PASSWORD=giveciaoatry" >> "$ciao_env"
echo "export CIAO_CA_CERT_FILE=/etc/pki/ciao/controller_cert.pem" >> "$ciao_env"
sleep 5
identity=$(grep CIAO_IDENTITY $ciao_ctl_log | sed 's/^.*export/export/')
echo "$identity" >> "$ciao_env"

echo "---------------------------------------------------------------------------------------"
echo ""
echo "Your ciao development environment has been initialised."
echo "To get started run:"
echo ""
echo ". ~/local/demo.sh"
echo ""
echo "Verify the cluster is working correctly by running"
echo ""
echo "~/local/verify.sh"
echo ""
echo "Use ciao-cli to manipulate and inspect the cluster, e.g., "
echo ""
echo "ciao-cli instance add --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=1"
echo ""
echo "When you're finished run the following command to cleanup"
echo ""
echo "~/local/cleanup.sh"
