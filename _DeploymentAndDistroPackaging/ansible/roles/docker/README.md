docker
=========

This role installs docker 1.12 on Ubuntu and ClearLinux

Requirements
------------

None

Role Variables
--------------

Variable  | Default Value | Description
--------  | ------------- | -----------
swupd_args |  | arguments for `swupd` (clearlinux)

Dependencies
------------

None

Example Playbook
----------------

    - hosts: servers
      roles:
         - docker

License
-------

Apache

Author Information
------------------

This role was created by [Alberto Murillo](alberto.murillo.silva@intel.com)
