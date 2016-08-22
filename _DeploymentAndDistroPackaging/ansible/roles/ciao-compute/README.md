# ciao-compute
Ansible role to install the compute node for a CIAO cluster

## Requirements
* docker
* qemu-kvm
* xorriso

## Role Variables
The available variables for this roles are the variables from [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common)

## Dependencies
* [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common)

## Example Playbook
file *ciao_compute.yml*
```
- hosts: ciao_compute
  roles:
    - ciao-compute
```

file *group_vars/all*
```
ciao_controller_fqdn: controller.example.com
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
