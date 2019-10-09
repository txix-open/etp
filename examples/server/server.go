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
