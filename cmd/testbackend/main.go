package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

func hello(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))
	log.Println("Replied with a hello message")
	time.Sleep(time.Duration(*sleep) * time.Microsecond)
	fmt.Fprintf(w, "Hello From Backend Server %s\n", *port)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))
	log.Println("Responding to health check")
	w.WriteHeader(http.StatusOK)
}

var port = flag.String("port", "", "port to run the HTTP server on")
var sleep = flag.Int("sleep", 0, "time to sleep before responding")

func main() {

	flag.Parse()

	http.HandleFunc("/", hello)
	http.HandleFunc("/healthcheck", healthCheck)
	err := http.ListenAndServe(fmt.Sprintf(":%s", *port), nil)
	if err != nil {
		fmt.Println(err)
	}
}
