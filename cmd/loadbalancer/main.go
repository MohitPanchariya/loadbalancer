package main

import (
	"io"
	"log"
	"net/http"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

var servers []string = []string{
	"http://localhost:9000",
	"http://localhost:9001",
	"http://localhost:9002",
}

// The Scheduler schedules requests to the servers in a round-robin fashion
type Scheduler struct {
	counter int
}

type LoadBalancer struct {
	Scheduler *Scheduler
}

func (s *Scheduler) scheduleRequest(r *http.Request) (*http.Response, error) {
	server := servers[(s.counter % len(servers))]
	s.counter++

	// Create the request
	req, err := http.NewRequest(r.Method, server, r.Body)
	if err != nil {
		return nil, err
	}

	// Copy all headers from the original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Execute the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Forward a http request to one of the healthy servers
func (lb *LoadBalancer) forward(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))

	res, err := lb.Scheduler.scheduleRequest(r)

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

func main() {
	scheduler := Scheduler{
		counter: 0,
	}
	loadbalancer := LoadBalancer{
		Scheduler: &scheduler,
	}

	http.HandleFunc("/", loadbalancer.forward)
	http.ListenAndServe(":80", nil)
}
