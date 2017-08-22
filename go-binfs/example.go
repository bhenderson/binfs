// +build ignore

package main

//go:generate go-binfs -dir /path/to/resources -out resources.go

import (
	"log"
	"net/http"

	"github.com/bhenderson/binfs"
)

func main() {
	http.Handle("/", http.FileServer(binfs.GetFS("")))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
