# clearlinux.ciao-network
Ansible role to install the network node for a CIAO cluster

## Requirements
* qemu-kvm

## Role Variables
The available variables for this roles are the variables from [clearlinux.ciao-common](https://github.com/clearlinux/ansible-role-ciao-common)

## Dependencies
* [clearlinux.ciao-common](https://github.com/clearlinux/ansible-role-ciao-common)

## Example Playbook
file *ciao.yml*
```
- hosts: networks
  roles:
    - clearlinux.ciao-network
```

file *group_vars/all*
```
ciao_controller_fqdn: controller.example.com
```

## Contribution
**Pull Requests and Issues should be opened at [clearlinux/clear-config-management](https://github.com/clearlinux/clear-config-management).**

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
