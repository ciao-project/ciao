#!/bin/bash

if [ -z "$1" ]; then
        IMAGE=clear.img
else
        IMAGE="$1"
fi

if [ -z "$2" ]; then
        VNIC="ERROR"
else
        VNIC="$2"
fi

if [ -z "$3" ]; then
        MAC="DE:AD:DE:AD:DE:AD"
else
        MAC="$3"
fi

if [[ "$IMAGE" =~ .xz$ ]]; then
        >&2 echo "File \"$IMAGE\" is still xz compressed. Uncompress it first with \"unxz\""
        exit 1
fi

if [ ! -f "$IMAGE" ]; then
        >&2 echo "Can't find image file \"$IMAGE\""
        exit 1
fi
rm -f debug.log

qemu-system-x86_64 \
        -enable-kvm \
        -bios OVMF.fd \
        -smp cpus=4,cores=2 -cpu host \
        -vga none -nographic \
        -drive file="$IMAGE",if=virtio,aio=threads \
        -net nic,model=virtio,macaddr=$3 -net tap,ifname=$2,script=no,downscript=no \
        -debugcon file:debug.log -global isa-debugcon.iobase=0x402
