# ciao-network
Ansible role to install the network node for a CIAO cluster

## Requirements
* qemu-kvm

## Role Variables
The available variables for this roles are the variables from [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common)

## Dependencies
* [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common)

## Example Playbook
```
- hosts: ciao_network
  roles:
    - ciao-network
```

file *group_vars/all*
```
ciao_controller_fqdn: controller.example.com
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
