package client

import "net/http"

const (
	defaultWorkersNum              = 1
	defaultWorkersBufferMultiplier = 10
)

type Config struct {
	// By default, the connection has a message read limit of 32768 bytes.
	// When the limit is hit, the connection will be closed with StatusMessageTooBig.
	ConnectionReadLimit int64
	// May be nil
	HttpClient *http.Client
	// May be nil
	HttpHeaders http.Header
	// Default is defaultWorkersNum
	WorkersNum int
	// Default is defaultWorkersBufferMultiplier
	WorkersBufferMultiplier int
}
