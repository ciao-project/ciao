# ciao-down

## Introduction

ciao-down is a small utility that simplifies the task of creating,
managing and connecting to virtual machines.  All you need to have
installed on your machine is:

- Go 1.8 or greater

Then simply type

```
go get github.com/01org/ciao/testutil/ciao-down
$GOPATH/bin/ciao-down create xenial
```

to create a new Ubuntu 16.04 VM.  ciao-down will install some needed
dependencies on your local PC such as qemu and xorriso. It will then
download an Ubuntu Cloud Image and create a VM based on this image.
It will boot the VM, create an account for you, with the same
name as your account on your host, and optionally update the default
packages.

Once it's finished you'll be able to connect to the the VM via SSH
using the following command.

```
ciao-down connect
```

The command above assumes that either $GOPATH/bin or ~/go/bin is in
your PATH.

ciao-down will cache the Ubuntu image locally so the next time you
create another xenial based VM you won't have to wait very long.
It also knows about HTTP proxies and will mirror your host computer's proxy
settings inside the VMs it creates.

You can delete the VM you've just created by running,

```
ciao-down delete
```

## Workloads

Each VM is created from a workload. A workload is a text file which contains
a set of instructions for initialising the VM.  A workload, among other things,
can specify:

- The hostname of the VM.
- The resources to be consumed by the VM
- The base image from which the VM is to be created
- The folders that should be shared between the host and the VM
- An annotated cloud-init file that contains the set of instructions
  to run on the first boot of the VM.  This file is used to create
  user accounts, install packages and configure the VM.

ciao-down ships with a number of workloads for creating VMs based on standard images,
such as Ubuntu 16.04 and Fedora 25.  Users are also free to create their own workloads.
Standard workloads are stored in $GOPATH/src/github.com/01org/ciao/testutil/ciao-down/workloads.
User created workloads are stored in ~/.ciao-down/workloads.  ciao-down always checks the
~/.ciao-down/workloads directory first so if a workload exists in both directories
with the same name, ciao-down will use the workload in ~/.ciao-down/workloads.

Which workload to use is specified when creating the VM with the create
command. The workload can be a file without the .yaml extension or a URI pointing to
a local or remote file. If the workload is a file without the .yaml extension then it
must be present in either of the two directories mentioned above. For example,
the create command in the introduction section creates an Ubuntu 16.04
workload (xenial).  This caused ciao-down to load the workload definition in
$GOPATH/src/github.com/01org/ciao/testutil/ciao-down/workloads/xenial.yaml.
In the case of a remote file the supported schemes are http, https and file. Be aware
of the remote file will not be saved hence each time this option is used, ciao-down
will try to get it from the remote location. For example, to create a workload using the
file /home/x/workload.yaml we have two options.

Using the file scheme:

```
ciao-down create file:///home/ciao-down/workload.yaml
```

or its absolute path:

```
ciao-down create  /home/ciao-down/workload.yaml
```


As the majority of the ciao-down workloads are cloud-init files, ciao-down only
works with images that are designed to run cloud-init on their first boot.  Typically,
these are the cloud images, e.g.,. the [Ubuntu Cloud Images](https://cloud-images.ubuntu.com/).

## Creating new Workloads

ciao-down workloads are multi-doc YAML files.  The first document contains invariants
about the VM created from the workload such as the image off which it is based and its
hostname.  The second contains dynamic instance data that can be altered on every boot,
such as the number of CPUs the VM uses and the ports mapped.  The final document contains
the cloud-init file.  If only one section is present it is assumed to the be cloud-init
file and default values are used for the other two sections.

An example workload is shown below:

```
---
base_image_url: https://cloud-images.ubuntu.com/xenial/current/xenial-server-cloudimg-amd64-disk1.img
base_image_name: Ubuntu 16.04
...
---
mem_gib: 2
cpus: 2
...
---
#cloud-config
package_upgrade: false
runcmd:
 - {{finished .}}

users:
  - name: {{.User}}
    gecos: CIAO Demo User
    lock-passwd: true
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - {{.PublicKey}}
...
```

### Templates

[Go templates](https://golang.org/pkg/text/template/) can be used in each
document in a workload yaml file.  These templates have access to an instance
of a structure called Workspace that contains some useful information about the
host's environment and the instance that is being created.  The following pieces
of information are available via the workload structure.

 - GoPath         : The GoPath on the host or ~/go if GOPATH is not defined
 - Home           : The home directory of the user who ran ciao-down
 - HTTPProxy      : The hosts HTTP proxy if set
 - HTTPSProxy     : The hosts HTTPS proxy if set
 - NoProxy        : The value of the host's no_proxy variable
 - User           : The user who invoked ciao-down
 - PublicKey      : The public key that will be used to access the instance over SSH
 - GitUserName    : The git user name of the user who ran ciao-down
 - GitEmail       : The git email address of the user who ran ciao-down
 - Mounts         : A slice of mounts, each describing a path to be shared between guest and host
 - Hostname       : The instance hostname
 - UUID           : A UUID for the new instance
 - PackageUpgrade : Indicates whether package upgrade should be performed during the first boot.

As an example consider the third document in the workload definition above.  The User and
PublicKey fields are accessed via the {{.User}} and {{.PublicKey}} Go templates.  Abstracting
all of the user and instance specific information via templates in this way allows us to keep our
workload definitions generic.

### The Instance Specification Document

The first yaml document is called the instance specification document.  It defines the fixed
characteristics of instances created from the workload which cannot be altered.  Three
fields are currently defined:

- base_image_url  : The URL of the qcow2 image upon which instances of the workload should be based.
- base_image_name : Friendly name for the base image.  This is optional.
- hostname        : The hostname of instances created from this workload.  Hostname is also optional and defaults to singlevm if not provided.

### The Instance Data Document

The second document is called the instance data document.  It contains information that is
set when an instance is first created but may be modified at a later date, without deleting
the instance.    Four fields are currently defined:

- mem_gib : Number of Gigabytes to assign to the VM.  Defaults to half the memory on your machine
- cpus    : Number of CPUs to assign to the VM.  Defaults to half the cores on your machine
- ports   : Slice of port objects which map host ports on 127.0.0.1 to guest ports
- mounts  : Slice of mount objects which describe the folders shared between the host and the guest

Each port object has two members, host and guest.  They are both integers and they specify
the mapping of port numbers from host to guest.  A default mapping of 10022 to 22 is always
added so that users can connect to their VMs via ciao-down connect.  An example port
mapping is shown below:

```
ports:
- host: 10022
  guest: 22
```

Mount objects can be used to share folders between the host and guest.  Folders are shared
using the 9p protocol.  Each mount object has three pieces of information.

- tag            : An id for the mount.  This information is needed when mounting the shared folder in the guest
- security_model : The 9p security model to use
- path           : The path of the host folder to share

An example of a mount is given below.

```
mounts:
- tag: hostgo
  security_model: passthrough
  path: /home/user/go
```

Note that specifying a mount in the instance data document only creates a
9p device which is visible inside the guest.  To actually access the shared
folder from inside the guest you need to mount the folder.  This can be
done in the cloud-init file discussed below.

### The Cloudinit document

The third document contains a cloud-init user data file that can be used
to create users, mount folders, install and update packages and perform
general configuration tasks on the newly created VM.  The user is
expected to be familiar with the
[cloudinit file format](https://cloudinit.readthedocs.io/en/latest/).

Like the instance specification and data documents, the cloudinit document
is processed by the Go template engine before being passed to cloudinit
running inside the guest VM.  The template engine makes a number of
functions available to the cloudinit document.  These functions are used
to perform common tasks such as configuring proxies and communicating with
ciao-down running on the host.  Each function must be passed the
workspace object as the first parameter.  The following functions are
available:

#### proxyVars

Generates a string containing variable definitions for all the proxy
variables.  It is useful for prepending to commands inside the runcmd:
section of the cloudinit document that require proxies to be set, e.g,

```
- {{proxyvars .}} wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz" "/tmp/go1.8.linux-amd64.tar.gz
```

may expand to something like after being processed by the template engine

```
- https_proxy=http://myproxy.mydomain.com:911 wget https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz" "/tmp/go1.8.linux-amd64.tar.gz
```

#### proxyEnv

Generates proxy definitions in a format that is suitable for the /etc/environment file.
It should be used as follows:

```
write_files:
{{with proxyEnv . 5}}
 - content: |
{{.}}
   path: /etc/environment
{{end}}
```

#### download

Download should be used to download files from inside the guest.  Download is preferable
to directly downloading files inside the cloudinit document using curl or wget for
example, as the downloaded files are cached by ciao-down.  This means that the next time
an instance of this workload is created the file will not have to be retrieved from
the Internet, resulting in quicker boot times for the new VM.

download takes three parameters

- The workspace object
- The URL of the file to download
- The location on the guest where the downloaded file should be stored

An example of its usage is

```
- {{download . "https://storage.googleapis.com/golang/go1.8.linux-amd64.tar.gz" "/tmp/go1.8.linux-amd64.tar.gz"}}
```

#### The task functions

There are four functions that can be used to provide visual information back to the user
who invoked ciao-down on the host.  These functions should always be issued in pairs.
The first function invoked should always be beginTask.

beginTask takes two parameters.  The first is the workspace object.  The second is a
string to present to the user.  beginTask is typically followed by a command and
then a call to either endTaskOk, endTaskFail or endTaskCheck.

endTaskOk prints "[OK]" to the screen, endTaskFail prints "[FAIL]" to the screen and
endTaskCheck prints "[OK]" or "[FAIL]" depending on the result of the previous command.

For example,

```
 - {{beginTask . "Unpacking Go" }}
 - tar -C /usr/local -xzf /tmp/go1.8.linux-amd64.tar.gz
 - {{endTaskCheck .}}
```

will print

Unpacking Go : [OK]

if the tar command succeeds

and

Unpacking Go : [FAIL]

if it fails.

Reporting a failure back to ciao-down does not cause the create command to exit.  The
failure is presented to the user and the setup of the VM continues.

#### Finished

Every cloudinit document must contain a runcmd: section and that section must contain
a call to the finished function.  The function takes one parameter, the workspace
parameter.  A call to

```
 - {{finished .}}
```

is usually the last command in the runcmd section.  Remember it's currently mandatory.

### Automatically mounting shared folders

As previously mentioned mounts specified in the instance data document will only
create 9p devices in the guest.  In order to access the files in the shared folder
you need to arrange to have these devices mounted.  The way you do this might
differ from distro to distro.  On Ubuntu it is done by adding a mount: section
to the cloudinit document, e.g.,

```
mounts:
{{range .Mounts}} - [{{.Tag}}, {{.Path}}, 9p, "x-systemd.automount,x-systemd.device-timeout=10,nofail,trans=virtio,version=9p2000.L", "0", "0"]
{{end -}}
```

The above command will arrange for all mounts specified in the instance data document or on
the create command line to be mounted to the same location that they are mounted on the
host.

Mounts added later via the start command will need to be mounted manually.

## Commands

### create

ciao-down create creates and configures a new ciao-down VM.  Currently,
only one VM can be configured at any one time.  All the files associated
with the VM are stored in ~/.ciao-down.

An example of ciao-down create is given below:

```
$ ./ciao-down create xenial
Booting VM with 7 GB RAM and 4 cpus
Booting VM : [OK]
Downloading Go : [OK]
Unpacking Go : [OK]
Installing apt-transport-https and ca-certificates : [OK]
Add docker GPG key : [OK]
Retrieving updated list of packages : [OK]
Upgrading : [OK]
Installing Docker : [OK]
Installing GCC : [OK]
Installing Make : [OK]
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

By default, ciao-down will assign half of your host's resources to the VM
that it creates and launches.  If you have 8 CPUs and 8 GB of RAM on your
host, ciao-down will assign 4GB of RAM and 4 VCPUs to the guest VM.  You
can control this behaviour by using the --mem and --cpu options.  For
example,

ciao-down create --cpus 2 -mem 2 ciao

Creates and boots a VM with 2 VCPUs and 2 GB of RAM.

The --package-upgrade option can be used to provide a hint to workloads
indicating whether packages contained within the base image should be updated or not
during the first boot.  Updating packages can be quite time consuming
and may not be necessary if the user just wants to create a throw away
VM to test something under a particular distro.  This hint only works if
the workloads define the following in their cloudinit documents.

```
package_upgrade: {{with .PackageUpgrade}}{{.}}{{else}}false{{end}}
```

#### Port mappings and Mounts

By default, ciao-down creates one port mapping for new VMs, 10022-22 for SSH
access.  You can specify additional port mappings or mounts or even override
the default settings on the command line.

For example,

./ciao-down create --mount docs,passthrough,$HOME/Documents --port 10000-80 xenial

shares the host directory, $HOME/Documents, with the ciao-down VM using the 9p
passthrough security model.  The directory can be mounted inside the VM using
the docs tag.  The command also adds a new port mapping.  127.0.0.1:10000 on
the host now maps to port 80 on the guest.

Multiple --mount and --port options can be provided and it's also possible to
override the default ports.  Default mounts, ones specified in the
instance data document, can be overridden by specifying
an existing tag with new options, and default ports can be overridden by
mapping a new host port to an existing guest port.  For example,

./ciao-down create --mount hostgo,none,$HOME/go -port 10023-22 xenial

changes the security model of the mount with the hostgo tag and makes the instance
available via ssh on 127.0.0.1:10023.


### delete

ciao-down delete, shuts down and deletes all the files associated with the VM.

### status

ciao-down status provides information about the current ciao-down VM, e.g., whether
it is running, and how to connect to it.  For example,

```
$ ciao-down status
Status	:	ciao up
SSH	:	ssh -q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i /home/user/.ciao-down/id_rsa 127.0.0.1 -p 10022
```

### stop

ciao-down stop is used to power down the ciao-down VM cleanly.

### start

ciao-down start boots a previously created but not running ciao-down VM.
The start command also supports the --mem and --cpu options.  So it's
possible to change the resources assigned to the guest VM by stopping it
and restarting it, specifying --mem and --cpu.  It's also possible to
specify additional port mappings or mounts via the start command.  See
the section on port mappings below.

Any parameters you pass to the start command override the parameters
you originally passed to create.  These settings are also persisted.
For example, if you were to run

```
./ciao-down create --mem=2 xenial
./ciao-down stop
./ciao-down start --mem=1
./ciao-down stop
./ciao-down start
```

The final ciao-down instance would boot with 1GB of RAM even though no mem
parameter was provided.

### quit

ciao-down quit terminates the VM immediately.  It does not shut down the OS
running in the VM cleanly.

