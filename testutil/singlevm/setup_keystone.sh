# Generate the post-keystone script to issue singlevm-specific openstack
# commands
( cat <<-EOF
#!/bin/bash

# Create basic services, users, and projects/tenants
openstack service create --name ciao compute
openstack user create --password "$ciao_password" "$ciao_username"
openstack role add --project service --user "$ciao_username" admin
openstack user create --password "$ciao_demo_password" "$ciao_demo_username"
openstack project show demo
if [[ \$? == 1 ]]; then
    openstack project create --domain default demo
fi
openstack role add --project demo --user "$ciao_demo_username" user

# Create image service endpoints
openstack service create --name glance --description "Image Service" image
openstack endpoint create --region RegionOne image public   https://$ciao_host:9292
openstack endpoint create --region RegionOne image internal https://$ciao_host:9292
openstack endpoint create --region RegionOne image admin    https://$ciao_host:9292

# admin should only be admin of the admin project. This role was created by the
# keystone container's bootstrap.
openstack role remove --project service --user admin admin

# Create storage endpoints
openstack service create --name cinderv2 --description "Volume Service" volumev2
openstack endpoint create --region RegionOne volumev2 public   'https://$ciao_host:8776/v2/%(tenant_id)s'
openstack endpoint create --region RegionOne volumev2 internal 'https://$ciao_host:8776/v2/%(tenant_id)s'
openstack endpoint create --region RegionOne volumev2 admin    'https://$ciao_host:8776/v2/%(tenant_id)s'

EOF
) > "$ciao_bin"/post-keystone.sh
chmod 755 "$ciao_bin"/post-keystone.sh

## Install keystone
sudo docker run -d -it --name keystone \
    --add-host="$ciao_host":"$ciao_ip" \
    -p $keystone_public_port:5000 \
    -p $keystone_admin_port:35357 \
    -e IDENTITY_HOST="$ciao_host" -e KEYSTONE_ADMIN_PASSWORD="${OS_PASSWORD}" \
    -v "$ciao_bin"/post-keystone.sh:/usr/bin/post-keystone.sh \
    -v $mysql_data_dir:/var/lib/mysql \
    -v "$keystone_cert":/etc/nginx/ssl/keystone_cert.pem \
    -v "$keystone_key":/etc/nginx/ssl/keystone_key.pem clearlinux/keystone:stable

echo -n "Waiting up to $keystone_wait_time seconds for keystone identity" \
    "service to become available"
try_until=$(($(date +%s) + $keystone_wait_time))
while : ; do
    while [ $(date +%s) -le $try_until ]; do
        # The keystone container tails the log at the end of its
        # initialization script
        if docker exec keystone pidof tail > /dev/null 2>&1; then
            echo READY
            break 2
        else
            echo -n .
            sleep 1
        fi
    done
    echo FAILED
    break
done