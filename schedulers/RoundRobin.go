package schedulers

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

// RoundRobinScheduler schedules requests to the servers in a round-robin fashion
type RoundRobinScheduler struct {
	Counter            atomic.Int64     // Used to load balance between the healthy servers in a round robin fashion
	HealthyServerCount atomic.Int64     // Number of healthy servers
	Servers            []*shared.Server // List of servers
	ServerLock         sync.Mutex       // Used to lock the slice of servers
}

func NewRoundRobinScheduler(config *shared.Config) *RoundRobinScheduler {
	scheduler := &RoundRobinScheduler{
		Counter:            atomic.Int64{},
		Servers:            make([]*shared.Server, len(config.Servers)),
		HealthyServerCount: atomic.Int64{},
	}
	scheduler.HealthyServerCount.Store(int64(len(config.Servers)))
	for index, server := range config.Servers {
		scheduler.Servers[index] = &shared.Server{
			Addr:    server,
			Healthy: true, // assume all serves are healthy initially
		}
	}
	return scheduler
}

// Performs a health check on s.Servers. If a server doesn't respond back
// with a 200 ok, requests are not forwarded to that server until it passes
// a health check.
func (s *RoundRobinScheduler) ServerHealthCheck(period <-chan time.Time) {
	for range period {
		// Parallely perform health checks on all the servers.
		// Mark the unhealthy servers
		wg := sync.WaitGroup{}
		for i := 0; i < len(s.Servers); i++ {
			wg.Add(1)
			go func() {
				res, err := http.Get(fmt.Sprintf("%s/healthcheck", s.Servers[i].Addr))
				if err != nil || res.Status != "200 OK" {
					s.Servers[i].Healthy = false
				} else {
					s.Servers[i].Healthy = true
				}
				wg.Done()
			}()
		}

		// Wait for all the go routines to finish the health check
		wg.Wait()

		unhealthyCounter := 0
		i := 0

		s.ServerLock.Lock()
		for i < len(s.Servers)-unhealthyCounter {
			if !s.Servers[i].Healthy {
				log.Printf("server: %s found to be unhealthy\n", s.Servers[i].Addr)
				unhealthyCounter++
				// Push the unhealthy server to the end
				s.Servers[i], s.Servers[len(s.Servers)-unhealthyCounter] = s.Servers[len(s.Servers)-unhealthyCounter], s.Servers[i]
			} else { // move onto next server if current server is healthy
				i++
			}
		}
		s.HealthyServerCount.Store(int64(len(s.Servers) - unhealthyCounter))
		s.ServerLock.Unlock()
	}
}

func (s *RoundRobinScheduler) ScheduleRequest(r *http.Request) (*http.Response, error) {
	// Aquire a lock on the slice of healthy servers
	s.ServerLock.Lock()
	// Pick the next healthy server
	server := s.Servers[(s.Counter.Load() % s.HealthyServerCount.Load())]
	s.Counter.Add(1)

	s.ServerLock.Unlock()

	// Create the request
	req, err := http.NewRequest(r.Method, server.Addr, r.Body)
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
