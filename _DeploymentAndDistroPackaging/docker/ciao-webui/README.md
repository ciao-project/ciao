Ciao webui
==========
This provides a ciao web dashboard container

Build
-----
```
    docker build -t clearlinux/ciao-webui .
```

Or just pull it from Dockerhub
------------------------------
```
    docker pull clearlinux/ciao-webui
```
Create Ciao-webui SSL certificates
----------------------------------
```
    YOUR_HOST=`hostname -f`

    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout webui_key.pem \
                -out webui_cert.pem -subj "/CN=$YOUR_HOST"
```

Run the Ciao-webui Container
----------------------------
```
    docker run -v `pwd`/webui_key.pem:/etc/pki/ciao-webui-key.pem \
               -v `pwd`/webui_cert.pem:/etc/pki/ciao-webui-cert.pem \
               -e CONTROLLER_HOST=controller.example.com \
               -e IDENTITY_HOST=keystone.example.com \
               -p 443:443 -d clearlinux/ciao-webui
```

Environment Variables
---------------------
- ``IDENTITY_HOST``
  Identity (Keystone) host
- ``CONTROLLER_HOST``
  Controller (ciao-controller) host
- ``CERT_PASS``
  Passprhase for the ciao-webui certificate
- ``MODE``
  Ciao-webui deployment mode (production|development)
  default is `production`

Extra Build ARGs
----------------
- ``swupd_args`` Specifies [SWUPD](https://clearlinux.org/documentation/swupdate_how_to_run_the_updater.html) flags

Default build args in Docker are on: https://docs.docker.com/engine/reference/builder/#/arg
