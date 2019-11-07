package restserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"
)

// Server embed a http server and some tools
type Server struct {
	http.Server

	conf *Config
	log  *Logger
	auth *PasswordFile

	repoSize int64 // repoSize must be accessed using sync/atomic

	shutdownCh chan struct{} // shutdownCh is used to monitor when server can start shutdown process
	startedCh  chan struct{} // startCh is used to monitor when server has been started
	testing    bool          // testing defines wether we are in a testing environement
}

// NewServer creates a new server from a config
func NewServer(conf *Config) *Server {
	s := &Server{
		conf:       conf,
		shutdownCh: make(chan struct{}),
		startedCh:  make(chan struct{}),
	}
	return s
}

// Run the server
func (s *Server) Run() error {
	l, err := NewLogger(s.conf)
	if err != nil {
		return err
	}

	s.log = l

	s.log.Info("server is starting...")

	if s.conf.Debug {
		s.log.Info("debug enabled")
	}

	s.log.Info(fmt.Sprintf("data directory: %s", s.conf.Path))

	if s.conf.CPUProfile != "" {
		f, err := os.Create(s.conf.CPUProfile)
		if err != nil {
			return err
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			return err
		}

		s.log.Info("cpu profiling enabled")
	}

	if s.conf.PrivateRepos {
		s.log.Info("private repositories enabled")
	} else {
		s.log.Info("private repositories disabled")
	}

	r := s.NewRouter()

	if !s.conf.NoAuth {

		auth, err := NewPasswordFile(filepath.Join(s.conf.Path, ".htpasswd"))
		if err != nil {
			return fmt.Errorf("cannot load .htpasswd (use --no-auth to disable): %v", err)
		}

		s.auth = auth
		r.Use(s.AuthMiddleware)

		s.log.Info("authentication enabled")
	} else {
		s.log.Info("authentication disabled")
	}

	s.Addr = s.conf.ListenAddr
	s.Handler = r

	// Server is done starting
	s.startedCh <- struct{}{}

	if !s.testing {
		go func() {
			s.log.Info(fmt.Sprintf("started listening at: '%s'", s.conf.ListenAddr))

			if s.conf.TLS {
				s.log.Info("tls enabled")
				if err = s.ListenAndServeTLS(s.conf.TLSCertFile, s.conf.TLSKeyFile); err != nil {
					s.log.Error(err)
				}
			} else {
				if err := s.ListenAndServe(); err != nil {
					s.log.Error(err)
				}
			}

			s.shutdownCh <- struct{}{}
		}()

		// Wait for server to shutdown gracefully
		s.WaitShutdown()
	}
	<-s.shutdownCh
	s.log.Info("server is shutting down...")

	pprof.StopCPUProfile()
	s.log.Close()
	return nil
}

// WaitShutdown to avoid a brutal shutdown
func (s *Server) WaitShutdown() {
	irqSig := make(chan os.Signal, 1)
	signal.Notify(irqSig, syscall.SIGINT, syscall.SIGTERM)

	// Wait for interuption signal
	<-irqSig
	s.log.Info("received interuption signal")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.Shutdown(ctx)
	if err != nil {
		s.log.Error(err)
	}
}
