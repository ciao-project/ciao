#!/bin/bash

ciao_host=$(hostname)
ciao_bin="$HOME/local"
ciao_cert="$ciao_bin""/cert-Scheduler-""$ciao_host"".pem"

ciao_email="ciao-devel@lists.clearlinux.org"
ciao_org="Intel"
ciao_src="$GOPATH"/src/github.com/01org/ciao
ciao_gobin="$GOPATH"/bin
ciao_scripts="$GOPATH"/src/github.com/01org/ciao/testutil/singlevm
ciao_env="$ciao_bin/demo.sh"
ciao_ctl_log="/var/lib/ciao/logs/controller/ciao-controller.ERROR"
ciao_cnci_image="clear-8260-ciao-networking.img"
download=0


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

sudo mkdir -p /var/lib/ciao/images
if [ ! -d /var/lib/ciao/images ]
then
	echo "FATAL ERROR: Unable to create /var/lib/ciao/images"
	exit 1
fi

sudo mkdir -p /etc/ciao/
if [ ! -d /etc/ciao ]
then
	echo "FATAL ERROR: Unable to create /etc/ciao"
	exit 1
fi
sudo cp "$ciao_scripts"/configuration.yaml /etc/ciao

#Stop any running agents and CNCIs
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sudo killall qemu-system-x86_64
sudo rm -rf /var/lib/ciao/instances
sleep 5


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
	curl -O https://download.clearlinux.org/image/clear-"${LATEST}"-cloud.img.xz
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
"$ciao_gobin"/ciao-cli --list-workloads
"$ciao_gobin"/ciao-cli --launch-instances --workload=e35ed972-c46c-4aad-a1e7-ef103ae079a2 --instances=2
"$ciao_gobin"/ciao-cli --list-instances
"$ciao_gobin"/ciao-cli --launch-instances --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=2
sleep 30
"$ciao_gobin"/ciao-cli --list-instances

#Check docker networking
docker_id=$(sudo docker ps -q | head -1)
sudo docker logs "$docker_id"

#Check SSH connectivity
sleep 2
"$ciao_gobin"/ciao-cli --list-instances
ssh_ip=$(ciao-cli --list-instances |  grep "SSH IP:" | sed 's/^.*SSH IP: //' | head -1)
head -1 < /dev/tcp/"$ssh_ip"/33002

#Now delete all instances
"$ciao_gobin"/ciao-cli --all-instances --delete-instance
"$ciao_gobin"/ciao-cli --list-instances
#Also kill the CNCI (as there is no other way to delete it today)
sudo killall qemu-system-x86_64
sudo rm -rf /var/lib/ciao/instances
