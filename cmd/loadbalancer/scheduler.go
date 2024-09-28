package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// The Scheduler schedules requests to the servers in a round-robin fashion
type Scheduler struct {
	counter            atomic.Int64 // Used to load balance between the healthy servers in a round robin fashion
	healthyServerCount int          // Number of healthy servers
	servers            []Server     // List of healthy servers
	serverLock         sync.Mutex   // Used to lock the slice of healthy servers
}

func NewScheduler(config *Config) *Scheduler {
	scheduler := &Scheduler{
		counter:            atomic.Int64{},
		servers:            make([]Server, len(config.Servers)),
		healthyServerCount: len(config.Servers),
	}
	for index, server := range config.Servers {
		scheduler.servers[index] = Server{
			addr:    server,
			healthy: true, // assume all serves are healthy initially
		}
	}
	return scheduler
}

// Performs a health check on s.Servers. If a server doesn't respond back
// with a 200 ok, requests are not forwarded to that server until it passes
// a health check.
func (s *Scheduler) ServerHealthCheck(period <-chan time.Time) {
	for range period {
		// Parallely perform health checks on all the servers.
		// Mark the unhealthy servers
		wg := sync.WaitGroup{}
		for i := 0; i < len(s.servers); i++ {
			wg.Add(1)
			go func() {
				res, err := http.Get(fmt.Sprintf("%s/healthcheck", s.servers[i].addr))
				if err != nil || res.Status != "200 OK" {
					s.servers[i].healthy = false
				} else {
					s.servers[i].healthy = true
				}
				wg.Done()
			}()
		}

		// Wait for all the go routines to finish the health check
		wg.Wait()

		unhealthyCounter := 0
		i := 0

		s.serverLock.Lock()
		for i < len(s.servers)-unhealthyCounter {
			if !s.servers[i].healthy {
				log.Printf("server: %s found to be unhealthy\n", s.servers[i].addr)
				unhealthyCounter++
				// Push the unhealthy server to the end
				s.servers[i], s.servers[len(s.servers)-unhealthyCounter] = s.servers[len(s.servers)-unhealthyCounter], s.servers[i]
			} else { // move onto next server if current server is healthy
				i++
			}
		}
		s.healthyServerCount = len(s.servers) - unhealthyCounter
		s.serverLock.Unlock()
	}
}

func (s *Scheduler) ScheduleRequest(r *http.Request) (*http.Response, error) {
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
