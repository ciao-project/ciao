# BAT tests

This folder contains a set of BAT tests.  Each set of tests validates a specific
part of ciao, such as storage, and is implemented by a separate go package in one of
the following sub folders.  

```
.
├── base      - Basic tests that verify that the cluster is functional
```

The tests are implemented using the go testing framework.  This is convenient
as this framework is used for ciao's unit tests and so is already familiar
to ciao developers, it requires no additional dependencies and it works with ciao's
existing test case runner, test-cases.

# A Short Guide to Running the BAT Tests

## Set up

The BAT tests require that certain environment variables have been set before they
can be run:

* "CIAO_IDENTITY" - the URL and port number of your identity service
* "CIAO_CONTROLLER" - the URL and port number of the ciao compute service
* "CIAO_USERNAME" - a test user with user level access to a test tenant
* "CIAO_PASSWORD" - your test user's password
* "CIAO_ADMIN_USERNAME" - your cluster admin user name
* "CIAO_ADMIN_PASSWORD" - your cluster admin password.

## Running all the BAT tests

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v ./...
```

## Run the BAT Tests and Generate a Pretty Report

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases ./...
```

## Run the BAT Tests and Generate TAP report

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases -format tap ./...
```

## Run the BAT Tests and Generate a Test Plan

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# test-cases -format html ./...
```

## Run a Single Set of Tests

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v github.com/01org/ciao/_release/bat/base
```

## Run a Single Test

```
# cd $GOPATH/src/github.com/01org/ciao/_release/bat
# go test -v -run TestGetAllInstances ./...
```
