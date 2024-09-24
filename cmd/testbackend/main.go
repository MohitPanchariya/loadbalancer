package main

import (
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
	http.HandleFunc("/", hello)
	err := http.ListenAndServe(":6000", nil)
	if err != nil {
		fmt.Println(err)
	}
}
