# ciao-cli

Command-line interface for the Cloud Integrated Advanced Orchestrator
(CIAO).

ciao-cli sends HTTPS requests to the [Ciao controller](https://github.com/01org/ciao/tree/master/ciao-controller)
compute API [endpoints](https://github.com/01org/ciao/blob/master/ciao-controller/compute.go).

## Usage

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

### List all instances for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-instances -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
```

### List at most the first 10 instances for a given tenant

```shell
$GOBIN/ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -list-instances -list-length 10 -tenant 68a76514-5c8e-40a8-8c9e-0570a11d035b
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



