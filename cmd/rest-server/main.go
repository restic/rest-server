package main

import (
	"log"
	"os"

	restserver "github.com/restic/rest-server"
	"github.com/spf13/cobra"
)

// Default restserver config
var conf = &restserver.Config{
	ListenAddr: ":8000",
	Path:       "/tmp/restic",
}

func init() {
	flags := rootCmd.Flags()
	flags.StringVar(&conf.ListenAddr, "listen", conf.ListenAddr, "listen address")
	flags.StringVar(&conf.Path, "path", conf.Path, "data directory")
	flags.Int64Var(&conf.MaxRepoSize, "max-size", conf.MaxRepoSize, "maximum data repo size in bytes")
	flags.StringVar(&conf.Log, "log", conf.Log, "log files")

	flags.StringVar(&conf.CPUProfile, "cpu-profile", conf.CPUProfile, "CPU profile file")
	flags.BoolVar(&conf.Debug, "debug", conf.Debug, "debug messages")

	flags.BoolVar(&conf.TLS, "tls", conf.TLS, "enabled TLS")
	flags.StringVar(&conf.TLSCertFile, "tls-cert", conf.TLSCertFile, "TLS certificate file")
	flags.StringVar(&conf.TLSKeyFile, "tls-key", conf.TLSKeyFile, "TLS key file")

	flags.BoolVar(&conf.NoAuth, "no-auth", conf.NoAuth, "disable http authentication")
	flags.BoolVar(&conf.AppendOnly, "append-only", conf.AppendOnly, "enable append only mode")
	flags.BoolVar(&conf.PrivateRepos, "private-repos", conf.PrivateRepos, "users can only access their private repo")

	flags.BoolVar(&conf.Prometheus, "prometheus", conf.Prometheus, "enable Prometheus metrics")
}

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "Run a restic server",
	Run: func(cmd *cobra.Command, args []string) {
		if err := conf.Check(); err != nil {
			log.Fatal(err)
		}

		server := restserver.NewServer(conf)

		if err := server.Run(); err != nil {
			log.Fatal(err)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
