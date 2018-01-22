package main

import (
	"testing"

	restserver "github.com/restic/rest-server"
)

func TestTLSSettings(t *testing.T) {
	type expected struct {
		TLSKey  string
		TLSCert string
		Error   bool
	}
	type passed struct {
		Path    string
		TLS     bool
		TLSKey  string
		TLSCert string
	}

	var tests = []struct {
		passed   passed
		expected expected
	}{
		{passed{TLS: false}, expected{"", "", false}},
		{passed{TLS: true}, expected{"/tmp/restic/private_key", "/tmp/restic/public_key", false}},
		{passed{Path: "/tmp", TLS: true}, expected{"/tmp/private_key", "/tmp/public_key", false}},
		{passed{Path: "/tmp", TLS: true, TLSKey: "/etc/restic/key", TLSCert: "/etc/restic/cert"}, expected{"/etc/restic/key", "/etc/restic/cert", false}},
		{passed{Path: "/tmp", TLS: false, TLSKey: "/etc/restic/key", TLSCert: "/etc/restic/cert"}, expected{"", "", true}},
		{passed{Path: "/tmp", TLS: false, TLSKey: "/etc/restic/key"}, expected{"", "", true}},
		{passed{Path: "/tmp", TLS: false, TLSCert: "/etc/restic/cert"}, expected{"", "", true}},
	}

	defaultConfig := restserver.Config
	for _, test := range tests {

		t.Run("", func(t *testing.T) {
			defer func() { restserver.Config = defaultConfig }()
			if test.passed.Path != "" {
				restserver.Config.Path = test.passed.Path
			}
			restserver.Config.TLS = test.passed.TLS
			restserver.Config.TLSKey = test.passed.TLSKey
			restserver.Config.TLSCert = test.passed.TLSCert

			gotTLS, gotKey, gotCert, err := tlsSettings()
			if err != nil && !test.expected.Error {
				t.Fatalf("tls_settings returned err (%v)", err)
			}
			if test.expected.Error {
				if err == nil {
					t.Fatalf("Error not returned properly (%v)", test)
				} else {
					return
				}
			}
			if gotTLS != test.passed.TLS {
				t.Errorf("TLS enabled, want (%v), got (%v)", test.passed.TLS, gotTLS)
			}
			wantKey := test.expected.TLSKey
			if gotKey != wantKey {
				t.Errorf("wrong TLSPrivPath path, want (%v), got (%v)", wantKey, gotKey)
			}

			wantCert := test.expected.TLSCert
			if gotCert != wantCert {
				t.Errorf("wrong TLSCertPath path, want (%v), got (%v)", wantCert, gotCert)
			}

		})
	}
}

func TestIsUserPath(t *testing.T) {
	var tests = []struct {
		username string
		path     string
		result   bool
	}{
		{"foo", "/", false},
		{"foo", "/foo", true},
		{"foo", "/foo/", true},
		{"foo", "/foo/bar", true},
		{"foo", "/foobar", false},
	}

	for _, test := range tests {
		result := isUserPath(test.username, test.path)
		if result != test.result {
			t.Errorf("isUserPath(%q, %q) was incorrect, got: %v, want: %v.", test.username, test.path, result, test.result)
		}
	}
}
