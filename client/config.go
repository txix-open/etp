package client

import "net/http"

type Config struct {
	// By default, the connection has a message read limit of 32768 bytes.
	// When the limit is hit, the connection will be closed with StatusMessageTooBig.
	ConnectionReadLimit int64
	// May be nil
	HttpClient *http.Client
	// May be nil
	HttpHeaders http.Header
}
