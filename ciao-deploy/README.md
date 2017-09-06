# Deploying a ciao cluster

## Prerequisites

Prior to deploying ciao the following prerequisites must be met:

1. Ensure that there is a user account on the system accessed by key
authentication with **passwordless** sudo access.

2. If your systems access the internet via proxies that these are correctly
configured. Docker in particular needs special configuration for network
proxies.

3. A running ceph cluster along with a ceph keyring that can access rbd
installed on all machines in the cluster. The [Ceph
Deployment](http://docs.ceph.com/docs/master/rados/deployment/) tool can be
used to both build the cluster and deploy the keyring to all nodes.

## Setting up the master node of the cluster

All deployment is driven from the master node of the cluster. On this machine
you will need to install Go 1.8. Instructions for doing this can be found [on
the Go website](https://www.golang.org)

Next download, install and build ciao:

```
go get -u github.com/01org/ciao/...
```

Please ensure you have `$GOPATH/bin` in your `$PATH`. If you have not specified
a `$GOPATH` this will default to `$HOME/go`

Now setup the master node using `ciao-deploy`. The parameters below are the
minimum that you will need to specify to have a working setup.

`--https-cert and --https-ca-cert` are the private and public keychains needed
to operate the HTTPS server in Ciao.
.

```
ciao-deploy setup \
	--https-cert=/etc/pki/ciao/controller_key.pem \
	--https-ca-cert=/etc/pki/ciao/controller_cert.pem \
```

In your setup you may need to change other flags. `ciao-deploy setup --help`
provides detailed information on all the flags.

Upon successful completion this tool will report the environment variables that
can be used to access the cluster. For example:

```
Environment variables to access cluster:

export CIAO_CONTROLLER="ciao-ctl"
export CIAO_ADMIN_CLIENT_CERT_FILE="/etc/pki/ciao/auth-admin.pem"
```

Make a note of these as we will use them to test the cluster.

## Joining nodes to the cluster

Provisioning nodes into the cluster requires key driven access to the nodes.
You will need to SSH into each node and check its host key as the
`.ssh/known_hosts` file is relied upon for trusting nodes. You will also need
to run `ssh-agent` and ensure that the key required to access the nodes is
added.

You will need to join two types of node to the cluster. Network nodes are
special nodes that are used to host network concentrators and used for cluster
ingress and egress.

Add a network node to the cluster with a command like the following. Where you
have replaced ciao-nn with your network node hostname:

```
ciao-deploy join --network ciao-nn
```

Next add all your compute nodes. The join command can take multiple hostnames
and they will be configured in parallel.

```
ciao-deploy join ciao-cn01 ciao-cn02 ciao-cn03 
```

## Validating that the cluster is functioning

The easiest way to validate the the cluster is functioning as expected is to
create a test instance. The deployment tool has a built-in tool that can
populate the cluster with some sample images and workloads.

```
ciao-deploy create-bat-workloads
```

Will download the image, upload to the cluster and create the workloads. You
will need both `CIAO_ADMIN_CLIENT_CERT_FILE` and `CIAO_CLIENT_CERT_FILE` set.
The latter with a non-admin user that you've added with `ciao-deploy auth create`

Once workloads are created you can create a new instance and experiment with
the ciao cluster.

```
ciao-cli instance add -workload <workload id>
```

## Updating the cluster

To update to a newer version ciao. Download and compile it as above and then
run the following commands.

To update the master use:

```
ciao-deploy update
```

To update the nodes run the same command as to join them:

```
ciao-deploy join ciao-cn01 ciao-cn02 ciao-cn03 
```

## Teardown

To remove ciao from the worker nodes in the cluster `ciao-deploy unjoin` can be
used.

```
ciao-deploy unjoin ciao-nn ciao-cn01 ciao-cn02 ciao-cn03
```

The same command can be used for both network and compute nodes without
differentiation.

Teardown of the master node is not currently supported.

## Support 

The ciao development team can be reached via our [mailing
list](https://lists.clearlinux.org/mailman/listinfo/ciao-devel) and on IRC
in channel #ciao-project on [Freenode](https://freenode.net/kb/answer/chat).
