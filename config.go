package restserver

import (
	"errors"
)

// Config defines the server configuration
type Config struct {
	ListenAddr  string // Listen is the address the http server will listen
	Path        string // Path is the data storage directory path
	MaxRepoSize int64  // MaxRepoSize is the maximal repo size in bytes
	Log         string // Log is the optionnal logging file path

	CPUProfile string // CPUProfile is the optionnal cpu profiling file path
	Debug      bool   // Debug defines whether to print debug messages

	TLS         bool   // Debug defines whether to enable TLS
	TLSKeyFile  string // TLSKeyFile is the path to the TLS key file
	TLSCertFile string // TLSCertFile is the path to the TLS cert file

	NoAuth       bool // NoAuth defines whether to enable http auth
	AppendOnly   bool // AppendOnly defines whether to enable append only
	PrivateRepos bool // PrivateRepos defines whether to enable private repos

	Prometheus bool // Prometheus defines whether to enable prometheus metrics
}

// Check will check the configuration and return any error
func (conf *Config) Check() error {

	// Check TLS configuration
	if !conf.TLS && (conf.TLSKeyFile != "" || conf.TLSCertFile != "") {
		return errors.New("requires enabled TLS")
	} else if !conf.TLS {
		return nil
	}

	if conf.TLSKeyFile == "" {
		return errors.New("missing tls key file")
	}
	if conf.TLSCertFile == "" {
		return errors.New("missing tls cert file")
	}

	return nil
}
