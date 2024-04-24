package example_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/txix-open/etp/v3"
	"github.com/txix-open/etp/v3/msg"
	"nhooyr.io/websocket"
)

func ExampleServer() {
	srv := etp.NewServer(etp.WithServerAcceptOptions(&websocket.AcceptOptions{
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
