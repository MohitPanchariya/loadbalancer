package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

func hello(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))
	log.Println("Replied with a hello message")
	fmt.Fprintf(w, "Hello From Backend Server")
}

func main() {
	port := flag.String("port", "", "port to run the HTTP server on")
	flag.Parse()

	http.HandleFunc("/", hello)
	err := http.ListenAndServe(fmt.Sprintf(":%s", *port), nil)
	if err != nil {
		fmt.Println(err)
	}
}
