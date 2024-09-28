package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/MohitPanchariya/loadbalancer/schedulers"
	"github.com/MohitPanchariya/loadbalancer/shared"
	"gopkg.in/yaml.v3"
)

type Scheduler interface {
	ServerHealthCheck(period <-chan time.Time)
	ScheduleRequest(r *http.Request) (*http.Response, error)
}

// Returns a scheduler that implements the algorithm specified in the config
func getScheduler(config *shared.Config) (Scheduler, error) {
	algorithm := config.Algorithm
	fmt.Println(algorithm)
	switch algorithm {
	case "roundrobin":
		return schedulers.NewRoundRobinScheduler(config), nil
	case "averageresponsetime":
		return schedulers.NewAverageResponseTime(config), nil
	default:
		return nil, fmt.Errorf("%s scheduler not found", algorithm)
	}
}

func main() {
	configPath := flag.String("config", "./config.yml", "path to config file")
	flag.Parse()

	fd, err := os.Open(*configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("Config file doesn't exist at provided path")
		} else {
			log.Fatalf("Failed to open config file.\n %s", err)
		}
	}
	defer fd.Close()

	config := &shared.Config{}
	decoder := yaml.NewDecoder(fd)
	err = decoder.Decode(config)
	if err != nil {
		log.Fatalf("Error unmarshalling config.\n %s", err)
	}

	scheduler, err := getScheduler(config)

	if err != nil {
		log.Fatalf("Failed to get scheduler.\n%s\n", err)
	}

	loadbalancer := LoadBalancer{
		Scheduler: scheduler,
	}
	ticker := time.NewTicker(time.Duration(config.Frequency) * time.Second)
	// perform health checks on the servers every 10 seconds
	go scheduler.ServerHealthCheck(ticker.C)

	http.HandleFunc("/", loadbalancer.forward)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
