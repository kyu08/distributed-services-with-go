package main

import (
	"log"

	"github.com/kyu08/distributed-services-with-go/internal/server"
)

func main() {
	srv := server.NewHTTPServer("127.0.0.1:8080")
	log.Fatal(srv.ListenAndServe())
}
