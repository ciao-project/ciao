#!/bin/bash
set -x
set -e

go version

go get -t -v ./...
go get github.com/intel/ccloudvm

sudo apt-get update
sudo apt-get install qemu xorriso rsync -y

pushd ~
rm -rf .renv .nvm .phpenv .kerl .phpbrew .pyenv
popd

sudo apt-get autoremove -y
sudo apt-get clean -y

export PATH=$GOPATH/bin:$PATH
sudo chmod ugo+rwx /dev/kvm
ccloudvm create -debug ciao

myssh=`ccloudvm status | grep ssh`
mysshcmd=`echo $myssh | cut -d : -f 2`
sshrsync=`echo $mysshcmd | cut -d " " -f -12`
port=`echo $mysshcmd | cut -d " " -f 14-`
mysshrsync="$sshrsync $port"

$mysshcmd "mkdir test"
rsync -avzk -e "$mysshrsync" --progress $GOPATH/* 127.0.0.1:~/test

echo "export GOPATH=~/test && ~/test/src/github.com/ciao-project/ciao/testutil/singlevm/setup.sh" | $mysshcmd bash -l -s
echo ". ~/local/demo.sh && ~/test/src/github.com/ciao-project/ciao/testutil/singlevm/verify.sh" | $mysshcmd bash -l -s

echo "export GOPATH=~/test; cd ~/test/src/github.com/ciao-project/ciao/_release/bat; . ~/local/demo.sh && test-cases -v ./..." |$mysshcmd bash -l -s
