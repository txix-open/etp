package main

import (
	"context"
	"errors"
	etpclient "github.com/integration-system/isp-etp-go/client"
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
		}).
		OnError(func(err error) {
			log.Println("OnError err:", err)
			var closeErr websocket.CloseError
			if errors.As(err, &closeErr) {
				log.Println(closeErr.Code)
			}
		})
	client.On(testEvent, func(data []byte) {
		log.Printf("Received %s:%s\n", testEvent, string(data))
	})
	err := client.Dial(context.TODO(), address)
	if err != nil {
		log.Fatalln("dial error:", err)
	}

	err = client.Emit(context.TODO(), helloEvent, []byte("hello"))
	if err != nil {
		log.Fatalln("emit error:", err)
	}

	closeCh := make(chan struct{})
	<-closeCh
}
