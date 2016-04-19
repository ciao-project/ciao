#!/bin/bash
sudo ./snnetcli create bridge br00
sudo ./snnetcli show br00
sudo ./snnetcli enable bridge br00
sudo ./snnetcli disable bridge br00
sudo ./snnetcli destroy bridge br00
sudo ./snnetcli create vnic vnic00
sudo ./snnetcli show vnic00
sudo ./snnetcli enable vnic vnic00
sudo ./snnetcli disable vnic vnic00
sudo ./snnetcli destroy vnic vnic00
sudo ./snnetcli create bridge br00
sudo ./snnetcli create vnic vnic00
sudo ./snnetcli attach br00 vnic00
brctl show
ip link
sudo ./snnetcli detach br00 vnic00
brctl show
ip link
sudo ./snnetcli destroy bridge br00
sudo ./snnetcli destroy vnic vnic00
ip link
