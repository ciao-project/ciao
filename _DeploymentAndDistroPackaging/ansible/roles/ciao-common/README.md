# clearlinux.ciao-common
This role is a requirement for other ciao-* roles

## Requirements
None

## Role Variables
The following variables are available for all ciao roles

Variable  | Default Value | Description
--------  | ------------- | -----------
cephx_user | admin | cephx user to login into the ceph cluster
skip_ceph | False | When set to true, ansible will not configure ceph on Ciao nodes
ciao_dev | False | Set to True to install from source, otherwise install form OS packages
gopath | /tmp/go | golang GOPATH
ciao_controller_fqdn | `{{ ansible_fqdn }}` | FQDN for CIAO controller node
cnci_image_url | [clear-8260-ciao-networking.img.xz](https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz) | URL for the latest ciao networking image
ovmf_url | [OVMF.fd](https://download.clearlinux.org/image/OVMF.fd) | EFI firmware required for CNCI Image.
clear_cloud_image_url | [clear-10820-cloud.img.xz](https://download.clearlinux.org/releases/10820/clear/clear-10820-cloud.img.xz) | URL for the latest clearlinux cloud image

## Dependencies
None

## Example Playbook
None

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
