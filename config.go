package etp

import "net/http"

type ServerConfig struct {
	// Can be used to define custom CORS policy.
	RequestHandler func(*http.Request) error
	// By default, the connection has a message read limit of 32768 bytes.
	// When the limit is hit, the connection will be closed with StatusMessageTooBig.
	ConnectionReadLimit int64
	// Checks origin header to prevent CSRF attack.
	InsecureSkipVerify bool
}
