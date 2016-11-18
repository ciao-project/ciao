#!/bin/bash
ciao_host=$(hostname)
ciao_data="$HOME/local/data"

sudo rm $ciao_data/*

sudo "$GOPATH"/bin/ciao-controller --cacert=./CAcert-"$ciao_host".pem --cert=./cert-Controller-"$ciao_host".pem \
     --tables_init_path=$ciao_data --workloads_path=$ciao_data --database_path=$ciao_data/ciao-controller.db \
     --image_database_path=$ciao_data/ciao-image.db --stats_path=$ciao_data/ciao-controller-stats.db --single --v 3 &
