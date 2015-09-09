package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/restic/restic/backend"
	"github.com/stretchr/testify/require"
)

func TestRepositoryConfig(t *testing.T) {
	path := "/tmp/repository"
	repository, _ := NewRepository(path)
	defer os.RemoveAll(path)

	_, e1 := os.Stat(path)
	require.NoError(t, e1, "repository not created")

	require.False(t, repository.HasConfig())

	_, e2 := repository.ReadConfig()
	require.Error(t, e2, "reading config should fail")

	e3 := repository.WriteConfig([]byte("test"))
	require.NoError(t, e3, "writing config should succeed")

	require.True(t, repository.HasConfig())

	config, _ := repository.ReadConfig()
	require.Equal(t, config, []byte("test"), "reading config should succeed")
}

func TestRepositoryBlob(t *testing.T) {
	path := "/tmp/repository"
	repository, _ := NewRepository(path)
	//defer os.RemoveAll(path)

	_, e1 := os.Stat(path)
	require.NoError(t, e1, "repository not created")

	require.False(t, repository.HasBlob(backend.Data, BlobID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")))

	_, e2 := repository.ReadBlob(backend.Data, BlobID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	require.Error(t, e2, "reading blob should fail")

	e3 := repository.WriteBlob(backend.Data, BlobID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), []byte("test"))
	require.NoError(t, e3, "saving blob should succeed")

	require.True(t, repository.HasBlob(backend.Data, BlobID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")))

	blob, _ := repository.ReadBlob(backend.Data, BlobID("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"))
	bytes, e4 := ioutil.ReadAll(blob)
	require.NoError(t, e4, e4.Error())
	require.Equal(t, bytes, []byte("test"), "reading blob should succeed")
}
