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
* The admin password is "ciao"

### Cluster status

```bash
ciao-cli -password ciao -username admin -scope admin -identity https://ciao-identity.intel.com:35357 -controller ciao-ctl.intel.com -cluster-status
```
