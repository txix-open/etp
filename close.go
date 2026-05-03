package etp

import (
	"errors"

	"github.com/coder/websocket"
)

// IsNormalClose checks if the error is a WebSocket normal closure.
func IsNormalClose(err error) bool {
	if wsError, ok := errors.AsType[websocket.CloseError](err); ok {
		return wsError.Code == websocket.StatusNormalClosure
	}
	return false
}
