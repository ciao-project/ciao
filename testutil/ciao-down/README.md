# ciao-down

ciao-down is a small utility for setting up a VM that contains
everything you need to run ciao's Single VM. All you need to have
installed on your machine is:

- Go 1.7 or greater
- An ssh key

Then simply type

```
go get github.com/01org/ciao/testutil/ciao-down
$GOPATH/bin/ciao-down prepare
```

ciao-down will install some needed dependencies on your local PC such
as qemu and xorriso. It will then download an Ubuntu Cloud Image and
create a VM based on this image. It will boot the VM and install in that
VM everything you need to run ciao Single VM, including docker, ceph,
go, gcc, etc. When ciao-down prepare has finished you can connect to the
newly created VM with

```
$GOPATH/bin/ciao-down connect
```

Your host's GOPATH is mounted inside the VM. Thus you can edit your
the ciao code on your host machine and test in Single VM.

Follow the instructions at the prompt to start Single VM.

The nice thing about ciao-down is that it is built using existing ciao
packages. It uses osprepare to install dependencies on the host, some
launcher code to create the images needed by the VMs, and the qemu
package to launch and manage the VMs. All the rest is done via cloud-init.

## ciao-down commands

### prepare

ciao-down prepare creates and configures a new ciao-down VM.  Currently,
only one VM can be configured at any one time.  All the files associated
with the VM are stored in ~/.ciao-down.  The first time you run ciao-down
it will download the Ubuntu 16.04 cloud image which takes some time.  This
image is cached in ~/.ciao-down for future use.

An example of ciao-down prepare is given below:

```
$ ./ciao-down prepare
Checking environment
Installing host dependencies
OS Detected: ubuntu
Booting VM : [OK]
Downloading Go : [OK]
Unpacking Go : [OK]
Installing apt-transport-https and ca-certificates : [OK]
Add docker GPG key : [OK]
Retrieving updated list of packages : [OK]
Upgrading : [OK]
Installing Docker : [OK]
Installing GCC : [OK]
Installing QEMU : [OK]
Installing xorriso : [OK]
Installing ceph-common : [OK]
Installing Openstack client : [OK]
Auto removing unused components : [OK]
Building ciao : [OK]
Installing Go development utils : [OK]
Pulling ceph/demo : [OK]
Pulling clearlinux/keystone : [OK]
Downloading Fedora-Cloud-Base-24-1.2.x86_64.qcow2 : [OK]
Downloading CNCI image : [OK]
Downloading latest clear cloud image : [OK]
Setting git user.name : [OK]
Setting git user.email : [OK]
VM successfully created!
Type ciao-down connect to start using it.
```

### delete

ciao-down delete, shuts down and deletes all the files associated with the VM.

### status

ciao-down status provides information about the current ciao-down VM, e.g., whether
it is running, and how to connect to it.  For example,

```
$ ciao-down status
Status	:	ciao up
SSH	:	ssh 127.0.0.1 -p 10022
```

### stop

ciao-down stop is used to power down the ciao-down VM cleanly.

### start

ciao-down start boots a previously prepared but not running ciao-down VM.

### quit

ciao-down quit terminates the VM immediately.  It does not shut down the OS
running in the VM cleanly.



