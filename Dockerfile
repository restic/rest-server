FROM alpine

ENV DATA_DIRECTORY /data
ENV PASSWORD_FILE /data/.htpasswd

RUN apk add --no-cache --update apache2-utils

COPY docker/create_user /usr/bin/
COPY docker/delete_user /usr/bin/
COPY docker/entrypoint.sh /entrypoint.sh
COPY rest-server /usr/bin

VOLUME /data
EXPOSE 8000

CMD [ "/entrypoint.sh" ]
