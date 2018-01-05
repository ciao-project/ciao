#!/bin/bash
ciao_host=$(hostname)
ciao_ip=$(hostname -i)
ext_int=$(ip -o route get 8.8.8.8 | cut -d ' ' -f 5)
ciao_bridge=ciao_br
ciao_vlan_ip=198.51.100.1
ciao_vlan_subnet=${ciao_vlan_ip}/24
ciao_vlan_brdcast=198.51.100.255
ciao_bin="$HOME/local"
https_key="$ciao_bin"/https_key.pem
https_cert="$ciao_bin"/https_cert.pem
workload_sshkey="$ciao_bin"/testkey
ciao_pki_path=/etc/pki/ciao
client_auth_ca_path="$ciao_pki_path/auth-CA.pem"
export no_proxy=$no_proxy,$ciao_vlan_ip,$ciao_host

GOPATH=`go env GOPATH`
ciao_src="$GOPATH"/src/github.com/ciao-project/ciao
ciao_gobin="$GOPATH"/bin
ciao_scripts="$GOPATH"/src/github.com/ciao-project/ciao/testutil/singlevm
ciao_env="$ciao_bin/demo.sh"
ciao_dir=/var/lib/ciao
all_images=0


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
export CIAO_SSH_KEY="$workload_sshkey"
export CIAO_CLIENT_CERT_FILE="$ciao_bin/auth-testuser.pem"
export CIAO_ADMIN_CLIENT_CERT_FILE="$ciao_pki_path/auth-admin.pem"
export CIAO_CA_CERT_FILE="$https_cert"

# Save these vars for later use, too
> "$ciao_env" # Clean out previous data
set | grep ^CIAO_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

echo "Subnet =" $ciao_vlan_subnet

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

usage="$(basename "$0") [-a --all-images] By default only the Ubuntu cloud image is downloaded.  Specify this option to download and create additional images and workloads"

while :
do
    case "$1" in
      -a | --all-images)
          all_images=1
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

echo "Generating workload ssh key $workload_sshkey"
rm -f "$workload_sshkey" "$workload_sshkey".pub
ssh-keygen -f "$workload_sshkey" -t rsa -N ''
chmod 600 "$workload_sshkey".pub
#Note: Password is set to ciao
test_passwd='$6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO.'

cleanup

cd "$ciao_bin"

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

#Check if the build was a success
if [ ! -f "$ciao_gobin"/ciao ]
then
	echo "FATAL ERROR: build failed"
	exit 1
fi

#Generate Certificates

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$https_key" -out "$https_cert" -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=$ciao_host"

#Copy the certs
sudo install -d "$ciao_pki_path"
sudo install -m 0644 -t "$ciao_pki_path" "$https_cert"

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
    #Do this only in the case of ccloudvm as it can potentially
    #open up the machine. On bare metal the user will need to explicitly
    #add this rule
    if [ "$ciao_host" == "singlevm" ]; then
	sudo iptables -A FORWARD -p all -i "$ext_int" -j ACCEPT
	#NAT out all the traffic departing ccloudvm
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
sudo docker run --name ceph-demo -d --net=host -v /etc/ceph:/etc/ceph -e MON_IP=$ciao_vlan_ip -e CEPH_PUBLIC_NETWORK=$ciao_vlan_subnet ceph/demo
sudo ceph auth get-or-create client.ciao -o /etc/ceph/ceph.client.ciao.keyring mon 'allow *' osd 'allow *' mds 'allow'

ciao-deploy setup \
--force \
--https-cert="$https_key" \
--https-ca-cert="$https_cert" \
--admin-ssh-key="$workload_sshkey".pub \
--local-launcher \
--mgmt-net="$ciao_vlan_subnet" --compute-net="$ciao_vlan_subnet" \
--server-ip="$ciao_vlan_ip" \
--ceph-id="ciao" \
--image-cache-directory="$ciao_bin" \
--cnci="tiny"
ciao-deploy auth create testuser

# Make configuration.yaml world readable otherwise storage tests will be skipped.
sudo chmod a+r /etc/ciao/configuration.yaml

curl -O https://download.clearlinux.org/image/OVMF.fd
sudo cp -f OVMF.fd  /usr/share/qemu/OVMF.fd

if [ $all_images -eq 1 ]; then
ciao-deploy create-bat-workloads --all-workloads --password="$test_passwd" --ssh-public-key-file="$workload_sshkey".pub --image-cache-directory="$ciao_bin"
else
ciao-deploy create-bat-workloads --password="$test_passwd" --ssh-public-key-file="$workload_sshkey".pub --image-cache-directory="$ciao_bin"
fi

echo "---------------------------------------------------------------------------------------"
echo ""
echo "Your ciao development environment has been initialised."
echo "To get started run:"
echo ""
echo ". ~/local/demo.sh"
echo ""
echo "Verify the cluster is working correctly by running"
echo ""
echo "$GOPATH/src/github.com/ciao-project/ciao/testutil/singlevm/verify.sh"
echo ""
echo "Use ciao to manipulate and inspect the cluster, e.g., "
echo ""
echo "ciao create instance ab68111c-03a6-11e6-87de-001320fb6e31"
echo ""
echo "When you're finished run the following command to cleanup"
echo ""
echo "~/local/cleanup.sh"
