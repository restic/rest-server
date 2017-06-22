#!/bin/sh

set -e

mkdir -p $DATA_DIRECTORY

if [ -z "$DISABLE_AUTHENTICATION" ]; then
    if [ ! -f $PASSWORD_FILE ]; then
        touch $PASSWORD_FILE
    fi

    if [ ! -s $PASSWORD_FILE ]; then
        echo
        echo "**WARNING** No user exists, please 'docker exec -ti \$CONTAINER_ID create_user'"
        echo
    fi
else
    rm -f $PASSWORD_FILE
fi

rest-server --listen ":80" --path $DATA_DIRECTORY
