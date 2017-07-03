#!/bin/bash

ciao_scripts="$GOPATH"/src/github.com/01org/ciao/testutil/singlevm

source $ciao_scripts/config.sh

export no_proxy=$no_proxy,$ciao_vlan_ip,$ciao_host

#Create a directory where all the certificates, binaries and other
#dependencies are placed
mkdir -p "$ciao_bin"

if [ ! -d  "$ciao_bin" ]
then
	echo "FATAL ERROR: Unable to create $ciao_bin"
	exit 1
fi

# Variables for ciao binaries
export CIAO_DEMO_PATH="$ciao_bin"
export CIAO_CONTROLLER="$ciao_host"
export CIAO_USERNAME="$ciao_demo_username"
export CIAO_PASSWORD="$ciao_demo_password"
export CIAO_ADMIN_USERNAME="$ciao_admin_username"
export CIAO_ADMIN_PASSWORD="$ciao_admin_password"
export CIAO_IDENTITY="$ciao_identity_url"
export CIAO_SSH_KEY="$workload_sshkey"

# Save these vars for later use, too
> "$ciao_env" # Clean out previous data
set | grep ^CIAO_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

# Variables for keystone
export OS_USER_DOMAIN_NAME=default
export OS_IMAGE_API_VERSION=2
export OS_PROJECT_NAME=admin
export OS_IDENTITY_API_VERSION=3
export OS_PASSWORD=${ciao_admin_password}
export OS_AUTH_URL=https://"$ciao_host":$keystone_admin_port/v3
export OS_USERNAME=${ciao_admin_username}
export OS_KEY=
export OS_CACERT="$keystone_cert"
export OS_PROJECT_DOMAIN_NAME=default

# Save these vars for later use, too
set | grep ^OS_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

echo "Subnet =" $ciao_vlan_subnet

echo "Cleaning up"
"$ciao_bin"/cleanup.sh 2> /dev/null 1>/dev/null

# Copy the cleanup scripts
cp "$ciao_scripts"/cleanup.sh "$ciao_bin"

cleanup()
{
    echo "Performing cleanup"
    "$ciao_bin"/cleanup.sh
}

# Ctrl-C Trapper
trap ctrl_c INT

function ctrl_c() {
    echo "Trapped CTRL-C, performing cleanup"
    cleanup
    exit 1
}

usage="$(basename "$0") [-d --download] The script will download dependencies if needed. Specifying --download will force download the dependencies even if they are cached locally
$(basename "$0") [-a --all-images] By default only the Ubuntu cloud image is downloaded.  Specify this option to download and create additional images and workloads"

while :
do
    case "$1" in
      -c | --cnciurl)
          ciao_cnci_url="$2"
	  shift 2
	  ;;
      -d | --download)
          download=1
          shift 1
          ;;
      -a | --all-images)
          all_images=1
          shift 1
          ;;
      -i | --cnciimage)
          ciao_cnci_image="$2"
	  shift 2
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

echo "Generating workload ssh key $workload_sshkey"
rm -f "$workload_sshkey" "$workload_sshkey".pub
ssh-keygen -f "$workload_sshkey" -t rsa -N '' > /dev/null
test_sshkey=$(< "$workload_sshkey".pub)
chmod 600 "$workload_sshkey".pub
#Note: Password is set to ciao
test_passwd='$6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO.'

sudo mkdir -p ${ciao_pki_path}
if [ ! -d ${ciao_pki_path} ]
then
	echo "FATAL ERROR: Unable to create ${ciao_pki_path}"
	exit 1
fi

sudo mkdir -p /etc/ciao/
if [ ! -d /etc/ciao ]
then
	echo "FATAL ERROR: Unable to create /etc/ciao"
	exit 1
fi

sudo mkdir -p ${ciao_ctl_dir}/workloads
if [ ! -d ${ciao_ctl_dir}/workloads ]
then
	echo "FATAL ERROR: Unable to create ${ciao_ctl_dir}/workloads}"
	exit 1
fi

#Build ciao
rm -f "$ciao_gobin"/ciao*
cd "$ciao_src"
go install -v --tags 'debug' ./...

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to build ciao"
	exit 1
fi

#Check if the build was a success
if [ ! -f "$ciao_gobin"/ciao-cli ]
then
	echo "FATAL ERROR: build failed"
	exit 1
fi

# Started early to minimise overall running time
$ciao_scripts/setup_keystone.sh

killall ciao-deploy-server
"$ciao_gobin"/ciao-deploy-server -admin-password="$test_passwd" \
    -admin-ssh-key="$test_sshkey" -ceph-id="ciao" \
    -compute-net="$ciao_vlan_subnet" -https-ca-cert="$keystone_cert" \
    -https-cert="$keystone_key" -keystone-password="$ciao_password" \
    -keystone-url="https://$ciao_host:$keystone_admin_port" \
    -keystone-user="$ciao_username" -mgmt-net="$ciao_vlan_subnet" \
    -scheduler -server-ip="$ciao_vlan_ip" &

source $ciao_scripts/setup_webui.sh

# Set macvlan interface
if [ -x "$(command -v ip)" ]; then
    sudo ip link del "$ciao_bridge"
    sudo ip link add name "$ciao_bridge" type bridge
    sudo iptables -A FORWARD -p all -i "$ciao_bridge" -j ACCEPT
    sudo ip link add link "$ciao_bridge" name ciaovlan type macvlan mode bridge
    sudo ip addr add "$ciao_vlan_subnet" brd "$ciao_vlan_brdcast" dev ciaovlan
    sudo ip link set dev ciaovlan up
    sudo ip -d link show ciaovlan
    sudo ip link set dev "$ciao_bridge" up
    sudo ip -d link show "$ciao_bridge"
    sudo iptables -A FORWARD -p all -i ciaovlan -j ACCEPT
    #Do this only in the case of ciao-down as it can potentially
    #open up the machine. On bare metal the user will need to explicitly
    #add this rule
    if [ "$ciao_host" == "singlevm" ]; then
	sudo iptables -A FORWARD -p all -i "$ext_int" -j ACCEPT
	#NAT out all the traffic departing ciao-down
	sudo iptables -t nat -A POSTROUTING -o "$ext_int" -j MASQUERADE
    fi

else
    echo 'ip command is not supported'
fi

# Set DHCP server with dnsmasq
sudo mkdir -p /var/lib/misc
if [ -x "$(command -v ip)" ]; then
    sudo dnsmasq -C $ciao_scripts/dnsmasq.conf.ciaovlan \
	 --pid-file=/tmp/dnsmasq.ciaovlan.pid
else
    echo 'dnsmasq command is not supported'
fi

# Install ceph
# This runs *after* keystone so keystone will get port 5000 first
sudo docker run --name ceph-demo -d --net=host -v /etc/ceph:/etc/ceph -e MON_IP=$ciao_vlan_ip -e CEPH_PUBLIC_NETWORK=$ciao_vlan_subnet ceph/demo
sudo ceph auth get-or-create client.ciao -o /etc/ceph/ceph.client.ciao.keyring mon 'allow *' osd 'allow *' mds 'allow'

source "$ciao_scripts"/wait_for_keystone.sh

cp "$ciao_scripts"/verify.sh "$ciao_bin"

"$GOPATH"/bin/ciao-deploy-client -deploy-server="http://$ciao_host:9000" -force \
    -role controller
"$GOPATH"/bin/ciao-deploy-client -deploy-server="http://$ciao_host:9000" -force \
    -role dual   

# become admin in order to upload images and setup workloads
export CIAO_USERNAME=$CIAO_ADMIN_USERNAME
export CIAO_PASSWORD=$CIAO_ADMIN_PASSWORD

$ciao_scripts/setup_images.sh
$ciao_scripts/setup_workloads.sh

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
