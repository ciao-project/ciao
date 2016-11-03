# clearlinux.docker
This role installs docker 1.12 on Ubuntu, Fedora and ClearLinux

## Requirements
None

## Role Variables

Variable  | Default Value | Description
--------  | ------------- | -----------
swupd_args |  | arguments for `swupd` (clearlinux)

## Dependencies
None

## Example Playbook
file *site.yml*
```
- hosts: servers
  roles:
    - clearlinux.docker
```

## Contribution
**Pull Requests and Issues should be opened at [clearlinux/clear-config-management](https://github.com/clearlinux/clear-config-management).**

## License
Apache-2.0

## Author Information
This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
