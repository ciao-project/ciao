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