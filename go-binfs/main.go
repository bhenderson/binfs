package main

import (
	"bytes"
	"flag"
	"go/format"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"text/template"
)

var outTemplate = template.Must(template.New("out").
	Parse(`
package {{.PackageName}}

import "github.com/bhenderson/binfs"

var binFS = binfs.NewFS()

func init() { {{range .Files }}
	binFS.Add(
		"{{.Path}}",
		"{{.Name}}",
		{{.Size}},
		{{printf "0%o" .Mode}},
		"{{.ModTime.MarshalBinary | printf "%x"}}",
		{{.IsDir}},
		"{{.Bytes | printf "%x"}}",
	){{ end }}
}
`))

var data = struct {
	PackageName string
	Dir         string
	Output      string
	Files       []FileStat
}{}

func main() {
	flag.StringVar(&data.PackageName, "packagename", "main", "package name")
	flag.StringVar(&data.Dir, "dir", ".", "directory to get files")
	flag.StringVar(&data.Output, "out", "binfs.go", "name of output file")
	flag.Parse()

	var err error
	if data.Output != "-" {
		data.Output, err = filepath.Abs(data.Output)
		fatal(err)
	}

	err = os.Chdir(data.Dir)
	fatal(err)

	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		data.Files = append(data.Files, copyFileStat(path, info))
		return nil
	})

	var buf bytes.Buffer

	err = outTemplate.Execute(&buf, data)
	fatal(err)

	out, err := format.Source(buf.Bytes())
	if err != nil {
		io.Copy(os.Stdout, &buf)
		panic(err)
	}

	if data.Output == "-" {
		os.Stdout.Write(out)
		return
	}

	ioutil.WriteFile(data.Output, out, 0666)
}

type FileStat struct {
	Path string
	os.FileInfo
	Bytes []byte
}

func copyFileStat(filePath string, info os.FileInfo) FileStat {
	fs := FileStat{
		Path:     filePath,
		FileInfo: info,
	}

	if fs.IsDir() {
		return fs
	}

	file, err := os.Open(filePath)
	if err != nil {
		panic(err)
	}
	fs.Bytes, err = ioutil.ReadAll(file)
	if err != nil {
		panic(err)
	}

	return fs
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
