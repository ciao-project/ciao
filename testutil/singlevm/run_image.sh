#!/bin/bash

if [ ! -z $1 ]; then
    sudo "$GOPATH"/bin/ciao-image --identity $1 &
else
    sudo "$GOPATH"/bin/ciao-image &
fi

