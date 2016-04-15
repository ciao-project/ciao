# ciao-cli

Command-line interface for the Cloud Integrated Advanced Orchestrator
(CIAO).

ciao-cli sends HTTPS requests to the [Ciao controller](https://github.com/01org/ciao/tree/master/ciao-controller)
compute API [endpoints](https://github.com/01org/ciao/blob/master/ciao-controller/compute.go).

## Usage

```shell
Usage of ciao-cli:
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
  -dump-tenant-id
    	Dump tenant UUID
  -dump-token
    	Dump keystone tokens
  -identity string
    	Keystone URL
  -instance string
    	Instance UUID
  -instance-marker string
    	Show instance list starting from the next instance after instance-marker
  -instance-offset int
    	Show instance list starting from instance #instance-offset
  -instances int
    	Number of instances to create (default 1)
  -launch-instances
    	Launch Ciao instances
  -list-cncis
    	List all CNCIs
  -list-cns
    	List all compute nodes
  -list-instances
    	List all instances for a tenant or for a compute node
  -list-length int
    	Maximum number of items in the reponse
  -list-quotas
    	List quotas status for a tenant
  -list-resources
    	List consumed resources for a tenant for the past 15mn
  -list-tenants
    	List all tenants
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
  -scheme value
    	Compute API URL scheme (http or https) (default https)
  -scope string
    	Scope tenant name (default "service")
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -stop-instance
    	Stop a Ciao instance
  -tenant string
    	Tenant UUID
  -username string
    	Openstack Service Username (default "nova")
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
  -workload string
    	Workload UUID
```

## Examples

Let's assume we're running a Ciao cluster with the following settings:

* The Ciao controller is running at `ciao-ctl.intel.com`
* The Keystone server is running at `https://ciao-identity.intel.com:35357`
* The `admin` user is part of the `admin` project
* The admin password is `ciao`
* One tenant/project UUID is `68a76514-5c8e-40a8-8c9e-0570a11d035b`

### Cluster status

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -cluster-status
```

### List all compute nodes

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-cns
```

### List all CNCIs

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-cncis
```

### List all tenants/projects

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-tenants
```

### List quotas per tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-quotas -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List consumed resources per tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-resources -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List all instances for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-instances -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List at most the first 10 instances for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-instances -list-length 10 -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List at most the 20 instances starting from instance number 10 for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-instances -list-length 20 -instance-offset 10 -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List all workloads for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-workloads -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### Launch a new instance

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -launch-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -workload 69e84267-ed01-4738-b15f-b47de06b62e7
```

### Launch 1000 new instances

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -launch-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -workload 69e84267-ed01-4738-b15f-b47de06b62e7 -instances 1000
```

### Stop a running instance

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -stop-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Restart a stopped instance

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -restart-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete an instance

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -delete-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -instance 4c46ace5-cf92-4ce5-a0ac-68f6d524f8aa
```

### Delete all instances for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -delete-instance -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b -all-instances
```



