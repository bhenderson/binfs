// +build ignore

package main

//go:generate go-binfs -dir /path/to/resources -out resources.go

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(binFS))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
