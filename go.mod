module github.com/restic/rest-server

go 1.14

require (
	github.com/coreos/go-systemd v0.0.0-20191104093116-d3cd4ed1dbcf
	github.com/felixge/httpsnoop v1.0.2 // indirect
	github.com/gorilla/handlers v1.5.1
	github.com/minio/sha256-simd v1.0.0
	github.com/itchio/ox v0.0.0-20200826161350-12c6ca18d236
	github.com/miolini/datacounter v1.0.2
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.30.0 // indirect
	github.com/prometheus/procfs v0.7.2 // indirect
	github.com/spf13/cobra v1.2.1
	golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	golang.org/x/sys v0.0.0-20210806184541-e5e7981a1069
	google.golang.org/protobuf v1.27.1 // indirect
)

replace goji.io v2.0.0+incompatible => github.com/goji/goji v2.0.0+incompatible

replace github.com/gorilla/handlers v1.3.0 => github.com/gorilla/handlers v1.3.0
