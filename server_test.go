package etp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/integration-system/isp-etp-go/v2/ack"
	"github.com/integration-system/isp-etp-go/v2/client"
	"github.com/stretchr/testify/assert"
)

func SetupTestServer() (Server, *httptest.Server) {
	config := ServerConfig{
		InsecureSkipVerify: true,
	}
	server := NewServer(context.TODO(), config)
	mux := http.NewServeMux()
	mux.HandleFunc("/isp-etp/", server.ServeHttp)
	httpServer := httptest.NewServer(mux)
	return server, httpServer
}

func SetupTestClient(address string, cl *http.Client) client.Client {
	address = strings.Replace(address, "http://", "ws://", 1)
	address = address + "/isp-etp/"
	config := client.Config{
		HttpClient: cl,
	}
	client := client.NewClient(config)
	err := client.Dial(context.TODO(), address)
	if err != nil {
		log.Fatalln("dial error:", err)
	}
	return client
}

func wait(wg *sync.WaitGroup, duration time.Duration) error {
	ch := make(chan struct{})
	go func() {
		wg.Wait()
		close(ch)
	}()

	select {
	case <-time.After(duration):
		return errors.New("waiting timeout exceeded")
	case <-ch:

	}
	return nil
}

func TestServer_On(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"
	testEventData := []byte("testdata")
	wg := new(sync.WaitGroup)
	wg.Add(1)

	server, httpServer := SetupTestServer()
	defer httpServer.Close()

	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer cli.Close()

	var receivedData []byte
	server.On(testEvent, func(conn Conn, data []byte) {
		defer wg.Done()
		receivedData = append(receivedData, data...)
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", conn, err)
	})

	err := cli.Emit(context.Background(), testEvent, testEventData)

	err2 := wait(wg, time.Second)
	a.NoError(err2)

	a.NoError(err)
	a.Equal(testEventData, receivedData)
}

func TestServer_OnWithAck(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"
	testEventData := []byte("testdata")
	testResponseData := []byte("testdata_response")

	server, httpServer := SetupTestServer()
	defer httpServer.Close()

	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer cli.Close()

	var receivedData []byte
	server.OnWithAck(testEvent, func(conn Conn, data []byte) []byte {
		receivedData = append(receivedData, data...)
		return testResponseData
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", err)
	})

	resp, err := cli.EmitWithAck(context.Background(), testEvent, testEventData)
	a.NoError(err)
	a.Equal(testEventData, receivedData)
	a.Equal(testResponseData, resp)
}

func TestServer_OnWithAck_ClosedConn(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"
	testEventData := []byte("testdata")
	testResponseData := []byte("testdata_response")

	server, httpServer := SetupTestServer()
	defer httpServer.Close()

	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer cli.Close()

	var receivedData []byte
	server.OnWithAck(testEvent, func(conn Conn, data []byte) []byte {
		receivedData = append(receivedData, data...)
		return testResponseData
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", err)
	})
	server.Close()
	resp, err := cli.EmitWithAck(context.Background(), testEvent, testEventData)

	a.Equal(err, ack.ErrConnClose)
	a.Equal([]byte(nil), receivedData)
	a.Equal([]byte(nil), resp)
}

func TestServer_OnDefault(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"
	testEvent2 := "test_event_2"
	testEventData := []byte("testdata")
	wg := new(sync.WaitGroup)
	wg.Add(1)

	server, httpServer := SetupTestServer()
	defer httpServer.Close()

	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer cli.Close()

	server.OnWithAck(testEvent, func(conn Conn, data []byte) []byte {
		a.Fail("OnWithAck", string(data))
		return nil
	})
	server.On(testEvent2, func(conn Conn, data []byte) {
		a.Fail("On", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", conn, err)
	})
	var receivedData []byte
	server.OnDefault(func(event string, conn Conn, data []byte) {
		receivedData = append(receivedData, data...)
		wg.Done()
	})

	err := cli.Emit(context.Background(), testEvent, testEventData)

	err2 := wait(wg, time.Second)
	a.NoError(err2)

	a.NoError(err)
	a.Equal(testEventData, receivedData)
}

func TestConn_Close(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"
	testEventData := []byte("testdata")
	wg := new(sync.WaitGroup)
	wg.Add(2)

	server, httpServer := SetupTestServer()
	var disconnectedCount int64 = 0
	var connectedCount int64 = 0
	server.OnConnect(func(conn Conn) {
		defer wg.Done()
		atomic.AddInt64(&connectedCount, 1)
		err := conn.Close()
		a.NoError(err)
	})
	server.OnDisconnect(func(conn Conn, err error) {
		defer wg.Done()
		atomic.AddInt64(&disconnectedCount, 1)
	})

	defer httpServer.Close()

	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer cli.Close()

	server.OnWithAck(testEvent, func(conn Conn, data []byte) []byte {
		a.Fail("OnWithAck", string(data))
		return nil
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", conn, err)
	})

	server.Close()
	_, err := cli.EmitWithAck(context.Background(), testEvent, testEventData)

	err2 := wait(wg, time.Second)
	a.NoError(err2)

	a.Equal(err, ack.ErrConnClose)
	a.EqualValues(disconnectedCount, 1)
	a.EqualValues(connectedCount, 1)
}

func TestClient_Emit(t *testing.T) {
	a := assert.New(t)
	testEventEmit := "test_event"
	testEventAck := "test_event_ack"
	testDataEmit := []byte("testdata")
	testDataAck := []byte("test ack data")
	testAckResponseData := []byte("testdata_response")
	wg := new(sync.WaitGroup)
	wg.Add(4)

	server, httpServer := SetupTestServer()
	defer httpServer.Close()
	cli := SetupTestClient(httpServer.URL, httpServer.Client())
	defer func() {
		err := cli.Close()
		a.NoError(err)
	}()
	defer server.Close()

	var serverReceivedDataAck []byte
	server.OnWithAck(testEventAck, func(conn Conn, data []byte) []byte {
		defer wg.Done()
		serverReceivedDataAck = append(serverReceivedDataAck, data...)
		err := conn.Emit(context.Background(), testEventEmit, testDataEmit)
		a.NoError(err)
		return testAckResponseData
	})
	var serverReceivedDataEmit []byte
	server.On(testEventEmit, func(conn Conn, data []byte) {
		serverReceivedDataEmit = append(serverReceivedDataEmit, data...)
		go func() {
			defer wg.Done()
			resp, err := conn.EmitWithAck(context.Background(), testEventAck, testDataAck)
			a.NoError(err)
			a.Equal(testAckResponseData, resp)
		}()
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", conn, err)
	})

	var clientReceivedDataAck []byte
	cli.OnWithAck(testEventAck, func(data []byte) []byte {
		defer wg.Done()
		clientReceivedDataAck = append(clientReceivedDataAck, data...)
		return testAckResponseData
	})
	var clientReceivedDataEmit []byte
	cli.On(testEventEmit, func(data []byte) {
		defer wg.Done()
		clientReceivedDataEmit = append(clientReceivedDataEmit, data...)
	})
	cli.OnDefault(func(event string, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	cli.OnError(func(err error) {
		a.Fail("OnError", err)
	})

	resp, err := cli.EmitWithAck(context.Background(), testEventAck, testDataAck)
	a.NoError(err)
	a.Equal(testAckResponseData, resp)
	err = cli.Emit(context.Background(), testEventEmit, testDataEmit)
	a.NoError(err)

	err2 := wait(wg, time.Second)
	a.NoError(err2)

	a.EqualValues(testDataAck, clientReceivedDataAck)
	a.EqualValues(testDataAck, serverReceivedDataAck)

	a.EqualValues(testDataEmit, serverReceivedDataEmit)
	a.EqualValues(testDataEmit, clientReceivedDataEmit)
}

func TestConn_ManyConcurrentWrites(t *testing.T) {
	a := assert.New(t)
	testEvent := "test_event"

	wg := new(sync.WaitGroup)
	messagesNumber := 2000
	var finishedMessages int64 = 0
	wg.Add(messagesNumber)

	server, httpServer := SetupTestServer()
	defer server.Close()
	defer httpServer.Close()

	server.OnWithAck(testEvent, func(conn Conn, data []byte) []byte {
		response := make([]byte, len(data))
		copy(response, data)
		return response
	})
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", conn, err)
	})

	cli1 := SetupTestClient(httpServer.URL, &http.Client{Transport: &http.Transport{}})
	defer cli1.Close()
	cli1.OnError(func(err error) {
		a.NoError(err, "cli1 onError")
	})
	cli2 := SetupTestClient(httpServer.URL, &http.Client{Transport: &http.Transport{}})
	defer cli2.Close()
	cli2.OnError(func(err error) {
		a.NoError(err, "cli2 onError")
	})

	for i := 0; i < messagesNumber; i++ {
		go func(i int) {
			defer func() {
				atomic.AddInt64(&finishedMessages, 1)
				wg.Done()
			}()
			msg := fmt.Sprintf("msg-%d", i)
			var cli client.Client
			if i%2 == 0 {
				cli = cli1
			} else {
				cli = cli2
			}
			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			response, err := cli.EmitWithAck(ctx, testEvent, []byte(msg))

			a.NoError(err)
			a.Equal(string(response), msg, "from %d client", i%2)
		}(i)
	}

	err2 := wait(wg, 5*time.Second)
	a.NoError(err2)

	a.EqualValues(atomic.LoadInt64(&finishedMessages), messagesNumber)
}
