#!/bin/bash

sudo systemctl disable ciao-controller
sudo systemctl disable ciao-scheduler
sudo systemctl stop ciao-controller
sudo systemctl stop ciao-scheduler
sudo rm /etc/systemd/system/ciao*
sudo rbd ls -l | cut -f 1 -d " " | grep ciao-image | xargs -n 1 sudo rbd snap unprotect  -f
sudo rbd ls | xargs -n 1 sudo rbd snap purge
sudo rbd ls | xargs -n 1 sudo rbd rm
sudo rm -rf /var/lib/ciao/data/
sudo rm -f /usr/local/bin/ciao*
sudo rm -rf /etc/ciao
sudo rm /etc/pki/ciao/cert-*.pem
sudo rm /etc/pki/ciao/CAcert.pem
