package repo

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/minio/sha256-simd"
	"github.com/miolini/datacounter"
	"github.com/restic/rest-server/quota"
)

// Options are options for the Handler accepted by New
type Options struct {
	AppendOnly     bool // if set, delete actions are not allowed
	WriteOnly      bool // if set, delete and get blob actions are not allowed
	Debug          bool
	DirMode        os.FileMode
	FileMode       os.FileMode
	NoVerifyUpload bool

	// If set, we will panic when an internal server error happens. This
	// makes it easier to debug such errors.
	PanicOnError bool

	BlobMetricFunc BlobMetricFunc
	QuotaManager   *quota.Manager
}

func (o *Options) deleteDataForbidden() bool {
	return o.AppendOnly || o.WriteOnly
}

// DefaultDirMode is the file mode used for directory creation if not
// overridden in the Options
const DefaultDirMode os.FileMode = 0700

// DefaultFileMode is the file mode used for file creation if not
// overridden in the Options
const DefaultFileMode os.FileMode = 0600

// New creates a new Handler for a single Restic backup repo.
// path is the full filesystem path to this repo directory.
// opt is a set of options.
func New(path string, opt Options) (*Handler, error) {
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if opt.DirMode == 0 {
		opt.DirMode = DefaultDirMode
	}
	if opt.FileMode == 0 {
		opt.FileMode = DefaultFileMode
	}
	h := Handler{
		path: path,
		opt:  opt,
	}
	return &h, nil
}

// Handler handles all REST API requests for a single Restic backup repo
// Spec: https://restic.readthedocs.io/en/latest/100_references.html#rest-backend
type Handler struct {
	path string // filesystem path of repo
	opt  Options
}

// httpDefaultError write a HTTP error with the default description
func httpDefaultError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

// httpMethodNotAllowed writes a 405 Method Not Allowed HTTP error with
// the required Allow header listing the methods that are allowed.
func httpMethodNotAllowed(w http.ResponseWriter, allowed []string) {
	w.Header().Set("Allow", strings.Join(allowed, ", "))
	httpDefaultError(w, http.StatusMethodNotAllowed)
}

// errFileContentDoesntMatchHash is the error raised when the file content hash
// doesn't match the hash provided in the URL
var errFileContentDoesntMatchHash = errors.New("file content does not match hash")

// BlobPathRE matches valid blob URI paths with optional object IDs
var BlobPathRE = regexp.MustCompile(`^/(data|index|keys|locks|snapshots)/([0-9a-f]{64})?$`)

// ObjectTypes are subdirs that are used for object storage
var ObjectTypes = []string{"data", "index", "keys", "locks", "snapshots"}

// FileTypes are files stored directly under the repo direct that are accessible
// through a request
var FileTypes = []string{"config"}

func isHashed(objectType string) bool {
	return objectType == "data"
}

// BlobOperation describe the current blob operation in the BlobMetricFunc callback.
type BlobOperation byte

// Define all valid operations.
const (
	BlobRead   = 'R' // A blob has been read
	BlobWrite  = 'W' // A blob has been written
	BlobDelete = 'D' // A blob has been deleted
)

// BlobMetricFunc is the callback signature for blob metrics. Such a callback
// can be passed in the Options to keep track of various metrics.
// objectType: one of ObjectTypes
// operation: one of the BlobOperations above
// nBytes: the number of bytes affected, or 0 if not relevant
// TODO: Perhaps add http.Request for the username so that this can be cached?
type BlobMetricFunc func(objectType string, operation BlobOperation, nBytes uint64)

// ServeHTTP performs strict matching on the repo part of the URL path and
// dispatches the request to the appropriate handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	if urlPath == "/" {
		// TODO: add HEAD and GET
		switch r.Method {
		case "POST":
			h.createRepo(w, r)
		default:
			httpMethodNotAllowed(w, []string{"POST"})
		}
		return
	} else if urlPath == "/config" {
		switch r.Method {
		case "HEAD":
			h.checkConfig(w, r)
		case "GET":
			h.getConfig(w, r)
		case "POST":
			h.saveConfig(w, r)
		case "DELETE":
			h.deleteConfig(w, r)
		default:
			httpMethodNotAllowed(w, []string{"HEAD", "GET", "POST", "DELETE"})
		}
		return
	} else if objectType, objectID := h.getObject(urlPath); objectType != "" {
		if objectID == "" {
			// TODO: add HEAD
			switch r.Method {
			case "GET":
				h.listBlobs(w, r)
			default:
				httpMethodNotAllowed(w, []string{"GET"})
			}

			return
		}

		switch r.Method {
		case "HEAD":
			h.checkBlob(w, r)
		case "GET":
			h.getBlob(w, r)
		case "POST":
			h.saveBlob(w, r)
		case "DELETE":
			h.deleteBlob(w, r)
		default:
			httpMethodNotAllowed(w, []string{"HEAD", "GET", "POST", "DELETE"})
		}

		return
	}
	httpDefaultError(w, http.StatusNotFound)
}

// getObject parses the URL path and returns the objectType and objectID,
// if any. The objectID is optional.
func (h *Handler) getObject(urlPath string) (objectType, objectID string) {
	m := BlobPathRE.FindStringSubmatch(urlPath)
	if len(m) == 0 {
		return "", "" // no match
	}
	if len(m) == 2 || m[2] == "" {
		return m[1], "" // no objectID
	}
	return m[1], m[2]
}

// getSubPath returns the path for a file or subdir in the root of the repo.
func (h *Handler) getSubPath(name string) string {
	return filepath.Join(h.path, name)
}

// getObjectPath returns the path for an object file in the repo.
// The passed in objectType and objectID must be valid due to earlier validation
func (h *Handler) getObjectPath(objectType, objectID string) string {
	// If we hit an error, this is a programming error, because all of these
	// must have been validated before. We still check them here as a safeguard.
	if objectType == "" || objectID == "" {
		panic("invalid objectType or objectID")
	}
	if isHashed(objectType) {
		if len(objectID) < 2 {
			// Should never happen, because BlobPathRE checked this
			panic("getObjectPath: objectID shorter than 2 chars")
		}
		// Added another dir in between with the first two characters of the hash
		return filepath.Join(h.path, objectType, objectID[:2], objectID)
	}

	return filepath.Join(h.path, objectType, objectID)
}

// sendMetric calls op.BlobMetricFunc if set. See its signature for details.
func (h *Handler) sendMetric(objectType string, operation BlobOperation, nBytes uint64) {
	if f := h.opt.BlobMetricFunc; f != nil {
		f(objectType, operation, nBytes)
	}
}

// needSize tells you if we need the file size for metrics of quota accounting
func (h *Handler) needSize() bool {
	return h.opt.BlobMetricFunc != nil || h.opt.QuotaManager != nil
}

// incrementRepoSpaceUsage increments the repo space usage if quota are enabled
func (h *Handler) incrementRepoSpaceUsage(by int64) {
	if h.opt.QuotaManager != nil {
		h.opt.QuotaManager.IncUsage(by)
	}
}

// wrapFileWriter wraps the file writer if repo quota are enabled, and returns it
// as is if not.
// If an error occurs, it returns both an error and the appropriate HTTP error code.
func (h *Handler) wrapFileWriter(r *http.Request, w io.Writer) (io.Writer, int, error) {
	if h.opt.QuotaManager == nil {
		return w, 0, nil // unmodified
	}
	return h.opt.QuotaManager.WrapWriter(r, w)
}

// checkConfig checks whether a configuration exists.
func (h *Handler) checkConfig(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("checkConfig()")
	}
	cfg := h.getSubPath("config")

	st, err := os.Stat(cfg)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// getConfig allows for a config to be retrieved.
func (h *Handler) getConfig(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("getConfig()")
	}
	cfg := h.getSubPath("config")

	bytes, err := ioutil.ReadFile(cfg)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	_, _ = w.Write(bytes)
}

// saveConfig allows for a config to be saved.
func (h *Handler) saveConfig(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("saveConfig()")
	}
	cfg := h.getSubPath("config")

	f, err := os.OpenFile(cfg, os.O_CREATE|os.O_WRONLY|os.O_EXCL, h.opt.FileMode)
	if err != nil && os.IsExist(err) {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusForbidden)
		return
	}

	_, err = io.Copy(f, r.Body)
	if err != nil {
		h.internalServerError(w, err)
		return
	}

	err = f.Close()
	if err != nil {
		h.internalServerError(w, err)
		return
	}

	_ = r.Body.Close()
}

// deleteConfig removes a config.
func (h *Handler) deleteConfig(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("deleteConfig()")
	}

	if h.opt.deleteDataForbidden() {
		httpDefaultError(w, http.StatusForbidden)
		return
	}

	cfg := h.getSubPath("config")

	if err := os.Remove(cfg); err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		if os.IsNotExist(err) {
			httpDefaultError(w, http.StatusNotFound)
		} else {
			h.internalServerError(w, err)
		}
		return
	}
}

const (
	mimeTypeAPIV1 = "application/vnd.x.restic.rest.v1"
	mimeTypeAPIV2 = "application/vnd.x.restic.rest.v2"
)

// listBlobs lists all blobs of a given type in an arbitrary order.
func (h *Handler) listBlobs(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("listBlobs()")
	}

	switch r.Header.Get("Accept") {
	case mimeTypeAPIV2:
		h.listBlobsV2(w, r)
	default:
		h.listBlobsV1(w, r)
	}
}

// listBlobsV1 lists all blobs of a given type in an arbitrary order.
// TODO: unify listBlobsV1 and listBlobsV2
func (h *Handler) listBlobsV1(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("listBlobsV1()")
	}
	objectType, _ := h.getObject(r.URL.Path)
	if objectType == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type: %s", r.URL.Path))
		return
	}
	path := h.getSubPath(objectType)

	items, err := ioutil.ReadDir(path)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	var names []string
	for _, i := range items {
		if isHashed(objectType) {
			subpath := filepath.Join(path, i.Name())
			var subitems []os.FileInfo
			subitems, err = ioutil.ReadDir(subpath)
			if err != nil {
				if h.opt.Debug {
					log.Print(err)
				}
				httpDefaultError(w, http.StatusNotFound)
				return
			}
			for _, f := range subitems {
				names = append(names, f.Name())
			}
		} else {
			names = append(names, i.Name())
		}
	}

	data, err := json.Marshal(names)
	if err != nil {
		h.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", mimeTypeAPIV1)
	_, _ = w.Write(data)
}

// Blob represents a single blob, its name and its size.
type Blob struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// listBlobsV2 lists all blobs of a given type, together with their sizes, in an arbitrary order.
// TODO: unify listBlobsV1 and listBlobsV2
func (h *Handler) listBlobsV2(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("listBlobsV2()")
	}

	objectType, _ := h.getObject(r.URL.Path)
	if objectType == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type: %s", r.URL.Path))
		return
	}
	path := h.getSubPath(objectType)

	items, err := ioutil.ReadDir(path)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	var blobs []Blob
	for _, i := range items {
		if isHashed(objectType) {
			subpath := filepath.Join(path, i.Name())
			var subitems []os.FileInfo
			subitems, err = ioutil.ReadDir(subpath)
			if err != nil {
				if h.opt.Debug {
					log.Print(err)
				}
				httpDefaultError(w, http.StatusNotFound)
				return
			}
			for _, f := range subitems {
				blobs = append(blobs, Blob{Name: f.Name(), Size: f.Size()})
			}
		} else {
			blobs = append(blobs, Blob{Name: i.Name(), Size: i.Size()})
		}
	}

	data, err := json.Marshal(blobs)
	if err != nil {
		h.internalServerError(w, err)
		return
	}

	w.Header().Set("Content-Type", mimeTypeAPIV2)
	_, _ = w.Write(data)
}

// checkBlob tests whether a blob exists.
func (h *Handler) checkBlob(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("checkBlob()")
	}

	objectType, objectID := h.getObject(r.URL.Path)
	if objectType == "" || objectID == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type or id: %s", r.URL.Path))
		return
	}
	path := h.getObjectPath(objectType, objectID)

	st, err := os.Stat(path)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	w.Header().Add("Content-Length", fmt.Sprint(st.Size()))
}

// getBlob retrieves a blob from the repository.
func (h *Handler) getBlob(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("getBlob()")
	}

	objectType, objectID := h.getObject(r.URL.Path)
	if objectType == "" || objectID == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type or id: %s", r.URL.Path))
		return
	}

	if h.opt.WriteOnly && objectType == "data" {
		httpDefaultError(w, http.StatusForbidden)
		return
	}

	path := h.getObjectPath(objectType, objectID)

	file, err := os.Open(path)
	if err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		httpDefaultError(w, http.StatusNotFound)
		return
	}

	wc := datacounter.NewResponseWriterCounter(w)
	http.ServeContent(wc, r, "", time.Unix(0, 0), file)

	if err = file.Close(); err != nil {
		h.internalServerError(w, err)
		return
	}

	h.sendMetric(objectType, BlobRead, wc.Count())
}

// saveBlob saves a blob to the repository.
func (h *Handler) saveBlob(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("saveBlob()")
	}

	objectType, objectID := h.getObject(r.URL.Path)
	if objectType == "" || objectID == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type or id: %s", r.URL.Path))
		return
	}
	path := h.getObjectPath(objectType, objectID)

	_, err := os.Stat(path)
	if err == nil {
		httpDefaultError(w, http.StatusForbidden)
		return
	}
	if !os.IsNotExist(err) {
		h.internalServerError(w, err)
		return
	}

	tmpFn := filepath.Join(filepath.Dir(path), objectID+".rest-server-temp")
	tf, err := tempFile(tmpFn, h.opt.FileMode)
	if os.IsNotExist(err) {
		// the error is caused by a missing directory, create it and retry
		mkdirErr := os.MkdirAll(filepath.Dir(path), h.opt.DirMode)
		if mkdirErr != nil {
			log.Print(mkdirErr)
		} else {
			// try again
			tf, err = tempFile(tmpFn, h.opt.FileMode)
		}
	}
	if err != nil {
		h.internalServerError(w, err)
		return
	}

	// ensure this blob does not put us over the quota size limit (if there is one)
	outFile, errCode, err := h.wrapFileWriter(r, tf)
	if err != nil {
		if h.opt.Debug {
			log.Println(err)
		}
		httpDefaultError(w, errCode)
		return
	}

	var written int64

	if h.opt.NoVerifyUpload {
		// just write the file without checking the contents
		written, err = io.Copy(outFile, r.Body)
	} else {
		// calculate hash for current request
		hasher := sha256.New()
		written, err = io.Copy(outFile, io.TeeReader(r.Body, hasher))

		// reject if file content doesn't match file name
		if err == nil && hex.EncodeToString(hasher.Sum(nil)) != objectID {
			err = errFileContentDoesntMatchHash
		}
	}

	if err != nil {
		_ = tf.Close()
		_ = os.Remove(tf.Name())
		h.incrementRepoSpaceUsage(-written)
		if h.opt.Debug {
			log.Print(err)
		}
		var pathError *os.PathError
		if errors.As(err, &pathError) && (pathError.Err == syscall.ENOSPC ||
			pathError.Err == syscall.EDQUOT) {
			// The error is disk-related (no space left, no quota left),
			// notify the client using the correct HTTP status
			httpDefaultError(w, http.StatusInsufficientStorage)
		} else if errors.Is(err, errFileContentDoesntMatchHash) ||
			errors.Is(err, io.ErrUnexpectedEOF) ||
			errors.Is(err, http.ErrMissingBoundary) ||
			errors.Is(err, http.ErrNotMultipart) {
			// The error is connection-related, send a client-side HTTP status
			httpDefaultError(w, http.StatusBadRequest)
		} else {
			// Otherwise we have a different internal error, reply with
			// server-side HTTP status
			h.internalServerError(w, err)
		}
		return
	}

	if err := tf.Sync(); err != nil {
		_ = tf.Close()
		_ = os.Remove(tf.Name())
		h.incrementRepoSpaceUsage(-written)
		h.internalServerError(w, err)
		return
	}

	if err := tf.Close(); err != nil {
		_ = os.Remove(tf.Name())
		h.incrementRepoSpaceUsage(-written)
		h.internalServerError(w, err)
		return
	}

	if err := os.Rename(tf.Name(), path); err != nil {
		_ = os.Remove(tf.Name())
		h.incrementRepoSpaceUsage(-written)
		h.internalServerError(w, err)
		return
	}

	if err := syncDir(filepath.Dir(path)); err != nil {
		// Don't call os.Remove(path) as this is prone to race conditions with parallel upload retries
		h.internalServerError(w, err)
		return
	}

	h.sendMetric(objectType, BlobWrite, uint64(written))
}

// tempFile implements a custom version of ioutil.TempFile which allows modifying the file permissions
func tempFile(fn string, perm os.FileMode) (f *os.File, err error) {
	for i := 0; i < 10; i++ {
		name := fn + strconv.FormatInt(rand.Int63(), 10)
		f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if os.IsExist(err) {
			continue
		}
		break
	}
	return
}

func syncDir(dirname string) error {
	if runtime.GOOS == "windows" {
		// syncing a directory is not possible on windows
		return nil
	}

	dir, err := os.Open(dirname)
	if err != nil {
		return err
	}
	err = dir.Sync()
	if err != nil {
		_ = dir.Close()
		return err
	}
	return dir.Close()
}

// deleteBlob deletes a blob from the repository.
func (h *Handler) deleteBlob(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("deleteBlob()")
	}

	objectType, objectID := h.getObject(r.URL.Path)
	if objectType == "" || objectID == "" {
		h.internalServerError(w, fmt.Errorf(
			"cannot determine object type or id: %s", r.URL.Path))
		return
	}
	if h.opt.deleteDataForbidden() && objectType != "locks" {
		httpDefaultError(w, http.StatusForbidden)
		return
	}

	path := h.getObjectPath(objectType, objectID)

	var size int64
	if h.needSize() {
		stat, err := os.Stat(path)
		if err == nil {
			size = stat.Size()
		}
	}

	if err := os.Remove(path); err != nil {
		if h.opt.Debug {
			log.Print(err)
		}
		if os.IsNotExist(err) {
			httpDefaultError(w, http.StatusNotFound)
		} else {
			h.internalServerError(w, err)
		}
		return
	}

	h.incrementRepoSpaceUsage(-size)
	h.sendMetric(objectType, BlobDelete, uint64(size))
}

// createRepo creates repository directories.
func (h *Handler) createRepo(w http.ResponseWriter, r *http.Request) {
	if h.opt.Debug {
		log.Println("createRepo()")
	}

	if r.URL.Query().Get("create") != "true" {
		httpDefaultError(w, http.StatusBadRequest)
		return
	}

	log.Printf("Creating repository directories in %s\n", h.path)

	if err := os.MkdirAll(h.path, h.opt.DirMode); err != nil {
		h.internalServerError(w, err)
		return
	}

	for _, d := range ObjectTypes {
		if err := os.Mkdir(filepath.Join(h.path, d), h.opt.DirMode); err != nil && !os.IsExist(err) {
			h.internalServerError(w, err)
			return
		}
	}

	for i := 0; i < 256; i++ {
		dirPath := filepath.Join(h.path, "data", fmt.Sprintf("%02x", i))
		if err := os.Mkdir(dirPath, h.opt.DirMode); err != nil && !os.IsExist(err) {
			h.internalServerError(w, err)
			return
		}
	}
}

// internalServerError is called to repot an internal server error.
// The error message will be reported in the server logs. If PanicOnError
// is set, this will panic instead, which makes debugging easier.
func (h *Handler) internalServerError(w http.ResponseWriter, err error) {
	log.Printf("ERROR: %v", err)
	if h.opt.PanicOnError {
		panic(fmt.Sprintf("internal server error: %v", err))
	}
	httpDefaultError(w, http.StatusInternalServerError)
}
