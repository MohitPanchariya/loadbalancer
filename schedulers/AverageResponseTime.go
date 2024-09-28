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

// AverageResponseTime schedules requests to the servers with the least average response time
type AverageResponseTime struct {
	HealthyServerCount atomic.Int64     // Number of healthy servers
	Servers            []*shared.Server // List of servers
	ServerLock         sync.Mutex       // Used to lock the slice of servers
}

func NewAverageResponseTime(config *shared.Config) *AverageResponseTime {
	scheduler := &AverageResponseTime{
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
func (s *AverageResponseTime) ServerHealthCheck(period <-chan time.Time) {
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

// Returns the server with the least average response time
func (s *AverageResponseTime) leastResponseTime() *shared.Server {
	var server *shared.Server
	s.ServerLock.Lock()

	s.Servers[0].AverageResponseTimeLock.Lock()
	var averageResponseTime time.Duration = s.Servers[0].AverageResponseTime
	server = s.Servers[0]
	s.Servers[0].AverageResponseTimeLock.Unlock()

	for i := 1; i < int(s.HealthyServerCount.Load()); i++ {
		s.Servers[i].AverageResponseTimeLock.Lock()
		if s.Servers[i].AverageResponseTime < averageResponseTime {
			server = s.Servers[i]
		}
		s.Servers[i].AverageResponseTimeLock.Unlock()
	}

	s.ServerLock.Unlock()

	return server
}

// Forwards the http request to the server with the least response time
func (s *AverageResponseTime) ScheduleRequest(r *http.Request) (*http.Response, error) {
	var server *shared.Server = s.leastResponseTime()
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

	start := time.Now()
	// Execute the request
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	end := time.Since(start)

	// average the server's response time
	server.AverageResponseTimeLock.Lock()
	server.AverageResponseTime = (server.AverageResponseTime + end) / 2
	server.AverageResponseTimeLock.Unlock()

	fmt.Printf("Average response time: %v\n", server.AverageResponseTime)

	return res, nil
}
