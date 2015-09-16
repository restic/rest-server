package main

import (
	"testing"

	"github.com/restic/restic/backend"
)

func TestRepositoryName(t *testing.T) {
	var name string
	var err error

	name, err = RepositoryName("")
	if err == nil {
		t.Error("empty string should produce an error")
	}

	name, err = RepositoryName("/")
	if err == nil {
		t.Error("empty repository name should produce an error")
	}

	name, err = RepositoryName("//")
	if err == nil {
		t.Error("empty repository name should produce an error")
	}

	name, err = RepositoryName("/$test")
	if err == nil {
		t.Error("special characters should produce an error")
	}

	name, err = RepositoryName("/test")
	if name != "test" {
		t.Errorf("repository name is %s but should be test", name)
	}

	name, err = RepositoryName("/test-1234")
	if name != "test-1234" {
		t.Errorf("repository name is %s but should be test-1234", name)
	}

	name, err = RepositoryName("/test_1234")
	if name != "test_1234" {
		t.Errorf("repository name is %s but should be test_1234", name)
	}
}

func TestBackendType(t *testing.T) {
	var bt backend.Type

	bt = BackendType("/")
	if bt != "" {
		t.Error("backend type should be nil")
	}

	bt = BackendType("/test")
	if bt != "" {
		t.Error("backend type should be nil")
	}

	bt = BackendType("/test/config")
	if bt != backend.Config {
		t.Error("backend type should be config")
	}

	bt = BackendType("/test/config/")
	if bt != backend.Config {
		t.Error("backend type should be config")
	}

	bt = BackendType("/test/config/test")
	if bt != backend.Config {
		t.Error("backend type should be config")
	}
}

func TestBlobID(t *testing.T) {
	var id backend.ID

	id = BlobID("/")
	if !id.IsNull() {
		t.Error("blob id should be nil")
	}

	id = BlobID("/test")
	if !id.IsNull() {
		t.Error("blob id should be nil")
	}

	id = BlobID("/test/data")
	if !id.IsNull() {
		t.Error("blob id should be nil")
	}

	id = BlobID("/test/data/")
	if !id.IsNull() {
		t.Error("blob id should be nil")
	}

	id = BlobID("/test/data/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if id.IsNull() {
		t.Error("blob id should not be nil")
	}
}

func TestRestAPI(t *testing.T) {
	type route struct {
		method string
		path   string
	}

	validEndpoints := []route{
		route{"HEAD", "/repo/config"},
		route{"GET", "/repo/config"},
		route{"POST", "/repo/config"},
		route{"GET", "/repo/data/"},
		route{"HEAD", "/repo/data/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/data/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"POST", "/repo/data/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"DELETE", "/repo/data/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/snapshots/"},
		route{"HEAD", "/repo/snapshots/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/snapshots/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"POST", "/repo/snapshots/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"DELETE", "/repo/snapshots/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/index/"},
		route{"HEAD", "/repo/index/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/index/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"POST", "/repo/index/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"DELETE", "/repo/index/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/locks/"},
		route{"HEAD", "/repo/locks/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/locks/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"POST", "/repo/locks/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"DELETE", "/repo/locks/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/keys/"},
		route{"HEAD", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"POST", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"DELETE", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}

	for _, route := range validEndpoints {
		if RestAPI(route.method, route.path) == nil {
			t.Errorf("request %s %s should return a handler", route.method, route.path)
		}
	}

	invalidEndpoints := []route{
		route{"GET", "/"},
		route{"GET", "/repo"},
		route{"GET", "/repo/config/"},
		route{"GET", "/repo/config/aaaa"},
		route{"GET", "/repo/data"},
		route{"GET", "/repo/data/aaaaaaa"},
		route{"GET", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		route{"GET", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/"},
		route{"GET", "/repo/keys/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/test"},
	}

	for _, route := range invalidEndpoints {
		if RestAPI(route.method, route.path) != nil {
			t.Errorf("request %s %s should return nil", route.method, route.path)
		}
	}

}
