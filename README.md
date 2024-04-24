# etp

[![GoDoc](https://godoc.org/github.com/txix-open/etp/v3?status.svg)](https://godoc.org/github.com/txix-open/etp/v3)
![Build and test](https://github.com/txix-open/etp/actions/workflows/main.yml/badge.svg)
[![codecov](https://codecov.io/gh/txix-open/etp/branch/master/graph/badge.svg?token=JMTTJ5O6WB)](https://codecov.io/gh/txix-open/etp)
[![Go Report Card](https://goreportcard.com/badge/github.com/txix-open/etp/v3)](https://goreportcard.com/report/github.com/txix-open/etp/v3)

ETP - event transport protocol on WebSocket, simple and powerful.

Highly inspired by [socket.io](https://socket.io/). But there is no good implementation in Go, that is why `etp` exists.

The package based on [github.com/nhooyr/websocket](https://github.com/nhooyr/websocket).

## Install

```bash
go get -u github.com/txix-open/etp/v3
```

## Features:
- Rooms for server
- [Javascript client](https://github.com/txix-open/isp-etp-js-client)
- Concurrent event emitting
- Context based API
- Event acknowledgment (for sync communication)

## Internals
- WebSocket message
    - `websocket.MessageText` (not binary)
    - `<eventName>||<ackId>||<eventData>`
- Each event is handled concurrently in separated goroutine
- Message limit is `1MB` by default

## Complete example

```go
package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/txix-open/etp/v3"
	"github.com/txix-open/etp/v3/msg"
)

func main() {
	srv := etp.NewServer(etp.WithServerAcceptOptions(&etp.AcceptOptions{
		InsecureSkipVerify: true, //completely ignore CORS checks, enable only for dev purposes
	}))

	//callback to handle new connection
	srv.OnConnect(func(conn *etp.Conn) {
		//you have access to original HTTP request
		fmt.Printf("id: %d, url: %s, connected\n", conn.Id(), conn.HttpRequest().URL)
		srv.Rooms().Join(conn, "goodClients") //leave automatically then disconnected
	})

	//callback to handle disconnection
	srv.OnDisconnect(func(conn *etp.Conn, err error) {
		fmt.Println("disconnected", conn.Id(), err, etp.IsNormalClose(err))
	})

	//callback to handle any error during serving
	srv.OnError(func(conn *etp.Conn, err error) {
		connId := uint64(0)
		// be careful, conn can be nil on upgrading protocol error (before success WebSocket connection)
		if conn != nil {
			connId = conn.Id()
		}
		fmt.Println("error", connId, err)
	})

	//your business event handler
	//each event is handled in separated goroutine
	srv.On("hello", etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
		fmt.Printf("hello event received: %s, %s\n", event.Name, event.Data)
		return []byte("hello handled")
	}))

	//fallback handler
	srv.OnUnknownEvent(etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
		fmt.Printf("unknown event received, %s, %s\n", event.Name, event.Data)
		_ = conn.Close() //you may close a connection whenever you want
		return nil
	}))

	mux := http.NewServeMux()
	mux.Handle("GET /ws", srv)
	go func() {
		err := http.ListenAndServe(":8080", mux)
		if err != nil {
			panic(err)
		}
	}()
	time.Sleep(1 * time.Second)

	cli := etp.NewClient().
		OnDisconnect(func(conn *etp.Conn, err error) { //basically you have all handlers like a server here
			fmt.Println("client disconnected", conn.Id(), err)
		})
	err := cli.Dial(context.Background(), "ws://localhost:8080/ws")
	if err != nil {
		panic(err)
	}

	//to check connection is still alive
	err = cli.Ping(context.Background())
	if err != nil {
		panic(err)
	}

	//simply emit event
	err = cli.Emit(context.Background(), "hello", []byte("fire and forget"))
	if err != nil {
		panic(err)
	}

	//emit event and wait server response
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second) //highly recommended to set up timeout for waiting acknowledgment
	defer cancel()
	resp, err := cli.EmitWithAck(ctx, "hello", []byte("wait acknowledgment"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("hello resposne: '%s'\n", resp)

	//data can be empty
	err = cli.Emit(ctx, "unknown event", nil)
	if err != nil {
		panic(err)
	}

	time.Sleep(15 * time.Second)

	//call to disconnect all clients
	srv.Shutdown()
}
```

## V3 Migration

* Internal message format is the same as `v2`
* Each event now is handled in separated goroutine (completely async)
* Significantly reduce code base, removed redundant interfaces
* Fixed some memory leaks and potential deadlocks
* Main package `github.com/txix-open/isp-etp-go/v2` -> `github.com/txix-open/etp/v3`
* `OnDefault` -> `OnUnknownEvent`
* `On*` API are the same either `etp.Client` and `etp.Server`
* WAS

```go
srv.On("event", func (conn etp.Conn, data []byte) {
    log.Println("Received " + testEvent + ":" + string(data))
}).
```

* BECOME

```go
srv.On("hello", etp.HandlerFunc(func(ctx context.Context, conn *etp.Conn, event msg.Event) []byte {
    fmt.Printf("hello event received: %s, %s\n", event.Name, event.Data)
    return []byte("hello handled")
}))
```

The second param now is a interface.
