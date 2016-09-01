# CIAO Deployment with Ansible Playbooks

* Work in Progress

## Prework

 - Deploy SSH key from the host running ansible, to all hosts in the cluster setup.
 - Install Ansible and his dependencies.

## Environment setup

### Edit the hosts file according to your cluster setup
```shell
hosts.yml

[controllers]
controller.example.com

[networks]
network.example.com

[computes]
compute1.example.com
compute2.example.com
compute3.example.com
```

### Run the playbooks
```shell
ansible-playbook -i hosts ciao.yml --private-key=<ssh_key>
```

For more information visit the project clear-config-management: https://github.com/clearlinux/clear-config-management
