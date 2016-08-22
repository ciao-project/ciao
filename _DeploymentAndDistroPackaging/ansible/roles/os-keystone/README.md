# os-keystone
This role installs clearlinux/keystone docker container

## Requirements
* docker

## Role variables

Variable | Default Value | Description
-------- | ------------- | -----------
mysql_root_password | secret | Password for the `root` user in the database
keystone_fqdn(optional) | `{{ ansible_fqdn }}` | Fully Qualified Domain Name for Keystone server

#### The following variables can be used to specify custom services, projects, users and roles

<table>
<tr>
<td><b>Variable</b></td>
<td><b>Example</b></td>
<td><b>Description</b></td>
</tr>

<tr>
  <td>keystone_services</td>
  <td><pre><code>
    keystone_services:
      - service_name: nova
        service_type: compute
        description: OpenStack Compute Service
  </code></pre></td>
  <td>A list of services to be created</td>
</tr>

<tr>
  <td>keystone_projects</td>
  <td><pre><code>
    keystone_projects:
      - project_name: demo
        description: Demo Project
  </code></pre></td>
  <td>A list of projects to be created</td>
</tr>

<tr>
  <td>keystone_users</td>
  <td><pre><code>
    keystone_users:
      - user_name: demo
        password: secret
        project_name: demo
        email: demo@example.com
  </code></pre></td>
  <td>A list of users to be created</td>
</tr>

<tr>
  <td>keystone_roles</td>
  <td><pre><code>
    keystone_roles:
      - demo
      - admin
  </code></pre></td>
  <td>A list of roles to be created</td>
</tr>

<tr>
  <td>keystone_user_roles</td>
  <td><pre><code>
    keystone_user_roles:
      - user_name: demo
        project_name: demo
        role_name: demo
  </code></pre></td>
  <td>A list of user, role mappings</td>
</tr>

</table>

## Dependencies
None

## Example playbook
file *keystone.yml*
```
- hosts: openstack_identity
  roles:
    - os-keystone
```

file *group_vars/all*
```
mysql_root_password: secret
keystone_fqdn: identity.example.com

keystone_projects:
  - project_name: demo

keystone_users:
  - user_name: demo
    password: secret
    project_name: demo

keystone_roles:
  - demo

keystone_user_roles:
  - user_name: demo
    project_name: demo
    role_name: demo
```

## License
Apache-2.0

## Author Information
This role was created by [Leoswaldo Macias](leoswaldo.macias@intel.com) and [Obed Munoz](obed.n.munoz@intel.com)
