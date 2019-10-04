package main

import (
	"context"
	"errors"
	"github.com/integration-system/isp-etp-go"
	"log"
	"net/http"
	"nhooyr.io/websocket"
)

func main() {
	bindAddress := "127.0.0.1:7777"
	testEvent := "test_event"
	helloEvent := "hello"
	config := etp.ServerConfig{
		InsecureSkipVerify: true,
	}
	eventsServer := etp.NewServer(context.TODO(), config).
		OnConnect(func(conn etp.Conn) {
			log.Println("OnConnect", conn.ID())
		}).
		OnDisconnect(func(conn etp.Conn, err error) {
			log.Println("OnDisconnect id", conn.ID())
			log.Println("OnDisconnect err:", err)
		}).
		OnError(func(conn etp.Conn, err error) {
			log.Println("OnError err:", err)
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				log.Println(closeErr.Code)
			}
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

	mux := http.NewServeMux()
	mux.HandleFunc("/isp-etp/", eventsServer.ServeHttp)
	httpServer := &http.Server{Addr: bindAddress, Handler: mux}
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("Unable to start http server. err: %v", err)
		}
	}()

	closeCh := make(chan struct{})
	<-closeCh
}
