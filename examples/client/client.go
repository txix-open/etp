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
