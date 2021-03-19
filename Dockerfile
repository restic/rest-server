FROM golang:alpine3.13 AS builder
LABEL stage=builder
WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 go build -o rest-server ./cmd/rest-server


FROM alpine:3.13 AS final
WORKDIR /app

ENV DATA_DIRECTORY /data
ENV PASSWORD_FILE /data/.htpasswd
ENV PATH="/app:${PATH}"

RUN apk add --no-cache --update apache2-utils

COPY docker/. .
COPY --from=builder /workspace/rest-server .

VOLUME /data
EXPOSE 8000

CMD [ "./entrypoint.sh" ]
