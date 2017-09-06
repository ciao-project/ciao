# Setting up a Multi-Node Kubernetes (k8s) Cluster with ciao

ciao is a suite of Go programs designed to make it easy to set up and configure a private cloud.  It can orchestrate both containers and VMs in multiple tenants across multiple nodes.  It is designed to be fast and scalable and it enables security by default.  All control plane communication within the cluster is encrypted and each tenant gets its own private network.  All of these traits make ciao an ideal private undercloud for Kubernetes and the good news is, once you have a running ciao cluster, you can set up a complete k8s cluster with a single command.

In this article we will take a look at the simplest way of installing k8s on top of a ciao cluster which requires only a single machine.  We will do this by

- Installing Go
- Configuring proxies [OPTIONAL] 
- Downloading and building ciao-down
- Creating a VM in which to run ciao
- Logging into the VM and starting the ciao cluster
- Creating our k8s cluster
- Shutting down our k8s and ciao clusters

## Prerequisites

To follow the instructions below you will need a Linux system with at least 8GB of RAM.  Ideally, running Ubuntu 16.04 as that is what we use to develop ciao and is the environment in which ciao is most heavily tested.

The ciao development environment runs inside a VM managed by a tool called ciao-down which we will introduce shortly.  ciao itself will launch some VMs inside the ciao-down created VM for its own internal use and to host our k8s cluster.  To launch VMs ciao requires KVM to be enabled, hence KVM must be available inside the ciao-down VM.  In order for this to work our host machine needs to have nested KVM enabled.  This is enabled by default in Ubuntu 16.04 but may not be enabled on other distributions.  If nested KVM is not enabled you will get an error when running ciao-down in step 4.

On systems with Intel based CPUs you can verify that nested KVM is enabled by typing

```
$ cat /sys/module/kvm_intel/parameters/nested
Y
```

If all is well you should see 'Y' printed out on the console.  If you see 'N' you’ll need to enable nested KVM.  This can be done as follows

```
$ echo "options kvm-intel nested=1" | sudo tee /etc/modprobe.d/kvm-intel.conf
```

The code we are going to run will be downloaded using the Go tool chain.  Go has a dependency on git, therefore you must install git if it is not already present on your machine.  This can be done on Ubuntu as follows

```
$ sudo apt-get install git
```

## Installing Go

Download the latest stable version of Go from [here]( https://golang.org/dl/  ) and follow the installation instructions.  ciao requires Go 1.8 or later.

You should also ensure that the directory into which Go installs its binaries is present in your path.  You can do this by executing the following command:

```
export PATH=$PATH:$(go env GOPATH)/bin
```

## Configuring Proxies

**Skip this section if you do not access the Internet through a proxy.**

If your computer accesses the Internet through a proxy, you should make sure that the proxy environment variables are correctly configured in your shell.  This is important for two reasons:

The Go command used to download ciao will fail if it cannot find the proxy.
ciao-down will replicate your local proxy settings in all the VMs it creates.  It will ensure that proxy environment variables are correctly initialised for all users, and that both docker and the package managers, such as APT, are made aware of the appropriate proxy settings.  If the proxy environment variables are not correctly configured, ciao-down cannot do this.

So, assuming that you are using a corporate proxy you should enter the following commands, replacing the URLs and the domain names with values that are appropriate to your environment.

```
$ export http_proxy=http://my-proxy.my-company.com:port 
$ export https_proxy=http://my-proxy.my-company.com:port
$ export no_proxy=.my-company.com
```

## Downloading and Building ciao-down

Once Go is installed downloading and installing ciao-down is easy.  Simply type

```
$ go get github.com/01org/ciao/testutil/ciao-down
```

## Creating a VM in which to run ciao

The next step is to create a custom VM for running ciao.  This might sound complicated, and it is, but luckily the entire process is automated for us by a tool called ciao-down.  ciao-down is a small utility designed to create and manage custom VMs built from cloud images.  To create a VM we need to provide ciao-down with a set of instructions called a workload.  A workload for ciao has already been created, so to make a new VM designed to run ciao you simply need to type.

```
$ ciao-down create -mem=6 -cpus=2 ciao
```

This will create a new VM with 6GBs of memory and 2 VCPUs, which is the minimum needed for hosting a k8s cluster inside ciao-down.

ciao-down has dependencies on a number of other components, such as qemu.  The first thing it will do when executed is to check to see whether these packages are present on your host computer.  If they are not, it will install them, asking you for your password if necessary. 

The ciao-down create command has a lot of work to do so it can take some time to run.  In short, it performs the following tasks.

- Installs the dependencies ciao-down needs on the host machine
- Downloads an Ubuntu 16.04 cloud image
- Creates and boots a new VM based on this image
- Installs all of the dependencies needed by ciao inside this VM
- Updates the guest OS
- Creates a user account inside the VM with SSH enabled

An edited example ciao-down output is shown below

```
$ ciao-down create -mem=6 -cpus=2 ciao
Installing host dependencies
OS Detected: ubuntu
Missing packages detected: [xorriso]
[sudo] password for user: 
Reading package lists...
Building dependency tree...
Reading state information...
The following NEW packages will be installed
  xorriso
….
[SNIP]
….
Missing packages installed.
Downloading Ubuntu 16.04
Downloaded 10 MB of 287
….
[SNIP]
….
Downloaded 287 MB of 287
Booting VM with 6 GB RAM and 2 cpus
Booting VM : [OK]
Adding singlevm to /etc/hosts : [OK]
Mounting /home/markus/go-fork : [OK]
Downloading Go : [OK]
Unpacking Go : [OK]
Installing apt-transport-https and ca-certificates : [OK]
Add docker GPG key : [OK]
Add Google GPG key : [OK]
Retrieving updated list of packages : [OK]
Installing Docker : [OK]
Installing kubectl : [OK]
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
Downloading Fedora-Cloud-Base-24-1.2.x86_64.qcow2 : [OK]
Downloading xenial-server-cloudimg-amd64-disk1.img : [OK]
Downloading CNCI image : [OK]
Downloading latest clear cloud image : [OK]
Setting git user.name : [OK]
Setting git user.email : [OK]
VM successfully created!
Type ciao-down connect to start using it.
```

Please see the Troubleshooting section near the bottom of this document if this command fails.

## Starting the ciao Cluster

Now our VM has been created we need to log into it and start our ciao cluster.  This is easily done.  To log into the VM simply type

```
$ ciao-down connect
```

This will connect to the VM via SSH using a private key created specifically for this VM when we ran ciao-down create.  You will be presented with a set of instructions explaining how to start the ciao cluster.

```
Welcome to Ubuntu 16.04.2 LTS (GNU/Linux 4.4.0-83-generic x86_64)

To run Single VM:

cd /home/<user>/go-fork/src/github.com/01org/ciao/testutil/singlevm
./setup.sh
```

Follow the instructions for running Single VM, replace <user> with your user name, e.g.,

```
$ cd /home/<user>/go-fork/src/github.com/01org/ciao/testutil/singlevm
$ ./setup.sh
```

If everything works okay the script should terminate with output that looks similar to the following.

```
Your ciao development environment has been initialised.
To get started run:

. ~/local/demo.sh

Verify the cluster is working correctly by running

~/local/verify.sh

Use ciao-cli to manipulate and inspect the cluster, e.g., 

ciao-cli instance add --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=1

When you're finished run the following command to cleanup

~/local/cleanup.sh
```

To communicate with the newly created ciao cluster we need to initialise some environment variables.  This can be done by sourcing the file ~/local/demo.sh, e.g.,

```
$ . ~/local/demo.sh
```

We can then run a few simple ciao commands to check that everything is working correctly.  To get started let’s enumerate the list of ciao workloads.  A workload is a set of instructions for creating an instance (VM or a container) in ciao.  Our new ciao cluster comes with some predefined workloads, which you can see if you execute

```
$ ciao-cli workload list
Workload 1
	Name: Ubuntu latest test container
	UUID:332178db-aeeb-463c-b551-09780cfd7c57
	CPUs: 2
	Memory: 128 MB
Workload 2
	Name: Debian latest test container
	UUID:373eeebd-3daa-4377-a73d-2bf627f35977
	CPUs: 2
	Memory: 128 MB
Workload 3
	Name: Ubuntu test VM
	UUID:0495f35c-f1ab-46fa-9a07-676a210f383e
	CPUs: 2
	Memory: 256 MB
```

There are two workloads for creating container instances and one for creating a VM instance.  We’ll be creating some new VM workloads later on for our k8s master and worker nodes.

Now let’s enumerate the list of instances.  This can be done as follows

```
$ ciao-cli instance list
```

You should see that there are no instances.  This will change when we set up our k8s cluster.

Finally, should you get an error when running the setup.sh script to start ciao you could try tearing the cluster down using, ~/local/cleanup.sh and then re-running setup.sh.

## Creating a k8s Cluster

We’re going to set up our k8s cluster using another tool called kubicle.  Kubicle is a command line tool for creating Kubernetes clusters on top of an existing ciao cluster. It automatically creates ciao workloads for the various k8s roles (master and worker), creates instances from these workloads which self-form into a k8s cluster, and extracts the configuration information needed to control the new cluster from the master node.  The kubicle is installed by the setup.sh script we’ve just run.

Creating a Kubernetes cluster is easy. Kubicle only needs one piece of information, the UUID of the image to use for the k8s nodes. Currently, this UUID must refer to an Ubuntu server image, as the workloads created by kubicle for the k8s nodes assume Ubuntu.   Luckily the setup.sh script we ran earlier uploads an Ubuntu server image into ciao’s image service for us.  All we need to do is to determine its UUID.  We can do this using the ciao-cli image list command, e.g.,

```
$ ciao-cli image list
Image #1
	Name             [Ubuntu Server 16.04]
	Size             [2361393152 bytes]
	UUID             [bf36c771-d5cc-47c4-b965-78eaca505229]
	Status           [active]
	Visibility       [public]
	Tags             []
	CreatedAt        [2017-06-07 10:41:49.37279755 +0000 UTC]
```

Now we simply need to run the kubicle create command specifying the UUID (bf36c771-d5cc-47c4-b965-78eaca505229) of the above image, e.g.,

```
$ kubicle create --external-ip=198.51.100.2 bf36c771-d5cc-47c4-b965-78eaca505229
Creating master
Creating workers
Mapping external-ip
Instances launched.  Waiting for k8s cluster to start

k8s cluster successfully created
--------------------------------
Created master:
 - f6c494d1-e4b4-4c26-a663-0dab4dfb15db
Created 1 workers:
 - aaed1545-2125-4c59-8711-583c26c3299b
Created external-ips:
- 198.51.100.2
Created pools:
- k8s-pool-46959cfe-f584-45f1-9218-50ea3549a0ee
To access k8s cluster:
- export KUBECONFIG=$GOPATH/src/github.com/01org/ciao/testutil/singlevm/admin.conf
- If you use proxies, set
  - export no_proxy=$no_proxy,198.51.100.2
  - export NO_PROXY=$NO_PROXY,198.51.100.2
```

When executing this command, make sure to replace the image UUID with the UUID of the Ubuntu Server 16.04 image reported by running ciao-list on your cluster.  You shouldn’t change anything else, i.e., include the --external-ip=198.51.100.2 option verbatim.  The --external-ip option provides an IP address that can be used to administer the k8s cluster and to access services running within it.  The address 198.51.100.2 is safe to use inside a ciao-down VM created to run ciao.

Looking at the output of the kubicle command we can see that it has created a number of ciao objects for us.  It has created two new ciao workloads, one for the master and one for the workers.  From these workloads it has created two VM instances, one master node and one worker node.  Finally, it has created an external ip address for us which it has associated with the master node.  We’ll use this address to access the k8s cluster a little later.  Let’s inspect these new objects using the ciao-cli tool.  If you execute ciao-cli workload list, you should now see five workloads, the final two of which have just been created by kubicle.

```
$ ciao-cli workload list
...
[ SNIP ] 
...
Workload 4
	Name: k8s master
	UUID:a92ce5be-922d-445a-ab3c-8f8a42e043ac
	CPUs: 1
	Memory: 1024 MB
Workload 5
	Name: k8s worker
	UUID:ead96d3d-b23f-4c43-9921-4756d274e51d
	CPUs: 1
	Memory: 2048 MB
```

Now let’s enumerate our instances, i.e. our running VMs and containers using the ciao-cli instance list command.  You may remember that the last time we ran this command there was no output.  Running it again should show that we have two VM instances running.

```
$ ciao-cli instance list
# UUID                                 Status Private IP SSH IP        SSH PORT
1 f6c494d1-e4b4-4c26-a663-0dab4dfb15db active 172.16.0.2 198.51.100.59 33002
2 aaed1545-2125-4c59-8711-583c26c3299b active 172.16.0.3 198.51.100.59 33003
```

We can manipulate our newly formed k8s cluster using the kubectl tool.  Kubectl expects an environment variable, KUBECONFIG, to be set to the path of a configuration file which contains the configuration settings for the cluster.  Luckily for us, kubicle creates a configuration file for our new cluster and even provides us with the command needed to initialise KUBECONFIG.  If you scroll back up to where you executed the kubicle command you should see the following.

```
To access k8s cluster:
- export KUBECONFIG=$GOPATH/src/github.com/01org/ciao/testutil/singlevm/admin.conf
```

Execute this command.  If your ciao-down instance is running behind a proxy, you will also need to add the external-ip address we specified earlier to your no_proxy settings.  The reason for this is that kubectl will access the k8s cluster via this external ip address and we don’t want this access to go through a proxy.  Again the status message printed by the kubicle create command provides us with the commands we need to execute.

```
- If you use proxies, set
  - export no_proxy=$no_proxy,198.51.100.2
  - export NO_PROXY=$NO_PROXY,198.51.100.2
```

We’re now ready contact our k8s cluster.  First let’s check the nodes have successfully joined the cluster.

```
$ kubectl get nodes
NAME                                   STATUS    AGE       VERSION
0d112fc1-8b4b-427c-b258-e93c1ad989e6   Ready     5m        v1.6.7
38ec8bd5-3c50-477f-9b59-b8b332609551   Ready     6m        v1.6.7
```

They have and they’re ready to use.  Note it can take up to a minute for worker nodes to join the k8s cluster and transition to the Ready state.   If you only see one node listed, wait half a minute or so and try again.

Now we've got our kubectl tool running let's create a deployment

```
$ kubectl run nginx --image=nginx --port=80 --replicas=2
deployment "nginx" created
$ kubectl get pods
NAME                    READY     STATUS              RESTARTS   AGE
nginx-158599303-0p5dj   0/1       ContainerCreating   0          6s
nginx-158599303-th0fc   0/1       ContainerCreating   0          6s
```

So far so good. We've created a deployment of nginx with two pods. Let's now expose that deployment via an external IP address. There is a slight oddity here. We don't actually specify the external IP we passed to the kubicle create command. Instead we need to specify the ciao internal IP address of the master node, which is associated with the external IP address we passed to the create command. The reason for this is due to the way ciao implements external IP addresses. ciao’s networking translates external IP addresses into internal IP addresses and for this reason our k8s services need to be exposed using the ciao internal addresses. To find out which address to use, execute the ciao-cli external-ip list command, e.g.,

```
$ ciao-cli external-ip list
# ExternalIP   InternalIP InstanceID
1 198.51.100.2 172.16.0.2 38ec8bd5-3c50-477f-9b59-b8b332609551
```

You can see from the above command that the internal IP address associated with the external IP address we specified earlier is 172.16.0.2. So to expose the deployment we simply need to type.

Note: Here 172.16.0.2 is an IP address that is routable within the tenant network created by ciao for the Kubernetes cluster. The IP 198.51.100.2 is the IP address outside of the isolated tenant network with which, the internal IP 172.16.0.2 can be accessed. In a normal setup the IP 198.51.100.2 is either routable at the data center level, or in some cases may be exposed to the internet.

```
$ kubectl expose deployment nginx --external-ip=172.16.0.2
service "nginx" exposed
```

The nginx service should now be accessible outside the k8s cluster via the external-ip. You can verify this using curl, e.g.,

```
$ curl 198.51.100.2
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
```

## Tearing Down the Clusters

Kubicle created k8s clusters can be deleted with the kubicle delete command. This could of course be done manually, but it would be tedious to do so, particularly with large clusters. Instead use kubicle delete which deletes all the ciao objects (instances, volumes, workloads, pools and external-ips) created to support the k8s cluster in the correct order. For example, to delete the cluster created above simply type.

```
$ kubicle delete
External-ips deleted:
198.51.100.2

Pools Deleted:
K8s-pool-46959cfe-f584-45f1-9218-50ea3549a0ee

Workloads deleted:
a92ce5be-922d-445a-ab3c-8f8a42e043ac
ead96d3d-b23f-4c43-9921-4756d274e51d

Instances deleted:
f6c494d1-e4b4-4c26-a663-0dab4dfb15db
38ec8bd5-3c50-477f-9b59-b8b332609551
```

To delete the entire ciao-down VM simply log out of the VM and type

```
$ ciao-down delete
```

on your host.

## Troubleshooting

The ciao-down create command can sometimes fail.  The most common causes of this failure are discussed below.

### Failure to Access /dev/kvm

One cause of failure is that the user running ciao-down does not have the required permissions
to access /dev/kvm.  If this is the case you will see and error message similar to the one
below.

```
Booting VM with 6 GB RAM and 2 cpus
Failed to launch qemu : exit status 1, Could not access KVM kernel module: Permission denied
failed to initialize KVM: Permission denied
```

You can resolve this problem by adding yourself to the group of users permitted to access this
device.  Assuming that this group is called kvm you would execute

```
sudo gpasswd -a $USER kvm
```

### Port Conflict

 By default, ciao-down map a port on your computer's network interface to a port
 on the VM it creates. One of these ports, 10022, is used for SSH access. This
 port mapping is necessary to access these services in the VM from the host. For
 example, the ciao-down connect command is implementing by executing

```
ssh -q -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o IdentitiesOnly=yes -i /home/$USER/.ciao-down/id_rsa 127.0.0.1 -p 10022
```

on the host.  There's a potential problem here.  If either of these ports are already taken by some other service on
your computer, ciao-down create will fail, e.g.,

```
Installing host dependencies
OS Detected: ubuntu
Downloading Ubuntu 16.04
Booting VM with 6 GB RAM and 2 cpus
Failed to launch qemu : exit status 1, qemu-system-x86_64: -net user,hostfwd=tcp::10022-:22: could not set up host forwarding rule 'tcp::10022-:22'
qemu-system-x86_64: -net user,hostfwd=tcp::10022-:22: Device 'user' could not be initialized

```

Here we can see that port 10022 is already taken.  Going forward we will modify ciao-down to dynamically select available host ports.  In the meantime however, we can work around this problem by overriding the default ports on the command line, as follows:

```
ciao-down create -mem=6 -cpus=2 -port "10023-22" ciao
```

## Closing Remarks

I hope you’ve enjoyed this short overview of ciao and kubicle.   If you have any questions or comments please contact the ciao team via our [mailing list](https://lists.clearlinux.org/mailman/listinfo/ciao-devel) and on IRC
in channel #ciao-project on [Freenode](https://freenode.net/kb/answer/chat).
