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
	"go.uber.org/goleak"
	"nhooyr.io/websocket"
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
	return SetupTestClientWithConfig(address, client.Config{HttpClient: cl})
}

func SetupTestClientWithConfig(address string, config client.Config) client.Client {
	client := client.NewClient(config)
	err := client.Dial(context.TODO(), testUrl(address))
	if err != nil {
		log.Fatalln("dial error:", err)
	}
	return client
}

func testUrl(address string) string {
	address = strings.Replace(address, "http://", "ws://", 1)
	address = address + "/isp-etp/"
	return address
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
	defer goleak.VerifyNone(t)
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
	defer goleak.VerifyNone(t)
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

func TestServer_AckAndCloseOnConnect(t *testing.T) {
	defer goleak.VerifyNone(t)
	a := assert.New(t)
	const testEvent = "test_event"
	testEventData := []byte("testdata")

	clientEventsCh := make(chan int, 10)
	const (
		onConnect = iota + 1
		onMessage
		onDisconnect
	)

	server, httpServer := SetupTestServer()
	defer httpServer.Close()
	server.OnDefault(func(event string, conn Conn, data []byte) {
		a.Fail("OnDefault", string(data))
	})
	server.OnError(func(conn Conn, err error) {
		a.Fail("OnError", err)
	})
	server.OnConnect(func(conn Conn) {
		err := conn.Emit(context.Background(), testEvent, testEventData)
		a.NoError(err)
		err = conn.Emit(context.Background(), testEvent, testEventData)
		a.NoError(err)
		err = conn.Emit(context.Background(), testEvent, testEventData)
		a.NoError(err)
		err = conn.Close()
		a.NoError(err)
	})

	cli := client.NewClient(client.Config{HttpClient: httpServer.Client()})
	cli.OnConnect(func() {
		clientEventsCh <- onConnect
	})
	cli.OnDisconnect(func(err error) {
		clientEventsCh <- onDisconnect
	})
	cli.OnError(func(err error) {
		a.Fail("OnError", err)
	})
	cli.On(testEvent, func(data []byte) {
		time.Sleep(1 * time.Millisecond)
		clientEventsCh <- onMessage
	})

	err := cli.Dial(context.Background(), testUrl(httpServer.URL))
	a.NoError(err)

	a.EqualValues(onConnect, <-clientEventsCh)
	a.EqualValues(onMessage, <-clientEventsCh)
	a.EqualValues(onMessage, <-clientEventsCh)
	a.EqualValues(onMessage, <-clientEventsCh)
	a.EqualValues(onDisconnect, <-clientEventsCh)
	a.True(cli.Closed())
}

func TestServer_OnWithAck_ClosedConn(t *testing.T) {
	defer goleak.VerifyNone(t)
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
	defer goleak.VerifyNone(t)
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
	defer goleak.VerifyNone(t)
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

	a.NoError(wait(wg, time.Second))

	_, err := cli.EmitWithAck(context.Background(), testEvent, testEventData)
	closeErr := new(websocket.CloseError)
	a.True(errors.As(err, closeErr))
	a.Equal(websocket.StatusNormalClosure, closeErr.Code)

	a.EqualValues(disconnectedCount, 1)
	a.EqualValues(connectedCount, 1)
}

func TestClient_Emit(t *testing.T) {
	defer goleak.VerifyNone(t)
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
	defer goleak.VerifyNone(t)
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

func TestClientHandlersWorkers_SyncDefault(t *testing.T) {
	defer goleak.VerifyNone(t)
	clientHandlersWorkersTesting(t, 0, 0)
}

func TestClientHandlersWorkers_AsyncDefault(t *testing.T) {
	defer goleak.VerifyNone(t)
	startTime := time.Now()
	clientHandlersWorkersTesting(t, 3, 0)
	deltaTime := time.Now().Sub(startTime)
	maxDeltaTime := 550 * time.Millisecond
	if deltaTime > maxDeltaTime {
		t.Errorf("Asynchron handling by workers is not working: Time expected less: %v, got: %v", maxDeltaTime, deltaTime)
	}
}

func clientHandlersWorkersTesting(t *testing.T, WorkersNum, WorkersChanBufferMultiplier int) {
	testEvent := "test_event"
	testAckEvent := "test_ack_event"
	testDefaultEvent := "test_default_event"
	testAckResponseData := []byte("testdata_response")

	server, httpServer := SetupTestServer()
	defer httpServer.Close()

	var connect Conn
	connCh := make(chan Conn)
	server.OnConnect(func(c Conn) {
		connCh <- c
	})

	cli := SetupTestClientWithConfig(httpServer.URL, client.Config{
		HttpClient:              httpServer.Client(),
		WorkersNum:              WorkersNum,
		WorkersBufferMultiplier: WorkersChanBufferMultiplier,
	})
	defer cli.Close()

	cli.OnWithAck(testAckEvent, func(data []byte) []byte {
		time.Sleep(500 * time.Millisecond)
		return testAckResponseData
	})
	cli.On(testEvent, func(data []byte) {
		time.Sleep(500 * time.Millisecond)
	})
	cli.OnDefault(func(event string, data []byte) {
		time.Sleep(500 * time.Millisecond)
	})

	connect = <-connCh
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		resp, err := connect.EmitWithAck(context.Background(), testAckEvent, []byte(testAckEvent))
		if err != nil {
			t.Errorf("EmitWithAck was returned error: %v, with responce: %v", err, resp)
		} else if string(resp) != string(testAckResponseData) {
			t.Errorf("EmitWithAck was returned response: %v, but expected: %s", resp, testAckResponseData)
		}
	}()
	go func() {
		defer wg.Done()
		err := connect.Emit(context.Background(), testEvent, []byte(testEvent))
		if err != nil {
			t.Errorf("Emit with event: %s was returned error: %v", testEvent, err)
		}
	}()
	go func() {
		defer wg.Done()
		err := connect.Emit(context.Background(), testDefaultEvent, []byte(testDefaultEvent))
		if err != nil {
			t.Errorf("Emit with event: %s was returned error: %v", testDefaultEvent, err)
		}
	}()
	time.Sleep(100 * time.Millisecond)

	startTime := time.Now()
	if err := connect.Ping(context.Background()); err != nil {
		t.Errorf("Ping was returned error: %v, ", err)
	}
	deltaTime := time.Now().Sub(startTime)

	fmt.Println(deltaTime)
	if deltaTime > 10*time.Millisecond {
		t.Errorf("Ping Blocked")
	}

	wg.Wait()
}
