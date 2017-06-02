# Rest Server

[![Build Status](https://travis-ci.org/restic/rest-server.svg?branch=master)](https://travis-ci.org/restic/rest-server)
[![Go Report Card](https://goreportcard.com/badge/github.com/restic/rest-server)](https://goreportcard.com/report/github.com/restic/rest-server)
[![GoDoc](https://godoc.org/github.com/restic/rest-server?status.svg)](https://godoc.org/github.com/restic/rest-server)
[![License](https://img.shields.io/badge/license-BSD%20%282--Clause%29-003262.svg?maxAge=2592000)](https://github.com/restic/rest-server/blob/master/LICENSE)
[![Powered by](https://img.shields.io/badge/powered_by-Go-5272b4.svg?maxAge=2592000)](https://golang.org/)

Rest Server is a high performance HTTP server that implements restic's [REST backend
API](https://github.com/restic/restic/blob/master/doc/rest_backend.rst).  It provides secure and efficient way to backup
data remotely, using [restic](https://github.com/restic/restic) backup client.

## Requirements

Rest Server requires Go 1.7 or higher to build.  The only tested compiler is the official Go compiler.  Building server
with gccgo may work, but is not supported.

The required version of restic backup client to use with rest-server is
[v0.6.1](https://github.com/restic/restic/releases/tag/v0.6.1) or higher, due to some
[changes](https://github.com/restic/restic/commit/1a538509d0232f1a532266e07da509875fe9e0d6) in the REST backend API and
performance [improvements](https://github.com/restic/restic/commit/04b262d8f10ba9eacde041734c08f806c4685e7f).

## Installation

Run ```go run build.go```, afterwards you'll find the binary in the current directory.  You can move it anywhere you
want.  There's also an [example systemd service
file](https://github.com/restic/rest-server/blob/master/etc/rest-server.service) included, so you can get it up &
running as a proper service in no time.  Of course, you can also test it from the command line.

```
% go run build.go

% ./rest-server --help
Run a REST server for use with restic

Usage:
  rest-server [flags]

Flags:
      --cpuprofile string   write CPU profile to file
      --debug               output debug messages
  -h, --help                help for rest-server
      --listen string       listen address (default ":8000")
      --log string          log HTTP requests in the combined log format
      --path string         data directory (default "/tmp/restic")
      --tls                 turn on TLS support
```

Alternatively, you can compile and install it in your $GOBIN with a standard `go install`.  But, beware, you won't have
version info built into binary, when compiled that way.

## Getting started

By default the server persists backup data in `/tmp/restic`.  Start the server with a custom persistence directory:

```
% rest-server --path /user/home/backup
```

The server uses an `.htpasswd` file to specify users.  You can create such a file at the root of the persistence
directory by executing the following command.  In order to append new user to the file, just omit the `-c` argument.

```
% htpasswd -s -c .htpasswd username
```

By default the server uses HTTP protocol.  This is not very secure since with Basic Authentication, username and
passwords will travel in cleartext in every request.  In order to enable TLS support just add the `-tls` argument and
add a private and public key at the root of your persistence directory.

Signed certificate is required by the restic backend, but if you just want to test the feature you can generate unsigned
keys with the following commands:

```
% openssl genrsa -out private_key 2048
% openssl req -new -x509 -key private_key -out public_key -days 365
```

Rest Server uses exactly the same directory structure as local backend, so you should be able to access it both locally
and via HTTP, even simultaneously.

To learn how to use restic backup client with REST backend, please consult [restic
manual](https://restic.readthedocs.io/en/latest/manual.html#rest-server).

## Contributors

Contributors are welcome, just open a new issue / pull request.

## License

```
The BSD 2-Clause License

Copyright © 2015, Bertil Chapuis
Copyright © 2016, Zlatko Čalušić, Alexander Neumann
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
