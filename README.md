# Restic Server

[![Build Status](https://travis-ci.org/zcalusic/restic-server.svg?branch=master)](https://travis-ci.org/zcalusic/restic-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/zcalusic/restic-server)](https://goreportcard.com/report/github.com/zcalusic/restic-server)
[![GoDoc](https://godoc.org/github.com/zcalusic/restic-server?status.svg)](https://godoc.org/github.com/zcalusic/restic-server)
[![License](https://img.shields.io/badge/license-BSD%20%282--Clause%29-003262.svg?maxAge=2592000)](https://github.com/zcalusic/restic-server/blob/master/LICENSE)
[![Powered by](https://img.shields.io/badge/powered_by-Go-5272b4.svg?maxAge=2592000)](https://golang.org/)

Restic Server is a sample server that implements restic's REST backend API. It has been developed for demonstration
purpose and is not intended to be used in production.

## Getting started

By default the server persists backup data in `/tmp/restic`. Build and start the server with a custom persistence
directory:

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
## Contributors

Contributors are welcome, just open a new issue / pull request.

## License

```
The BSD 2-Clause License

Copyright © 2015, Bertil Chapuis
Copyright © 2016, Zlatko Čalušić
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

* Redistributions of source code must retain the above copyright notice, this
  list of conditions and the following disclaimer.

* Redistributions in binary form must reproduce the above copyright notice,
  this list of conditions and the following disclaimer in the documentation
  and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
```
