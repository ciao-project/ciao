# ciao-cli

Command-line interface for the Cloud Integrated Advanced Orchestrator
(CIAO).

ciao-cli sends HTTPS requests to the [Ciao controller](https://github.com/01org/ciao/tree/master/ciao-controller)
compute API [endpoints](https://github.com/01org/ciao/blob/master/ciao-controller/compute.go).

## Usage

```shell
ciao-cli: Command-line interface for the Cloud Integrated Advanced Orchestrator (CIAO)                                    [8/381]

Usage:

        ciao-cli [options] command sub-command [flags]

The options are:

  -alsologtostderr
        log to standard error as well as files
  -ca-file string
        CA Certificate
  -computeport int
        Openstack Compute API port (default 8774)
  -controller string
        Controller URL
  -identity string
        Keystone URL
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -password string
        Openstack Service Password
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -tenant-id string
        Tenant UUID
  -tenant-name string
        Tenant name
  -username string
        Openstack Service Username
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging


The commands are:

        event
        instance
        node
        tenant
        trace
        workload

Use "ciao-cli command -help" for more information about that command.
```

## Ciao environment variables

ciao-cli first looks for Ciao specific environment variables to retrieve
credentials and networking information:

* `CIAO_CONTROLLER` exports the Ciao controller URL
* `CIAO_IDENTITY` exports the Ciao keystone instance URL
* `CIAO_COMPUTEPORT` exports the Ciao compute alternative port
* `CIAO_USERNAME` exports the Ciao username
* `CIAO_PASSWORD` export the Ciao password for `CIAO_USERNAME`
* `CIAO_TENANT_NAME` export the Ciao tenant name for `CIAO_USERNAME`
* `CIAO_CA_CERT_FILE` (optional) use the supplied certificate as the CA

All those environment variables can be set through an rc file.
For example:

```shell
$ cat ciao-cli-example.sh

export CIAO_CONTROLLER=ciao-ctl.intel.com
export CIAO_IDENTITY=https://ciao-identity.intel.com:35357
export CIAO_USERNAME=user
export CIAO_PASSWORD=ciaouser
export CIAO_TENANT_NAME=admin
```

Exporting those variables is not compulsory and they can be defined
or overridden from the `ciao-cli` command line.

## Keystone certificates

ciao-cli interacts with the CIAO keystone instance over HTTPS.  As such you
will need to have the keystone CA certificate available in order to make
requests. You can either install the CA certificate system-wide:

* On Fedora:
```shell
sudo cp keystone_ca_cert.pem /etc/pki/ca-trust/source/anchors/
sudo update-ca-trust
```

* On Ubuntu
```shell
sudo cp keystone_ca_cert.pem /usr/local/share/ca-certificates/keystone.crt
sudo update-ca-certificates
```

Or, alternatively the CA certificate may be specified with the `-ca-file`
command line or with the `CIAO_CA_CERT_FILE` environment variable.

## Priviledged versus non priviledged CIAO users

Administrators of a CIAO cluster are privileged users that are part of the
`admin` or `service` projects. They are allowed to run each and every
ciao-cli commands.
Some ciao-cli commands are privileged and can only be run by administrators.

Non privileged commands can be run by all users. Administrators will have to specify
a tenant/project UUID through the -tenant option in order to specify against which
CIAO tenant/project they're running the command:
```shell
$GOBIN/ciao-cli -password ciao -username admin instance list -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

Non privileged users belonging to several tenants/projects will also have to specify
a tenant/project UUID or name through either the -tenant-id or -tenant-name options
in order to specify against which CIAO tenant/project they're running the command:

```shell
$GOBIN/ciao-cli -username user -password ciaouser -tenant-name project1 instance list
```

Non privileged users belonging to only one single tenant/project do not need to
pass the tenant/project UUID or name when running non privileged commands:

```shell
$GOBIN/ciao-cli -username user -password ciaouser instance list
```


## Examples

Let's assume we're running a Ciao cluster with the following settings:

* The Ciao controller is running at `ciao-ctl.intel.com`
* The Keystone server is running at `https://ciao-identity.intel.com:35357`
* The `admin` user is part of the `admin` project
* The admin password is `ciao`
* The `user` user is part of only one project: `project1`
* The password for `user` is `ciaouser`
* `project1` UUID is `68a76514-5c8e-40a8-8c9e-0570a11d035b`
*

This can be defined through the following Ciao rc file:

```shell
$ cat ciao-cli-example.sh

export CIAO_CONTROLLER=ciao-ctl.intel.com
export CIAO_IDENTITY=https://ciao-identity.intel.com:35357
export CIAO_USERNAME=user
export CIAO_PASSWORD=ciaouser
export CIAO_TENANT_NAME=project1
```

### Cluster status (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao node status
```

### List all compute nodes (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao node list -compute
```

### List all CNCIs (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao node list -cnci
```

### List all tenants/projects (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao tenant list -all
```

### List quotas

```shell
$GOBIN/ciao-cli tenant list -quotas
```

### List consumed resources

```shell
$GOBIN/ciao-cli tenant list -resources
```

### List all instances

```shell
$GOBIN/ciao-cli instance list
```

### List at most the first 10 instances

```shell
$GOBIN/ciao-cli instance list -limit 10
```

### List at most the 20 instances starting from instance number 10

```shell
$GOBIN/ciao-cli instance list -limit 20 -offset 10
```

### List all workloads

```shell
$GOBIN/ciao-cli workload list
```

### Launch a new instance

```shell
$GOBIN/ciao-cli instance add -workload 69e84267-ed01-4738-b15f-b47de06b62e7
```

### Launch 1000 new instances

```shell
$GOBIN/ciao-cli instance add -workload 69e84267-ed01-4738-b15f-b47de06b62e7 -instances 1000
```

### Launch 1000 instances and trace them

```shell
$GOBIN/ciao-cli instance add -workload 69e84267-ed01-4738-b15f-b47de06b62e7 -label start_trace_20160415 -instances 1000
```

### Stop a running instance

```shell
$GOBIN/ciao-cli instance stop -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Restart a stopped instance

```shell
$GOBIN/ciao-cli instance restart -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete an instance

```shell
$GOBIN/ciao-cli instance delete -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete all instances for a given tenant

```shell
$GOBIN/ciao-cli instance delete -all
```

### List all available trace labels (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao trace list
```

### Dump and display trace data from a given trace label (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao trace show -label start_trace_20160415
```

### List all cluster events (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao event list -all
```

### List all cluster events for a given tenant

```shell
$GOBIN/ciao-cli event list
```
