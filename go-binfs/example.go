// +build ignore

package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(binFS))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
