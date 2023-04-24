Changelog for rest-server unreleased (UNRELEASED)
============================================

The following sections list the changes in rest-server unreleased relevant
to users. The changes are ordered by importance.

Summary
-------

 * Fix #183: Allow usernames containing underscore and more
 * Fix #219: Ignore unexpected files in the data/ folder
 * Fix #1871: Return 500 "Internal server error" if files cannot be read
 * Chg #207: Return error if command-line arguments are specified
 * Chg #208: Update dependencies and require Go 1.17 or newer
 * Enh #133: Cache basic authentication credentials
 * Enh #187: Allow configurable location for `.htpasswd` file

Details
-------

 * Bugfix #183: Allow usernames containing underscore and more

   The security fix in rest-server 0.11.0 (#131) disallowed usernames containing and
   underscore "_". The list of allowed characters has now been changed to include Unicode
   characters, numbers, "_", "-", "." and "@".

   https://github.com/restic/rest-server/issues/183
   https://github.com/restic/rest-server/pull/184

 * Bugfix #219: Ignore unexpected files in the data/ folder

   If the data folder of a repository contained files, this would prevent restic from retrieving a
   list of file data files. This has been fixed. As a workaround remove the files that are directly
   contained in the data folder (e.g., `.DS_Store` files).

   https://github.com/restic/rest-server/issues/219
   https://github.com/restic/rest-server/pull/221

 * Bugfix #1871: Return 500 "Internal server error" if files cannot be read

   When files in a repository cannot be read by rest-server, for example after running `restic
   prune` directly on the server hosting the repositories in a way that causes filesystem
   permissions to be wrong, rest-server previously returned 404 "Not Found" as status code. This
   was causing confusing for users.

   The error handling has now been fixed to only return 404 "Not Found" if the file actually does not
   exist. Otherwise a 500 "Internal server error" is reported to the client and the underlying
   error is logged at the server side.

   https://github.com/restic/rest-server/issues/1871
   https://github.com/restic/rest-server/pull/195

 * Change #207: Return error if command-line arguments are specified

   Command line arguments are ignored by rest-server, but there was previously no indication of
   this when they were supplied anyway.

   To prevent usage errors an error is now printed when command line arguments are supplied,
   instead of them being silently ignored.

   https://github.com/restic/rest-server/pull/207

 * Change #208: Update dependencies and require Go 1.17 or newer

   Most dependencies have been updated. Since some libraries require newer language features,
   support for Go 1.15-1.16 has been dropped, which means that rest-server now requires at least
   Go 1.17 to build.

   https://github.com/restic/rest-server/pull/208

 * Enhancement #133: Cache basic authentication credentials

   To speed up the verification of basic auth credentials, rest-server now caches passwords for a
   minute in memory. That way the expensive verification of basic auth credentials can be skipped
   for most requests issued by a single restic run. The password is kept in memory in a hashed form
   and not as plaintext.

   https://github.com/restic/rest-server/issues/133
   https://github.com/restic/rest-server/pull/138

 * Enhancement #187: Allow configurable location for `.htpasswd` file

   It is now possible to specify the location of the `.htpasswd` file using the `--htpasswd-file`
   option.

   https://github.com/restic/rest-server/issues/187
   https://github.com/restic/rest-server/pull/188


Changelog for rest-server 0.11.0 (2022-02-10)
============================================

The following sections list the changes in rest-server 0.11.0 relevant
to users. The changes are ordered by importance.

Summary
-------

 * Sec #131: Prevent loading of usernames containing a slash
 * Fix #119: Fix Docker configuration for `DISABLE_AUTHENTICATION`
 * Fix #142: Fix possible data loss due to interrupted network connections
 * Fix #155: Reply "insufficient storage" on disk full or over-quota
 * Fix #157: Use platform-specific temporary directory as default data directory
 * Chg #112: Add subrepo support and refactor server code
 * Chg #146: Build rest-server at docker container build time
 * Enh #122: Verify uploaded files
 * Enh #126: Allow running rest-server via systemd socket activation
 * Enh #148: Expand use of security features in example systemd unit file

Details
-------

 * Security #131: Prevent loading of usernames containing a slash

   "/" is valid char in HTTP authorization headers, but is also used in rest-server to map
   usernames to private repos.

   This commit prevents loading maliciously composed usernames like "/foo/config" by
   restricting the allowed characters to the unicode character class, numbers, "-", "." and "@".

   This prevents requests to other users files like:

   Curl -v -X DELETE -u foo/config:attack http://localhost:8000/foo/config

   https://github.com/restic/rest-server/issues/131
   https://github.com/restic/rest-server/pull/132
   https://github.com/restic/rest-server/pull/137

 * Bugfix #119: Fix Docker configuration for `DISABLE_AUTHENTICATION`

   Rest-server 0.10.0 introduced a regression which caused the `DISABLE_AUTHENTICATION`
   environment variable to stop working for the Docker container. This has been fixed by
   automatically setting the option `--no-auth` to disable authentication.

   https://github.com/restic/rest-server/issues/119
   https://github.com/restic/rest-server/pull/124

 * Bugfix #142: Fix possible data loss due to interrupted network connections

   When rest-server was run without `--append-only` it was possible to lose uploaded files in a
   specific scenario in which a network connection was interrupted.

   For the data loss to occur a file upload by restic would have to be interrupted such that restic
   notices the interrupted network connection before the rest-server. Then restic would have to
   retry the file upload and finish it before the rest-server notices that the initial upload has
   failed. Then the uploaded file would be accidentally removed by rest-server when trying to
   cleanup the failed upload.

   This has been fixed by always uploading to a temporary file first which is moved in position only
   once it was uploaded completely.

   https://github.com/restic/rest-server/pull/142

 * Bugfix #155: Reply "insufficient storage" on disk full or over-quota

   When there was no space left on disk, or any other write-related error occurred, rest-server
   replied with HTTP status code 400 (Bad request). This is misleading (restic client will dump
   the status code to the user).

   Rest-server now replies with two different status codes in these situations: * HTTP 507
   "Insufficient storage" is the status on disk full or repository over-quota * HTTP 500
   "Internal server error" is used for other disk-related errors

   https://github.com/restic/rest-server/issues/155
   https://github.com/restic/rest-server/pull/160

 * Bugfix #157: Use platform-specific temporary directory as default data directory

   If no data directory is specificed, then rest-server now uses the Go standard library
   functions to retrieve the standard temporary directory path for the current platform.

   https://github.com/restic/rest-server/issues/157
   https://github.com/restic/rest-server/pull/158

 * Change #112: Add subrepo support and refactor server code

   Support for multi-level repositories has been added, so now each user can have its own
   subrepositories. This feature is always enabled.

   Authentication for the Prometheus /metrics endpoint can now be disabled with the new
   `--prometheus-no-auth` flag.

   We have split out all HTTP handling to a separate `repo` subpackage to cleanly separate the
   server code from the code that handles a single repository. The new RepoHandler also makes it
   easier to reuse rest-server as a Go component in any other HTTP server.

   The refactoring makes the code significantly easier to follow and understand, which in turn
   makes it easier to add new features, audit for security and debug issues.

   https://github.com/restic/rest-server/issues/109
   https://github.com/restic/rest-server/issues/107
   https://github.com/restic/rest-server/pull/112

 * Change #146: Build rest-server at docker container build time

   The Dockerfile now includes a build stage such that the latest rest-server is always built and
   packaged. This is done in a standard golang container to ensure a clean build environment and
   only the final binary is shipped rather than the whole build environment.

   https://github.com/restic/rest-server/issues/146
   https://github.com/restic/rest-server/pull/145

 * Enhancement #122: Verify uploaded files

   The rest-server now by default verifies that the hash of content of uploaded files matches
   their filename. This ensures that transmission errors are detected and forces restic to retry
   the upload. On low-power devices it can make sense to disable this check by passing the
   `--no-verify-upload` flag.

   https://github.com/restic/rest-server/issues/122
   https://github.com/restic/rest-server/pull/130

 * Enhancement #126: Allow running rest-server via systemd socket activation

   We've added the option to have systemd create the listening socket and start the rest-server on
   demand.

   https://github.com/restic/rest-server/issues/126
   https://github.com/restic/rest-server/pull/151
   https://github.com/restic/rest-server/pull/127

 * Enhancement #148: Expand use of security features in example systemd unit file

   The example systemd unit file now enables additional systemd features to mitigate potential
   security vulnerabilities in rest-server and the various packages and operating system
   components which it relies upon.

   https://github.com/restic/rest-server/issues/148
   https://github.com/restic/rest-server/pull/149


Changelog for rest-server 0.10.0 (2020-09-13)
============================================

The following sections list the changes in rest-server 0.10.0 relevant
to users. The changes are ordered by importance.

Summary
-------

 * Sec #60: Require auth by default, add --no-auth flag
 * Sec #64: Refuse overwriting config file in append-only mode
 * Sec #117: Stricter path sanitization
 * Chg #102: Remove vendored dependencies
 * Enh #44: Add changelog file

Details
-------

 * Security #60: Require auth by default, add --no-auth flag

   In order to prevent users from accidentally exposing rest-server without authentication,
   rest-server now defaults to requiring a .htpasswd. If you want to disable authentication, you
   need to explicitly pass the new --no-auth flag.

   https://github.com/restic/rest-server/issues/60
   https://github.com/restic/rest-server/pull/61

 * Security #64: Refuse overwriting config file in append-only mode

   While working on the `rclone serve restic` command we noticed that is currently possible to
   overwrite the config file in a repo even if `--append-only` is specified. The first commit adds
   proper tests, and the second commit fixes the issue.

   https://github.com/restic/rest-server/pull/64

 * Security #117: Stricter path sanitization

   The framework we're using in rest-server to decode paths to repositories allowed specifying
   URL-encoded characters in paths, including sensitive characters such as `/` (encoded as
   `%2F`).

   We've changed this unintended behavior, such that rest-server now rejects such paths. In
   particular, it is no longer possible to specify sub-repositories for users by encoding the
   path with `%2F`, such as `http://localhost:8000/foo%2Fbar`, which means that this will
   unfortunately be a breaking change in that case.

   If using sub-repositories for users is important to you, please let us know in the forum, so we
   can learn about your use case and implement this properly. As it currently stands, the ability
   to use sub-repositories was an unintentional feature made possible by the URL decoding
   framework used, and hence never meant to be supported in the first place. If we wish to have this
   feature in rest-server, we'd like to have it implemented properly and intentionally.

   https://github.com/restic/rest-server/issues/117

 * Change #102: Remove vendored dependencies

   We've removed the vendored dependencies (in the subdir `vendor/`) similar to what we did for
   `restic` itself. When building restic, the Go compiler automatically fetches the
   dependencies. It will also cryptographically verify that the correct code has been fetched by
   using the hashes in `go.sum` (see the link to the documentation below).

   Building the rest-server now requires Go 1.11 or newer, since we're using Go Modules for
   dependency management. Older Go versions are not supported any more.

   https://github.com/restic/rest-server/issues/102
   https://golang.org/cmd/go/#hdr-Module_downloading_and_verification

 * Enhancement #44: Add changelog file

   https://github.com/restic/rest-server/issues/44
   https://github.com/restic/rest-server/pull/62


