# clearlinux.ciao-compute
Ansible role to install the compute node for a CIAO cluster

## Requirements
* docker
* qemu-kvm
* xorriso

## Role Variables
The available variables for this roles are the variables from [clearlinux.ciao-common](https://github.com/clearlinux/ansible-role-ciao-common)

## Dependencies
* [clearlinux.ciao-common](https://github.com/clearlinux/ansible-role-ciao-common)

## Example Playbook
file *ciao.yml*
```
- hosts: computes
  roles:
    - clearlinux.ciao-compute
```

file *group_vars/all*
```
ciao_controller_fqdn: controller.example.com
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
