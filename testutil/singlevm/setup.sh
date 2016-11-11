#!/bin/bash
ciao_host=$(hostname)
ciao_interface=ciao_eth
ciao_ip=198.51.100.1
ciao_subnet=${ciao_ip}/24
ciao_brdcast=198.51.100.255
ciao_bin="$HOME/local"
ciao_cert="$ciao_bin""/cert-Scheduler-""$ciao_host"".pem"
ciao_controller_key=controller_key.pem
ciao_controller_cert=controller_cert.pem
ciao_image_key=ciao-image-key.pem
ciao_image_cert=ciao-image-cacert.pem
ciao_pki_path=/etc/pki/ciao
export no_proxy=$no_proxy,$ciao_ip,$ciao_host

ciao_email="ciao-devel@lists.clearlinux.org"
ciao_org="Intel"
ciao_src="$GOPATH"/src/github.com/01org/ciao
ciao_gobin="$GOPATH"/bin
ciao_scripts="$GOPATH"/src/github.com/01org/ciao/testutil/singlevm
ciao_env="$ciao_bin/demo.sh"
ciao_ctl_log="/var/lib/ciao/logs/controller/ciao-controller.ERROR"
ciao_cnci_image="clear-8260-ciao-networking.img"
ciao_cnci_url="https://download.clearlinux.org/demos/ciao"
fedora_cloud_image="Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
fedora_cloud_url="https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
download=0
conf_file="$ciao_bin"/configuration.yaml
ciao_username=csr
ciao_password=hello
ciao_admin_username=admin
ciao_admin_password=giveciaoatry
ciao_demo_username=demo
ciao_demo_password=hello
keystone_int_port=5000
keystone_ext_port=35357
mysql_data_dir="${ciao_bin}"/mysql
ciao_identity_url="https://""$ciao_host"":""$keystone_int_port"

# Variables for ciao binaries
export CIAO_CONTROLLER="$ciao_host"
export CIAO_USERNAME="$ciao_username"
export CIAO_PASSWORD="$ciao_password"
export CIAO_ADMIN_USERNAME="$ciao_admin_username"
export CIAO_ADMIN_PASSWORD="$ciao_admin_password"
export CIAO_CA_CERT_FILE="$ciao_pki_path"/"$ciao_controller_cert"
export CIAO_IDENTITY="$ciao_identity_url"

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
export OS_AUTH_URL=https://"$ciao_host":$keystone_ext_port/v3
export OS_USERNAME=${ciao_admin_username}
export OS_KEY=
export OS_CACERT=${ciao_pki_path}/${ciao_image_cert}
export OS_PROJECT_DOMAIN_NAME=default

# Save these vars for later use, too
set | grep ^OS_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

echo "Subnet =" $ciao_subnet

#Create a directory where all the certificates, binaries and other
#dependencies are placed
mkdir -p "$ciao_bin"

if [ ! -d  "$ciao_bin" ]
then
	echo "FATAL ERROR: Unable to create $ciao_bin"
	exit 1
fi

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
}

usage="$(basename "$0") [--download] The script will download dependencies if needed. Specifying --download will force download the dependencies even if they are cached locally"

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

echo "Generating configuration file $conf_file"
(
cat <<-EOF
configure:
  scheduler:
    storage_uri: /etc/ciao/configuration.yaml
  storage:
    ceph_id: ciao
  controller:
    compute_ca: ${ciao_pki_path}/${ciao_controller_cert}
    compute_cert: ${ciao_pki_path}/${ciao_controller_key}
    identity_user: ${ciao_username}
    identity_password: ${ciao_password}
  image_service:
    type: glance
    url: https://${ciao_host}
  launcher:
    compute_net: [${ciao_subnet}]
    mgmt_net: [${ciao_subnet}]
    disk_limit: false
    mem_limit: false
  identity_service:
    type: keystone
    url: https://${ciao_host}:${keystone_ext_port}
EOF
) > $conf_file

sudo mkdir -p /var/lib/ciao/images
if [ ! -d /var/lib/ciao/images ]
then
	echo "FATAL ERROR: Unable to create /var/lib/ciao/images"
	exit 1

fi

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
sudo install -m 0644 $conf_file /etc/ciao

#Stop any running agents and CNCIs
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sudo killall qemu-system-x86_64
sudo rm -rf /var/lib/ciao/instances

cd "$ciao_bin"

#Cleanup any old artifacts
rm -f "$ciao_bin"/*.pem
sudo rm -f "$ciao_bin"/ciao-controller.db-shm
sudo rm -f "$ciao_bin"/ciao-controller.db-wal
sudo rm -f "$ciao_bin"/ciao-controller.db
sudo rm -f /tmp/ciao-controller-stats.db
sudo rm -f "$ciao_bin"/ciao-image.db
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

#Check if the build was a success
if [ ! -f "$ciao_gobin"/ciao-cli ]
then
	echo "FATAL ERROR: build failed"
	exit 1
fi

#Generate Certificates
"$GOPATH"/bin/ciao-cert -anchor -role scheduler -email="$ciao_email" \
    -organization="$ciao_org" -host="$ciao_host" -ip="$ciao_ip" -verify

"$GOPATH"/bin/ciao-cert -role cnciagent -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -ip="$ciao_ip" -verify

"$GOPATH"/bin/ciao-cert -role controller -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -ip="$ciao_ip" -verify

"$GOPATH"/bin/ciao-cert -role agent,netagent -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" -ip="$ciao_ip" -verify

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout ${ciao_controller_key} -out ${ciao_controller_cert} -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=$ciao_host"

#Copy the certs
sudo install -m 0644 ${ciao_controller_key} ${ciao_pki_path}
sudo install -m 0644 ${ciao_controller_cert} ${ciao_pki_path}
sudo ln -s ${ciao_controller_key} ${ciao_pki_path}/${ciao_image_key}
sudo ln -s ${ciao_controller_cert} ${ciao_pki_path}/${ciao_image_cert}
#Update system's trusted certificates
CACERT_PROG=$(type -p update-ca-certificates)
CACERT_DIR=/usr/local/share/ca-certificates
if [ x"${CACERT_PROG}" != x ] && [ -x "${CACERT_PROG}" ] && \
    [ -d "${CACERT_DIR}" ]; then
    sudo install -m 0644 ${ciao_controller_cert} "${CACERT_DIR}"/ciao.crt
    sudo "${CACERT_PROG}" --fresh
fi

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
	"$GOPATH"/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/generate_cnci_cloud_image.sh -c "$ciao_bin" -i "$ciao_cnci_image" -d -u "$ciao_cnci_url"
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

# Set macvlan interface
if [ -x "$(command -v ip)" ]; then
    sudo ip link del "$ciao_interface"
    sudo ip link add name "$ciao_interface" type bridge
    sudo ip link add link "$ciao_interface" name ciaovlan type macvlan mode bridge
    sudo ip addr add "$ciao_subnet" brd "$ciao_brdcast" dev ciaovlan
    sudo ip link set dev ciaovlan up
    sudo ip -d link show ciaovlan
    sudo ip link set dev "$ciao_interface" up
    sudo ip -d link show "$ciao_interface"
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

# Helper functions for openstack commands
function os_exists() {
    os_cmd="$1"
    os_val="$2"
    os_col="${3:-Name}"

    openstack "$os_cmd" list -f value -c "$os_col" | grep -q -w "$os_val"
    return $?
}

function os_exists_ep() {
    # Special case
    os_name="$1"
    os_service="$2"
    os_region="$3"
    os_interface="$4"

    openstack endpoint list --service "$os_service" --region "$os_region" \
        --interface "$os_interface" -f value -c 'Service Type' | \
        grep -q -w "$os_name"
    return $?
}

function os_create_ep() {
    os_name="$1"
    os_service="$2"
    os_region="$3"
    os_interface="$4"
    os_url="$5"

    if ! os_exists_ep "$os_name" "$os_service" "$os_region" "$os_interface";
    then
        echo "Attempting to add \"$os_interface\" endpoint"
        openstack endpoint create --region "$os_region" "$os_name" \
            "$os_interface" "$os_url"
    fi
}

function os_create_project() {
    project="$1"

    if ! os_exists project "$project"; then
        echo "Attempting to create \"$project\" project"
        openstack project create "$project"
    fi
}

function os_create_service() {
    svc_name="$1"
    svc_type="$2"
    svc_desc="${3:-}"

    if ! os_exists service "$svc_name"; then
        echo "Attempting to create \"$svc_name\" $svc_type service"
        if [ x"$svc_desc" != x ]; then
            openstack service create --name "$svc_name" \
                --description "$svc_desc" "$svc_type"
        else
            openstack service create --name "$svc_name" "$svc_type"
        fi
    fi
}

function os_create_user() {
    user="$1"
    pass="$2"

    if ! os_exists user "$user"; then
        echo "Attempting to create \"$user\" user"
        openstack user create --password "$pass" "$user"
    fi
}

function os_create_role() {
    role="$1"

    if ! os_exists role "$role"; then
        echo "Attempting to create \"$role\" role"
        openstack role create "$role"
    fi
}

function os_add_role() {
    role="$1"
    user="$2"
    project="$3"

    # Duplicate is OK
    openstack role add --project "$project" --user "$user" "$role"
}


## Install keystone
sudo docker run -d -it --name keystone \
    -p $keystone_int_port:5000 \
    -p $keystone_ext_port:35357 \
    -e IDENTITY_HOST="$ciao_host" -e KEYSTONE_ADMIN_PASSWORD="${OS_PASSWORD}" \
    -v $mysql_data_dir:/var/lib/mysql \
    -v ${OS_CACERT}:/etc/nginx/ssl/keystone_cert.pem \
    -v ${ciao_pki_path}/${ciao_image_key}:/etc/nginx/ssl/keystone_key.pem clearlinux/keystone

echo -n "Waiting for keystone identity service to become available"
try_until=$(($(date +%s) + 30))
while : ; do
    while [ $(date +%s) -le $try_until ]; do
        if openstack service list > /dev/null 2>&1; then
            echo READY
            break 2
        else
            echo -n .
            sleep 1
        fi
    done
    echo FAILED
    break
done
# Create services/projects/roles/users if needed
os_create_service ciao compute
os_create_user "$ciao_username" "$ciao_password"
os_create_project service
os_create_role admin
os_add_role admin "$ciao_username" service
os_create_user "$ciao_demo_username" "$ciao_demo_password"
os_create_project demo
os_create_role user
os_add_role user "$ciao_demo_username" demo
os_create_service glance image "Image Service"
os_create_ep image glance RegionOne public   https://$ciao_host:9292
os_create_ep image glance RegionOne internal https://$ciao_host:9292
os_create_ep image glance RegionOne admin    https://$ciao_host:9292

# Install ceph
# This runs *after* keystone so keystone will get port 5000 first
sudo docker run --name ceph-demo -d --net=host -v /etc/ceph:/etc/ceph -e MON_IP=$ciao_ip -e CEPH_PUBLIC_NETWORK=$ciao_subnet ceph/demo
sudo ceph auth get-or-create client.ciao -o /etc/ceph/ceph.client.ciao.keyring mon 'allow *' osd 'allow *' mds 'allow'

#Kick off the agents
cd "$ciao_bin"
"$ciao_bin"/run_scheduler.sh  &> /dev/null
"$ciao_bin"/run_launcher.sh &> /dev/null
"$ciao_bin"/run_controller.sh &> /dev/null

sleep 5

# Run the image service

"$ciao_bin"/run_image.sh "$ciao_identity_url" &> /dev/null

sleep 1

. $ciao_env

echo ""
echo "Uploading test images to image service"
echo "---------------------------------------------------------------------------------------"
if [ -f "$ciao_cnci_image".qcow ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file "$ciao_cnci_image".qcow \
        --name "ciao CNCI image" --id 4e16e743-265a-4bf2-9fd1-57ada0b28904
fi

if [ -f clear-"${LATEST}"-cloud.img ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file clear-"${LATEST}"-cloud.img \
        --name "Clear Linux ${LATEST}" --id df3768da-31f5-4ba6-82f0-127a1a705169
fi

if [ -f $fedora_cloud_image ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file $fedora_cloud_image \
        --name "Fedora Cloud Base 24-1.2" --id 73a86d7e-93c0-480e-9c41-ab42f69b7799
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
echo "~/local/verify.sh"
echo ""
echo "Use ciao-cli to manipulate and inspect the cluster, e.g., "
echo ""
echo "ciao-cli instance add --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=1"
echo ""
echo "When you're finished run the following command to cleanup"
echo ""
echo "~/local/cleanup.sh"
