package main

import (
	"io"
	"log"
	"net/http"

	"github.com/MohitPanchariya/loadbalancer/shared"
)

// Forward a http request to one of the healthy servers
func forward(w http.ResponseWriter, r *http.Request) {
	log.Print(shared.NewRequestInfo(r))
	forwardURL := "http://localhost:6000"

	req, err := http.NewRequest(r.Method, forwardURL, r.Body)
	if err != nil {
		log.Printf("Failed to forward request. Error creating the request: %s\n", err)
		http.Error(w, "Failed to forward request. Error creating the request.", http.StatusInternalServerError)
		return
	}

	// Copy all headers from original request
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to forward request. Error making http request: %s\n", err)
		http.Error(w, "Failed to forward request. Error making http request", http.StatusBadGateway)
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
	log.Printf("Response from server: %s %s\n", res.Proto, res.Status)
}

func main() {
	http.HandleFunc("/", forward)
	http.ListenAndServe(":80", nil)
}
