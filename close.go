package etp

import (
	"errors"

	"nhooyr.io/websocket"
)

func IsNormalClose(err error) bool {
	wsError := websocket.CloseError{}
	if errors.As(err, &wsError) {
		return wsError.Code == websocket.StatusNormalClosure
	}
	return false
}
