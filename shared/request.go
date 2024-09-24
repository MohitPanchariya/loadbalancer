package shared

import (
	"fmt"
	"net/http"
)

type RequsetInfo struct {
	remoteAddr string
	method     string
	path       string
	proto      string
	host       string
	userAgent  string
	accept     string
}

func NewRequestInfo(r *http.Request) *RequsetInfo {
	reqInfo := &RequsetInfo{
		remoteAddr: r.RemoteAddr,
		method:     r.Method,
		path:       r.URL.Path,
		proto:      r.Proto,
		host:       r.Host,
		userAgent:  r.UserAgent(),
		accept:     r.Header.Get("Accept"),
	}
	return reqInfo
}

func (reqInfo *RequsetInfo) String() string {
	return fmt.Sprintf("Request from: %s\n%s %s %s\nHost: %s\nUser-Agent: %s\nAccept: %s", reqInfo.remoteAddr, reqInfo.method, reqInfo.path, reqInfo.proto, reqInfo.host, reqInfo.userAgent, reqInfo.accept)
}
