# Restic Server

Restic Server is a sample server that implements restic's REST backend API.
It has been developed for demonstration purpose and is not intended to be used in production.

## Getting started

By default the server persists backup data in `/tmp/restic`.
Build and start the server with a custom persistence directory:

```
go install
restic-server -path /user/home/backup
```

The server uses an `.htpasswd` file to specify users. You can create such a file at the root of the persistence
directory by executing the following command. In order to append new user to the file, just omit the `-c` argument.

```
htpasswd -s -c .htpasswd username
```

By default the server uses HTTP protocol. This is not very secure since with Basic Authentication, username and
passwords will be present in every request. In order to enable TLS support just add the `-tls` argument and add a
private and public key at the root of your persistence directory.

Signed certificate is required by the restic backend, but if you just want to test the feature you can generate unsigned
keys with the following commands:

```
openssl genrsa -out private_key 2048
openssl req -new -x509 -key private_key -out public_key -days 365
```
