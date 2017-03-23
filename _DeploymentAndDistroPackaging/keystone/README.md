Keystone with SSL
=================
[![](https://images.microbadger.com/badges/image/ciaoproject/keystone.svg)](http://microbadger.com/images/clearlinux/keystone "Get your own image badge on microbadger.com")
[![](https://images.microbadger.com/badges/version/ciaoproject/keystone.svg)](http://microbadger.com/images/clearlinux/keystone "Get your own version badge on microbadger.com")

This provides a SSL-enabled Keystone docker container

Build
-----
```
docker build -t ciaoproject/keystone .
```

Or just pull it from Dockerhub
---------------------------
```
docker pull ciaoproject/keystone
```

Create Keystone SSL certificates
--------------------------------
```
YOUR_HOST=`hostname -f`
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout keystone_key.pem \
	-out keystone_cert.pem -subj "/CN=$YOUR_HOST"
```

Start Keystone container
------------------------
```
YOUR_HOST=`hostname -f`
MYSQL_DATA_DIR=/var/lib/mysql/
docker run -d -it --name keystone -p 5000:5000 -p 35357:35357 \
       -e IDENTITY_HOST="$YOUR_HOST" \
       -e KEYSTONE_ADMIN_PASSWORD="secret" \
       -v $MYSQL_DATA_DIR:/var/lib/mysql \
       -v `pwd`/keystone_cert.pem:/etc/nginx/ssl/keystone_cert.pem \
       -v `pwd`/keystone_key.pem:/etc/nginx/ssl/keystone_key.pem \
       ciaoproject/keystone
```

Login into Keystone container
------------------------------
```
docker exec -it  keystone bash
# Inside the container
root@26bd2b8a8a60 /root # source openrc
openstack user list
+----------------------------------+-------+
| ID                               | Name  |
+----------------------------------+-------+
| 24620586335a473fb56fc2be2f6bfb53 | admin |
+----------------------------------+-------+
```

Environment Variables
---------------------
- ``IDENTITY_HOST``
  Identity (Keystone) host
- (Optional) ``KEYSTONE_ADMIN_PASSWORD``
  Keystone Admin user password. If you don't provide this variable, a random password will be created.

Default build args in Docker are on: https://docs.docker.com/engine/reference/builder/#/arg
