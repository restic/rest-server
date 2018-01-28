FROM alpine

ENV DATA_DIRECTORY /data
ENV PASSWORD_FILE /data/.htpasswd

RUN apk add --no-cache --update apache2-utils

COPY rest-server docker/*_user /usr/bin/

VOLUME /data

EXPOSE 8000

COPY docker/entrypoint.sh /entrypoint.sh

CMD [ "/entrypoint.sh" ]
