#!/bin/bash

NODE_ENV=$MODE
CONTROLLER_HOST=$CONTROLLER_HOST
IDENTITY_HOST=$IDENTITY_HOST
CERT_PASS=$CERT_PASS

_usage(){
    >&2 echo "usage:"
    >&2 echo "    $ docker run -v /path/to/ciao-webui-key.pem:/etc/pki/ciao-webui-key.pem \\"
    >&2 echo "                 -v /path/to/ciao-webui-cert.pem:/etc/pki/ciao-webui-cert.pem \\"
    >&2 echo "                 -e CONTROLLER_HOST=controller.example.com \\"
    >&2 echo "                 -e IDENTITY_HOST=keystone.example.com \\"
    >&2 echo "(optional:       -e CERT_PASS=certificate_passphrase )"
}

if [[ -z $MODE ]] ; then
    NODE_ENV="production"
fi

if [[ -z $CONTROLLER_HOST || -z $IDENTITY_HOST ]] ; then
    >&2 echo -e "missing CONTROLLER_HOST or IDENTITY_HOST arguments\n"
    _usage
    exit
fi

if [[ ! -r /etc/pki/ciao-webui-key.pem ]] ; then
    >&2 echo -e "missing /etc/pki/ciao-webui-key.pem\n"
    _usage
    exit
fi

if [[ ! -r /etc/pki/ciao-webui-cert.pem ]] ; then
    >&2 echo -e "missing /etc/pki/ciao-webui-cert.pem\n"
    _usage
    exit
fi

sed -i.bak s/##MODE##/$NODE_ENV/g                   /etc/ciao-webui/ciao_config.json
sed -i.bak s/##CONTROLLER_HOST##/$CONTROLLER_HOST/g /etc/ciao-webui/ciao_config.json
sed -i.bak s/##KEYSTONE_HOST##/$IDENTITY_HOST/g     /etc/ciao-webui/ciao_config.json
sed -i.bak s/##CERT_PASS##/$CERT_PASS/g             /etc/ciao-webui/ciao_config.json

ln -sf /etc/ciao-webui/ciao_config.json /ciao-webui/config/ciao_config.json

cd /ciao-webui
export PORT=443
./deploy.sh $NODE_ENV
