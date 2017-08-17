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
	_ http.FileSystem = fileSystem{}
	_ http.File       = &fileStat{}
)

type fileSystem map[string]*fileStat

func NewFS() fileSystem {
	return make(fileSystem)
}

func (s fileSystem) Open(name string) (http.File, error) {
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

func (s fileSystem) Add(path string, name string, size int64, mode os.FileMode, modTime string, isDir bool, data string) {
	fs := &fileStat{
		name:  name,
		size:  size,
		mode:  mode,
		isDir: isDir,
	}

	marshalTime(&fs.modTime, modTime)

	var buf []byte
	marshalBytes(&buf, data)

	fs.Reader = *bytes.NewReader(buf)

	s[path] = fs

	s.addDir(path)
}

func (s fileSystem) addDir(name string) {
	dir := path.Dir(name)
	if d, ok := s[dir]; ok {
		d.files = append(d.files, s[name])
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

func marshalBytes(bp *[]byte, s string) {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	*bp = b
}

func marshalTime(tp *time.Time, s string) {
	var buf []byte
	marshalBytes(&buf, s)
	tp.UnmarshalBinary(buf)
}

func matchDir(dir, name string) bool {
	return false
}
