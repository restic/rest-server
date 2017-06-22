FROM alpine:3.6

RUN apk add --no-cache --update apache2-utils
COPY docker/*_user rest-server /usr/bin/
COPY docker/entry.sh /

ENV DATA_DIRECTORY=/data
ENV PASSWORD_FILE=/data/.htpasswd

VOLUME /data

EXPOSE  80

ENTRYPOINT ["/entry.sh"]
