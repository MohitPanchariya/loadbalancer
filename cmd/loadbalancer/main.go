package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Servers   []string // List of servers
	Port      int      // Loadbalancer port
	Frequency int      // Health check frequency in seconds
}

type Server struct {
	addr    string // Adderss of the server
	healthy bool   // Health status of the server
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

	config := &Config{}
	decoder := yaml.NewDecoder(fd)
	err = decoder.Decode(config)
	if err != nil {
		log.Fatalf("Error unmarshalling config.\n %s", err)
	}

	scheduler := NewScheduler(config)
	loadbalancer := LoadBalancer{
		Scheduler: scheduler,
	}
	ticker := time.NewTicker(time.Duration(config.Frequency) * time.Second)
	// perform health checks on the servers every 10 seconds
	go scheduler.ServerHealthCheck(ticker.C)

	http.HandleFunc("/", loadbalancer.forward)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
