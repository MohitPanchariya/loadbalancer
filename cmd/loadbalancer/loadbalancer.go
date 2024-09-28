package main

import (
	"io"
	"log"
	"net/http"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

type LoadBalancer struct {
	Scheduler Scheduler
}

// Forward a http request to one of the healthy servers
func (lb *LoadBalancer) forward(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))

	res, err := lb.Scheduler.ScheduleRequest(r)

	if err != nil {
		log.Printf("Failed to schedule request. Error: %s\n", err)
		http.Error(w, "Failed to schedule request.", http.StatusInternalServerError)
		return
	}

	// Copy all headers from the response
	for key, values := range res.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set the status code
	w.WriteHeader(res.StatusCode)

	defer res.Body.Close()

	// Copy the response body
	_, err = io.Copy(w, res.Body)

	if err != nil {
		log.Printf("Failed to copy response body: %s\n", err)
		http.Error(w, "Failed to copy response body.", http.StatusInternalServerError)
		return
	}

	// log the response status
	log.Printf("Response from %s: %s %s\n", res.Request.Host, res.Proto, res.Status)
}
