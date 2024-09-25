package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

type Server struct {
	addr    string // Adderss of the server
	healthy bool   // Health status of the server
}

// The Scheduler schedules requests to the servers in a round-robin fashion
type Scheduler struct {
	counter            atomic.Int64 // Used to load balance between the healthy servers in a round robin fashion
	healthyServerCount int          // Number of healthy servers
	servers            []Server     // List of healthy servers
	serverLock         sync.Mutex   // Used to lock the slice of healthy servers
}

type LoadBalancer struct {
	Scheduler *Scheduler
}

// Performs a health check on s.Servers. If a server doesn't respond back
// with a 200 ok, requests are not forwarded to that server until it passes
// a health check.
func (s *Scheduler) serverHealthCheck(period <-chan time.Time) {
	for range period {
		unhealthyCounter := 0
		i := 0

		s.serverLock.Lock()
		for i < len(s.servers)-unhealthyCounter {
			res, err := http.Get(fmt.Sprintf("%s/healthcheck", s.servers[i].addr))
			if err != nil || res.Status != "200 OK" {
				log.Printf("server: %s found to be unhealthy\n", s.servers[i].addr)
				unhealthyCounter++
				s.servers[i].healthy = false
				// Push the unhealthy server to the end
				s.servers[i], s.servers[len(s.servers)-unhealthyCounter] = s.servers[len(s.servers)-unhealthyCounter], s.servers[i]
			} else { // move onto next server if current server is healthy
				s.servers[i].healthy = true
				i++
			}
		}
		s.healthyServerCount = len(s.servers) - unhealthyCounter
		s.serverLock.Unlock()
	}
}

func (s *Scheduler) scheduleRequest(r *http.Request) (*http.Response, error) {
	// Aquire a lock on the slice of healthy servers
	s.serverLock.Lock()
	// Pick the next healthy server
	server := s.servers[(s.counter.Load() % int64(s.healthyServerCount))]
	s.counter.Add(1)

	s.serverLock.Unlock()

	// Create the request
	req, err := http.NewRequest(r.Method, server.addr, r.Body)
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
		counter: atomic.Int64{},
		servers: []Server{
			{
				addr:    "http://localhost:9000",
				healthy: true,
			},
			{
				addr:    "http://localhost:9001",
				healthy: true,
			},
			{
				addr:    "http://localhost:9002",
				healthy: true,
			},
		},
		healthyServerCount: 3,
	}

	loadbalancer := LoadBalancer{
		Scheduler: &scheduler,
	}

	ticker := time.NewTicker(10 * time.Second)
	// perform health checks on the servers every 10 seconds
	go scheduler.serverHealthCheck(ticker.C)

	http.HandleFunc("/", loadbalancer.forward)
	http.ListenAndServe(":80", nil)

}
