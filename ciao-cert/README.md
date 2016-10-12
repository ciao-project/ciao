# ciao-cert

ciao-cert is a command line tool for generating [SSNTP](https://github.com/01org/ciao/tree/master/ssntp)
specific certificates. In particular it includes SSNTP roles as part of
the certificate extended key attribute, which is an SSNTP requirement.

ciao-cert generates generates PEM files containing self signed certificates
and private keys for all SSNTP roles.

## Usage

```shell
Usage of ciao-cert:
  -alsologtostderr
        log to standard error as well as files
  -directory string
        Installation directory (default ".")
  -dump string
        Print details about provided certificate
  -elliptic-key
        Use elliptic curve algorithms
  -email string
        Certificate email address (default "ciao-devel@lists.clearlinux.org")
  -host string
        Comma-separated hostnames to generate a certificate for
  -ip string
        Comma-separated IPs to generate a certificate for
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -organization string
        Certificates organization
  -role value
        Comma separated list of SSNTP role [agent, scheduler, controller, netagent, server, cnciagent]
  -server
        Whether this cert should be a server one
  -server-cert string
        Server certificate for signing a client one
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -v value
        log level for V logs
  -verify
        Verify client certificate
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

## Example

On our example cluster the scheduler is running on ciao-ctl.intel.com.
ciao.ctl.intel.com is a multi homed machine connected to the cluster
control plane through 192.168.1.118

We need to generate 5 private keys for the Scheduler, Controller, Networking Agent,
CNCI Agent and the Compute Node Agents. We also need to generate the CA certificate.

* Scheduler private key and CA certificate

```shell
$GOBIN/ciao-cert -server -role scheduler -email=ciao-devel@lists.clearlinux.org -organization=Intel -ip=192.168.1.118 -host=ciao-ctl.intel.com -verify
```

That will generate `CAcert-ciao-ctl.intel.com.pem` and `cert-Scheduler-ciao.ctl.intel.com.pem`.

* Compute Node Agent private key

```$GOBIN/ciao-cert -role agent -server-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify```

That will generate `cert-CNAgent-localhost.pem`.

* Networking Node Agent private key

```$GOBIN/ciao-cert -role netagent -server-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify```

That will generate `cert-NetworkingAgent-localhost.pem`.

* CNCI Agent private key

```$GOBIN/ciao-cert -role cnciagent -server-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify```

That will generate `cert-CNCIAgent-localhost.pem`.

* Controller private key

```$GOBIN/ciao-cert -role controller -server-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify```

That will generate `cert-Controller-localhost.pem`.

## Multi roles support

In some cases SSNTP clients or servers want to support
several roles at the same time and SSNTP supports that feature.

But certificates need to be generated accordingly, by passing a
comma separated list of roles to ciao-cert.
For example launcher may want to expose both the CN and NN agent roles:

```$GOBIN/ciao-cert -role agent,netagent -server-cert cert-Scheduler-ciao-ctl.intel.com.pem -email=ciao-devel@lists.clearlinux.org -organization=Intel -host=localhost -verify```

## Inspecting certificates

It is possible to have `ciao-cert` provide some information about the generated
certificates; this is done by using the `-dump` command line flag. Here is some
sample output from the scheduler certificate generated above:

```
$GOBIN/ciao-cert -dump ./cert-Scheduler-ciao-ctl.intel.com.pem
Certificate:    ./cert-Scheduler-ciao-ctl.intel.com.pem
Organization:   Intel
Is CA:          true
Validity:       2016-10-12 15:19:39 +0000 UTC to 2017-10-12 15:19:39 +0000 UTC
For role:       Scheduler-
For host:       ciao-ctl.intel.com
For IP:         192.168.1.118
Private key:    RSA PRIVATE KEY
```
