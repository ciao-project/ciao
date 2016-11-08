# Development Mode
This playbooks installs ciao from the latest ciao packaged for your distro.

If you want to try the latest features, there is a way to use this playbook to
deploy ciao from github.com/01org/ciao master branch source code.

Set `ciao_dev = True` in [group_vars/all](../group_vars/all) file or pass the
command line argument `--extra-vars "ciao_dev=true"`
