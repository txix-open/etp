# isp-etp-go

[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/txix-open/isp-etp-go?color=6b9ded&sort=semver)](https://github.com/txix-open/isp-etp-go/releases)
[![GoDoc](https://godoc.org/github.com/txix-open/isp-etp-go?status.svg)](https://godoc.org/github.com/txix-open/isp-etp-go)

Client and server implementation of event transport protocol.
Websocket framework based on library [github.com/nhooyr/websocket](https://github.com/nhooyr/websocket).

## Install

```bash
go get github.com/txix-open/isp-etp-go/v2
```

## Features:
- Rooms, broadcasting
- Store data object for connection
- [Javascript client](https://github.com/txix-open/isp-etp-js-client)
- Concurrent message write
- Context based requests

## Server example
```go
package main

import (
	"context"
	"errors"
	"github.com/txix-open/isp-etp-go/v2"
	"log"
	"net/http"
	"nhooyr.io/websocket"
)

func main() {
	bindAddress := "127.0.0.1:7777"
	testEvent := "test_event"
	helloEvent := "hello"
	unknownEvent := "unknown"

	config := etp.ServerConfig{
		InsecureSkipVerify: true,
	}
	server := etp.NewServer(context.TODO(), config).
		OnConnect(func(conn etp.Conn) {
			log.Println("OnConnect", conn.ID())
			err := conn.Emit(context.TODO(), unknownEvent, []byte("qwerty"))
			log.Println("unknown answer err:", err)
		}).
		OnDisconnect(func(conn etp.Conn, err error) {
			log.Println("OnDisconnect id", conn.ID())
			log.Println("OnDisconnect err:", err)
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				log.Println("OnDisconnect close code:", closeErr.Code)
			}
		}).
		OnError(func(conn etp.Conn, err error) {
			log.Println("OnError err:", err)
			if conn != nil {
				log.Println("OnError conn ID:", conn.ID())
			}
		}).
		On(testEvent, func(conn etp.Conn, data []byte) {
			log.Println("Received " + testEvent + ":" + string(data))
		}).
		On(helloEvent, func(conn etp.Conn, data []byte) {
			log.Println("Received " + helloEvent + ":" + string(data))
			answer := "hello, " + conn.ID()
			err := conn.Emit(context.TODO(), testEvent, []byte(answer))
			log.Println("hello answer err:", err)
		})
	// used as alternative to wildcards and custom handling
	server.OnDefault(func(event string, conn etp.Conn, data []byte) {
		log.Printf("Received default %s:%s\n", event, string(data))
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/isp-etp/", server.ServeHttp)
	httpServer := &http.Server{Addr: bindAddress, Handler: mux}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Unable to start http server. err: %v", err)
		}
	}()

	<-make(chan struct{})
}
```

## Client example
```go
package main

import (
	"context"
	"errors"
	etpclient "github.com/txix-open/isp-etp-go/v2/client"
	"log"
	"net/http"
	"nhooyr.io/websocket"
)

func main() {
	address := "ws://127.0.0.1:7777/isp-etp/"
	testEvent := "test_event"
	helloEvent := "hello"
	config := etpclient.Config{
		HttpClient: http.DefaultClient,
	}
	client := etpclient.NewClient(config).
		OnConnect(func() {
			log.Println("Connected")
		}).
		OnDisconnect(func(err error) {
			log.Println("OnDisconnect err:", err)
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				log.Println("OnDisconnect close code:", closeErr.Code)
			}
		}).
		OnError(func(err error) {
			log.Println("OnError err:", err)
		})
	client.On(testEvent, func(data []byte) {
		log.Printf("Received %s:%s\n", testEvent, string(data))
	})
	// used as alternative to wildcards and custom handling
	client.OnDefault(func(event string, data []byte) {
		log.Printf("Received default %s:%s\n", event, string(data))
	})
	err := client.Dial(context.TODO(), address)
	if err != nil {
		log.Fatalln("dial error:", err)
	}

	err = client.Emit(context.TODO(), helloEvent, []byte("hello"))
	if err != nil {
		log.Fatalln("emit error:", err)
	}

	<-make(chan struct{})
}

```

