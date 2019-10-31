package resticserver

import (
	"io"
	"log"
	"os"
)

// Logger structure
type Logger struct {
	log.Logger
	file *os.File
}

// NewLogger return a new configured logger
func NewLogger(conf *Config) (*Logger, error) {
	l := &Logger{}

	l.SetOutput(os.Stdout)
	l.SetFlags(log.LstdFlags)

	if conf.Log != "" {
		var err error

		if PathExist(conf.Log) {
			l.file, err = os.OpenFile(conf.Log, os.O_APPEND|os.O_WRONLY, 0640)
		} else {
			l.file, err = os.Create(conf.Log)
		}
		if err != nil {
			return nil, err
		}

		w := io.MultiWriter(os.Stdout, l.file)
		l.SetOutput(w)
	}

	return l, nil
}

// Close the logger
func (l *Logger) Close() error {
	if l.file != nil {
		if err := l.file.Close(); err != nil {
			return err
		}
	}
	return nil
}

// Info print a message
func (l *Logger) Info(i interface{}) {
	l.Printf("info: %v\n", i)
}

// Error print an error
func (l *Logger) Error(i interface{}) {
	l.Printf("error: %v\n", i)
}

// Fatal print an error and exit
func (l *Logger) Fatal(i interface{}) {
	l.Fatalf("fatal: %v\n", i)
}
