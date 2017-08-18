package binfs

import (
	"bytes"
	"compress/gzip"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	_ http.FileSystem = FileSystem{}
	_ http.File       = &FileStat{}
	_ os.FileInfo     = &FileStat{}
)

type FileSystem map[string]http.File

func NewFS() FileSystem {
	return make(FileSystem)
}

// Open implements http.FileSystem
func (s FileSystem) Open(name string) (http.File, error) {
	// modified from http/fs.go (Dir).Open
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) ||
		strings.Contains(name, "\x00") {
		return nil, errors.New("http: invalid character in file path")
	}
	dir := "."
	fullPath := filepath.Join(dir, filepath.FromSlash(path.Clean("/"+name)))
	f, ok := s[fullPath]
	if !ok {
		return nil, os.ErrNotExist
	}
	return f, nil
}

// Add creates a new http.File from the parameters and adds it to FileSystem
func (s FileSystem) Add(path string, name string, size int64, mode os.FileMode, modTimeBinary []byte, isDir bool, data []byte) {
	fs := &FileStat{
		name:  name,
		size:  size,
		mode:  mode,
		isDir: isDir,
	}

	fs.modTime.UnmarshalBinary(modTimeBinary)
	fs.Reader = *bytes.NewReader(inflate(data))
	s.add(path, fs)
}

// Generate produces a FileStat. Useful for templates. Complement of Add.
func Generate(filePath string, info os.FileInfo) (*FileStat, error) {
	fs := &FileStat{
		path:    filePath,
		name:    info.Name(),
		size:    info.Size(),
		mode:    info.Mode(),
		modTime: info.ModTime(),
		isDir:   info.IsDir(),
	}

	if !fs.isDir {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}
		buf, err := ioutil.ReadAll(file)
		if err != nil {
			return nil, err
		}
		fs.Reader = *bytes.NewReader(buf)
	}

	return fs, nil
}

func inflate(data []byte) []byte {
	gr, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	buf, err := ioutil.ReadAll(gr)
	if err != nil {
		panic(err)
	}
	return buf
}

func (s FileSystem) add(name string, fs *FileStat) {
	s[name] = fs
	dir := path.Dir(name)
	if v, ok := s[dir]; ok {
		d := v.(*FileStat)
		d.files = append(d.files, fs)
	}
}

// FileStat is the representation of a file in memory. It implements http.File
// as well as os.FileInto, among others.
type FileStat struct {
	path    string
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	files   []os.FileInfo
	readdir int
	bytes.Reader
}

func (fs *FileStat) Name() string               { return fs.name }
func (fs *FileStat) Size() int64                { return fs.size }
func (fs *FileStat) Mode() os.FileMode          { return fs.mode }
func (fs *FileStat) ModTime() time.Time         { return fs.modTime }
func (fs *FileStat) IsDir() bool                { return fs.isDir }
func (fs *FileStat) Sys() interface{}           { return "binfs" }
func (fs *FileStat) Close() error               { return nil }
func (fs *FileStat) Stat() (os.FileInfo, error) { return fs, nil }

func (fs *FileStat) Readdir(n int) ([]os.FileInfo, error) {
	if !fs.isDir {
		return nil, os.ErrInvalid
	}

	res := fs.files
	if n < 0 || n > len(res) {
		n = len(res)
	}

	var s int
	s, fs.readdir = fs.readdir, n
	res = res[s:n]

	if len(res) == 0 && len(fs.files) > 0 {
		return nil, io.EOF
	}
	return res, nil
}

// GenerateFields produces a string compatible with the parameters for Add.
// Useful in templates.
func (fs *FileStat) GenerateFields() (string, error) {
	tb, e := fs.modTime.MarshalBinary()
	if e != nil {
		return "", e
	}
	cb, e := fs.compressedBytes()
	if e != nil {
		return "", e
	}

	format := `"%s", "%s", %d, 0%o, binfs.MustHexDecode("%x"), %t, binfs.MustHexDecode("%x")`
	return fmt.Sprintf(format,
		fs.path,
		fs.name,
		fs.size,
		fs.mode,
		tb,
		fs.isDir,
		cb,
	), nil
}

func (fs *FileStat) compressedBytes() ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	b, _ := ioutil.ReadAll(fs)
	_, err := gw.Write(b)
	gw.Close()
	return buf.Bytes(), err
}

// MustHexDecode is a helper function, useful in templates to convert a hex
// string to bytes.
func MustHexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
