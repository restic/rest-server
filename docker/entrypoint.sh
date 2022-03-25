#!/bin/sh

set -e

if [ -n "$DISABLE_AUTHENTICATION" ]; then
    OPTIONS="--no-auth $OPTIONS"
else
    if [ ! -f "$PASSWORD_FILE" ]; then
        touch "$PASSWORD_FILE"
    fi

    if [ ! -s "$PASSWORD_FILE" ]; then
        echo
        echo "**WARNING** No user exists, please 'docker exec -it \$CONTAINER_ID create_user'"
        echo
    fi
fi

exec rest-server --path "$DATA_DIRECTORY" --htpasswd-file "$PASSWORD_FILE" $OPTIONS
