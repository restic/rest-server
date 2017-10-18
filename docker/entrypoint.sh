#!/bin/sh

set -e

if [ -z "$DISABLE_AUTHENTICATION" ]; then
    if [ ! -f "$PASSWORD_FILE" ]; then
        touch "$PASSWORD_FILE"
    fi

    if [ ! -s "$PASSWORD_FILE" ]; then
        echo
        echo "**WARNING** No user exists, please 'docker exec -it \$CONTAINER_ID create_user'"
        echo
    fi
else
    rm -f "$PASSWORD_FILE"
fi

exec rest-server --path "$DATA_DIRECTORY" $OPTIONS
