package shared

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Config struct {
	Servers   []string // List of servers
	Port      int      // Loadbalancer port
	Frequency int      // Health check frequency in seconds
	Algorithm string   // Load balancing algorithm
}

type Server struct {
	Addr                    string        // Adderss of the server
	Healthy                 bool          // Health status of the server
	AverageResponseTime     time.Duration // Average response time of the server
	AverageResponseTimeLock sync.Mutex
}

type RequsetInfo struct {
	RemoteAddr string
	Method     string
	Path       string
	Proto      string
	Host       string
	UserAgent  string
	Accept     string
}

func NewRequestInfo(r *http.Request) *RequsetInfo {
	reqInfo := &RequsetInfo{
		RemoteAddr: r.RemoteAddr,
		Method:     r.Method,
		Path:       r.URL.Path,
		Proto:      r.Proto,
		Host:       r.Host,
		UserAgent:  r.UserAgent(),
		Accept:     r.Header.Get("Accept"),
	}
	return reqInfo
}

func (reqInfo *RequsetInfo) String() string {
	return fmt.Sprintf("Request from: %s\n%s %s %s\nHost: %s\nUser-Agent: %s\nAccept: %s", reqInfo.RemoteAddr, reqInfo.Method, reqInfo.Path, reqInfo.Proto, reqInfo.Host, reqInfo.UserAgent, reqInfo.Accept)
}
