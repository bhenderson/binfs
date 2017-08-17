package binfs

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	_ http.FileSystem = FileSystem{}
	_ http.File       = &fileStat{}
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
	fs := &fileStat{
		name:  name,
		size:  size,
		mode:  mode,
		isDir: isDir,
	}

	fs.modTime.UnmarshalBinary(modTimeBinary)
	fs.Reader = *bytes.NewReader(data)
	s.add(path, fs)
}

func (s FileSystem) add(name string, fs *fileStat) {
	s[name] = fs
	dir := path.Dir(name)
	if v, ok := s[dir]; ok {
		d := v.(*fileStat)
		d.files = append(d.files, fs)
	}
}

type fileStat struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
	files   []os.FileInfo
	readdir int
	bytes.Reader
}

func (fs *fileStat) Name() string               { return fs.name }
func (fs *fileStat) Size() int64                { return fs.size }
func (fs *fileStat) Mode() os.FileMode          { return fs.mode }
func (fs *fileStat) ModTime() time.Time         { return fs.modTime }
func (fs *fileStat) IsDir() bool                { return fs.isDir }
func (fs *fileStat) Sys() interface{}           { return "binfs" }
func (fs *fileStat) Close() error               { return nil }
func (fs *fileStat) Stat() (os.FileInfo, error) { return fs, nil }

func (fs *fileStat) Readdir(n int) ([]os.FileInfo, error) {
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

func MustHexDecode(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func marshalTime(tp *time.Time, buf []byte) {
}

func matchDir(dir, name string) bool {
	return false
}
