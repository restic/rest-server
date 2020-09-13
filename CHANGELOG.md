Changelog for rest-server 0.10.0 (2020-09-13)
============================================

The following sections list the changes in rest-server 0.10.0 relevant
to users. The changes are ordered by importance.

Summary
-------

 * Sec #117: Stricter path sanitization
 * Sec #60: Require auth by default, add --no-auth flag
 * Sec #64: Refuse overwriting config file in append-only mode
 * Chg #102: Remove vendored dependencies
 * Enh #44: Add changelog file

Details
-------

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


