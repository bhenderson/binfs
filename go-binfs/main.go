// Package main is a command line tool which can take a directory and create a binary file system
//
// See example.go for usage.
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

	"github.com/bhenderson/binfs"
)

var outTemplate = template.Must(template.New("out").
	Parse(`
package {{.PackageName}}

import "github.com/bhenderson/binfs"

func init() {
	var fs = binfs.GetFS("{{.Name}}")
	{{range .Files }}
	fs.Add({{.GenerateFields}}){{ end }}
}
`))

var data = struct {
	PackageName string
	Name        string
	Dir         string
	Output      string
	Files       []*binfs.FileStat
}{}

func main() {
	flag.StringVar(&data.PackageName, "packagename", "main", "package name")
	flag.StringVar(&data.Name, "name", "", "reference name")
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
		fs, err := binfs.Generate(path, info)
		fatal(err)
		data.Files = append(data.Files, fs)
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

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
