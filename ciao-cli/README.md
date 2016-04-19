# ciao-cli

Command-line interface for the Cloud Integrated Advanced Orchestrator
(CIAO).

ciao-cli sends HTTPS requests to the [Ciao controller](https://github.com/01org/ciao/tree/master/ciao-controller)
compute API [endpoints](https://github.com/01org/ciao/blob/master/ciao-controller/compute.go).

## Usage

```shell
Usage of ./ciao-cli:
  -all-instances
    	Select all instances
  -alsologtostderr
    	log to standard error as well as files
  -cluster-status
    	List all compute nodes
  -cn string
    	Compute node UUID
  -cnci string
    	CNCI UUID
  -computeport int
    	Openstack Compute API port (default 8774)
  -controller string
    	Controller URL (default "localhost")
  -delete-instance
    	Delete a Ciao instance
  -dump-cnci
    	Dump a CNCI details
  -dump-label string
    	Dump all trace data for a given label
  -dump-tenant-id
    	Dump tenant UUID
  -dump-token
    	Dump keystone tokens
  -identity string
    	Keystone URL
  -instance string
    	Instance UUID
  -instance-label string
    	Set a frame label. This will trigger frame tracing
  -instance-marker string
    	Show instance list starting from the next instance after instance-marker
  -instance-offset int
    	Show instance list starting from instance #instance-offset
  -instances int
    	Number of instances to create (default 1)
  -launch-instances
    	Launch Ciao instances
  -list-cn-instances
    	List all instances for a compute node
  -list-cncis
    	List all CNCIs
  -list-cns
    	List all compute nodes
  -list-instances
    	List all instances for a tenant
  -list-labels
    	List all trace labels
  -list-length int
    	Maximum number of items in the reponse
  -list-quotas
    	List quotas status for a tenant
  -list-resources
    	List consumed resources for a tenant for the past 15mn
  -list-tenants
    	List all tenants
  -list-wl-instances
    	List all instances for a workload
  -list-workloads
    	List all workloads
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace (default :0)
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -password string
    	Openstack Service Username (default "nova")
  -restart-instance
    	Restart a Ciao instance
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -stop-instance
    	Stop a Ciao instance
  -tenant-id string
    	Tenant UUID
  -tenant-name string
    	Tenant name
  -username string
    	Openstack Service Username (default "nova")
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -workload string
    	Workload UUID
```

## Ciao environment variables

ciao-cli first look for Ciao specific environment variables to retrieve
credentials and networking information:

* `CIAO_CONTROLLER` exports the Ciao controller URL
* `CIAO_IDENTITY` exports the Ciao keystone instance URL
* `CIAO_COMPUTEPORT` exports the Ciao compute alternative port
* `CIAO_USERNAME` exports the Ciao username
* `CIAO_PASSWORD` export the Ciao password for `CIAO_USERNAME`

All those environmant variables can be set through an rc file.
For example:

```shell
$ cat ciao-cli-example.sh

export CIAO_CONTROLLER=ciao-ctl.intel.com
export CIAO_IDENTITY=https://ciao-identity.intel.com:35357
export CIAO_USERNAME=user
export CIAO_PASSWORD=ciaouser
```

Exporting those variables is not compulsory and they can be defined
or overridden from the `ciao-cli` command line.

## Keystone certificates

ciao-cli interact with the CIAO keystone instance over HTTPS.
As such you will have to install the keystone CA certificate locally
in order for ciao-cli to verify the keystone identity.

CA certificate installation is a distribution specific process. For example:

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

## Priviledged versus non priviledged CIAO users

Administrators of a CIAO cluster are priviledged users that are part of the
`admin` or `service` projects. They are allowed to run each and every
ciao-cli commands.
Some ciao-cli commands are privileged and can only be run by administrators.

Non priviledged commands can be run by all users. Administrators will have to specify
a tenant/project UUID through the -tenant-id option in order to specify against which
CIAO tenant/project they're running the command:
```shell
$GOBIN/ciao-cli -password ciao -username admin -list-instances -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

Non priviledged users belonging to several tenants/projects will also have to specify
a tenant/project UUID or name through either the -tenant-id or -tenant-name options
in order to specify against which CIAO tenant/project they're running the command:

```shell
$GOBIN/ciao-cli -username user -password ciaouser -list-instances -tenant-name project1
```

Non priviledged users belonging to only one single tenant/project do not need to
pass the tenant/project UUID or name when running non priviledged commands:

```shell
$GOBIN/ciao-cli -username user -password ciaouser -list-instances
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

This can be defined through the following Ciao rc file:

```shell
$ cat ciao-cli-example.sh

export CIAO_CONTROLLER=ciao-ctl.intel.com
export CIAO_IDENTITY=https://ciao-identity.intel.com:35357
export CIAO_USERNAME=user
export CIAO_PASSWORD=ciaouser
```

### Cluster status (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -cluster-status
```

### List all compute nodes (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -list-cns
```

### List all CNCIs (Privileged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -list-cncis
```

### List all tenants/projects (Priviledged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -list-tenants
```

### List quotas

```shell
$GOBIN/ciao-cli -username user -password ciaouser -list-quotas
```

### List consumed resources

```shell
$GOBIN/ciao-cli -list-resources
```

### List all instances

```shell
$GOBIN/ciao-cli -list-instances
```

### List at most the first 10 instances

```shell
$GOBIN/ciao-cli -list-instances -list-length 10
```

### List at most the 20 instances starting from instance number 10

```shell
$GOBIN/ciao-cli -list-instances -list-length 20 -instance-offset 10
```

### List all workloads

```shell
$GOBIN/ciao-cli -list-workloads
```

### Launch a new instance

```shell
$GOBIN/ciao-cli -launch-instances -workload 69e84267-ed01-4738-b15f-b47de06b62e7
```

### Launch 1000 new instances

```shell
$GOBIN/ciao-cli -launch-instances -workload 69e84267-ed01-4738-b15f-b47de06b62e7 -instances 1000
```

### Launch 1000 instances and trace them

```shell
$GOBIN/ciao-cli -launch-instances -workload 69e84267-ed01-4738-b15f-b47de06b62e7 -instance-label start_trace_20160415
```

### Stop a running instance

```shell
$GOBIN/ciao-cli -stop-instance -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Restart a stopped instance

```shell
$GOBIN/ciao-cli -restart-instance -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete an instance

```shell
$GOBIN/ciao-cli -delete-instance -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete all instances for a given tenant

```shell
$GOBIN/ciao-cli -delete-instance -all-instances
```

### List all available trace labels (Priviledged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -list-labels
```

### Dump and display trace data from a given trace label (Priviledged)

```shell
$GOBIN/ciao-cli -username admin -password ciao -dump-label -instance-label start_trace_20160415
```



