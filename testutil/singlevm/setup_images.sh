pushd $ciao_bin
#Download the firmware
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

echo -n "Waiting up to $ciao_image_wait_time seconds for the ciao image" \
    "service to become available "
try_until=$(($(date +%s) + $ciao_image_wait_time))
while : ; do
    while [ $(date +%s) -le $try_until ]; do
        if "$ciao_gobin"/ciao-cli image list > /dev/null 2>&1; then
            echo " READY"
            break 2
        else
            echo -n .
            sleep 1
        fi
    done
    echo FAILED
    break
done

echo ""
echo "Uploading CNCI image to image service"
echo "--------------------------------------------------------"
if [ -f "$ciao_cnci_image".qcow ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file "$ciao_cnci_image".qcow \
        --name "ciao CNCI image" --id 4e16e743-265a-4bf2-9fd1-57ada0b28904 \
	--visibility internal
fi
popd
