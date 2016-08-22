# ciao-common
This role is a requirement for other ciao roles

## Requirements
None

## Role Variables
The following variables are available for all ciao roles

Variable  | Default Value | Description
--------  | ------------- | -----------
ciao_controller_fqdn | `{{ ansible_fqdn }}` | FQDN for CIAO controller node
cnci_image_url | [clear-8260-ciao-networking.img.xz](https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz) | URL for the latest ciao networking image
ovmf_url | [OVMF.fd](https://download.clearlinux.org/image/OVMF.fd) | EFI firmware required for CNCI Image.
fedora_cloud_image_url | [Fedora-Cloud-Base-23-20151030.x86_64.qcow2](https://dl.fedoraproject.org/pub/fedora/linux/releases/23/Cloud/x86_64/Images/Fedora-Cloud-Base-23-20151030.x86_64.qcow2) | URL for the latest fedora cloud image
clear_cloud_image_url | [clear-8970-cloud.img.xz](https://download.clearlinux.org/image/clear-8970-cloud.img.xz) | URL for the latest clearlinux cloud image

## Dependencies
None

## Example Playbook
None

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
