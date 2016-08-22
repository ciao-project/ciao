# ciao-controller
Ansible role to install the controller node for a CIAO cluster

This role configures the following components

* ciao scheduler
* ciao controller

## Requirements
None

## Role Variables
The available variables for this roles are the variables from [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common) plus the following:

Note: Mandatory variables are shown in **bold**

Variable  | Default Value | Description
--------  | ------------- | -----------
ciao_controller_ip | `{{ ansible_default_ipv4['address'] }}` | IP Address for CIAO controller node
ciao_controller_fqdn | `{{ ansible_fqdn }}` | # FQDN for CIAO controller node
ciao_mgmt_subnets | `{{ ansible_default_ipv4['network'] }}` | CIAO management subnets
ciao_compute_subnets | `{{ ciao_mgmt_subnet }}` | CIAO compute subnets
ciao_service_user | csr | OpenStack user for CIAO services
ciao_service_password | secret | Password for `ciao_service_user`
ciao_admin_email | admin@example.com | CIAO administrator email address
ciao_cert_organization | Example Inc. | Name of the organization running the CIAO cluster
ciao_guest_user | demouser | CIAO virtual machines can be accessed with this username and it's public key
ciao_guest_key | ~/.ssh/id_rsa.pub | A path to an SSH public authentication key for `ciao_guest_user`

**WARNING**: `ciao_guest_user` and `ciao_guest_key` are a temporary development feature. They give the developer running a dev/test ciao cluster superuser ssh access to all compute workload instances and also all cnci instances. In the future this will be removed when cloud-init and user specified workloads are enabled in the webui and cli.

## Dependencies
* [ciao-common](https://github.com/clearlinux/clear-config-management/tree/master/roles/ciao-common)

## Example Playbook
file *ciao_controller.yml*
```
- hosts: controllers
  roles:
    - ciao-controller
```

file *group_vars/all*
```
keystone_fqdn: identity.example.com
keystone_admin_password: secret

ciao_service_user: csr
ciao_service_password: secret
ciao_guest_user: demouser
ciao_guest_key: ~/.ssh/id_rsa.pub
```

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
