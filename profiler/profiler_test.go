package profiler

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	. "gopkg.in/check.v1"

	"github.com/longhorn/types/pkg/generated/profilerrpc"

	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/utils"
)

const (
	opSHOW    = "SHOW"
	opENABLE  = "ENABLE"
	opDISABLE = "DISABLE"
)

func TestProfilerServiceOperations(t *testing.T) {
	profilerPort := availableProfilerPort(t)
	testCases := map[string]struct {
		op         string
		portNumber int32
		expectRet  bool
	}{
		"Show": {
			op:         opSHOW,
			portNumber: 0,
			expectRet:  true,
		},
		"Enable/Disable": {
			op:         opENABLE,
			portNumber: profilerPort,
			expectRet:  true,
		},
		"Invalidate op": {
			op:         "INVALID",
			portNumber: 0,
			expectRet:  false,
		},
	}

	client := server(t)

	for testName, testCase := range testCases {
		t.Run(testName, func(t *testing.T) {
			if !testCase.expectRet {
				_, err := client.ProfilerOP(testCase.op, testCase.portNumber)
				assert.Error(t, err, Commentf(test.ErrResultFmt, testName))
				return
			}

			_, err := client.ProfilerOP(testCase.op, testCase.portNumber)
			if !assert.NoError(t, err, Commentf(test.ErrResultFmt, testName)) {
				return
			}

			if testCase.op == opENABLE {
				assert.True(t, connected(testCase.portNumber), Commentf(test.ErrResultFmt, testName))
				_, err := client.ProfilerOP(opDISABLE, testCase.portNumber)
				assert.NoError(t, err, Commentf(test.ErrResultFmt, testName))
				assert.False(t, connected(testCase.portNumber), Commentf(test.ErrResultFmt, testName))
			}
		})
	}
}

func TestProfilerRepeatedEnableDisable(t *testing.T) {
	profiler := newProfilerServer(t)
	port := availableProfilerPort(t)

	for range 5 {
		_, err := profiler.EnableProfiler(port)
		if !assert.NoError(t, err) {
			return
		}

		_, err = profiler.DisableProfiler()
		if !assert.NoError(t, err) {
			return
		}
	}

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, listener.Close())
}

func TestProfilerDisableTimeoutCleansUp(t *testing.T) {
	profiler := newProfilerServer(t)

	listener, err := net.Listen("tcp", ":0")
	if !assert.NoError(t, err) {
		return
	}
	port := int32(listener.Addr().(*net.TCPAddr).Port)
	profilerAddress := fmt.Sprintf(":%d", port)

	handlerStarted := make(chan struct{})
	handlerDone := make(chan struct{})
	releaseHandler := make(chan struct{})
	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseHandler)
		})
	}

	httpServer := &http.Server{
		Addr: profilerAddress,
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			close(handlerStarted)
			<-releaseHandler
			close(handlerDone)
		}),
	}
	profiler.lock.Lock()
	profiler.server = httpServer
	profiler.listener = listener
	profiler.lock.Unlock()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.Serve(listener)
	}()

	transport := &http.Transport{DisableKeepAlives: true}
	requestCtx, cancelRequest := context.WithCancel(t.Context())
	client := http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}
	requestDone := make(chan struct{})
	var requestErr error
	t.Cleanup(func() {
		cancelRequest()
		release()
		transport.CloseIdleConnections()
		select {
		case <-requestDone:
		case <-time.After(time.Second):
			assert.Fail(t, "timed out waiting for profiler request")
		}
		select {
		case <-handlerDone:
		case <-time.After(time.Second):
			assert.Fail(t, "timed out waiting for profiler handler")
		}
		_ = httpServer.Close()
		select {
		case err := <-serveErr:
			if err != nil && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, net.ErrClosed) {
				assert.Fail(t, fmt.Sprintf("failed to stop profiler server: %v", err))
			}
		case <-time.After(time.Second):
			assert.Fail(t, "timed out waiting for profiler server")
		}
	})

	go func() {
		request, err := http.NewRequestWithContext(
			requestCtx,
			http.MethodGet,
			fmt.Sprintf("http://127.0.0.1:%d/", port),
			nil,
		)
		if err != nil {
			requestErr = err
			close(requestDone)
			return
		}
		response, err := client.Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		requestErr = err
		close(requestDone)
	}()

	select {
	case <-handlerStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for profiler handler")
	}

	ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(-time.Second))
	defer cancel()
	profiler.lock.Lock()
	_, shutdownErr := profiler.disableProfilerLocked(ctx)
	profiler.lock.Unlock()
	if !assert.ErrorIs(t, shutdownErr, context.DeadlineExceeded) {
		return
	}

	shownAddress, showErr := profiler.ShowProfiler()
	if !assert.NoError(t, showErr) {
		return
	}
	assert.Equal(t, shutdownErr.Error(), shownAddress)
	assert.NotEqual(t, profilerAddress, shownAddress)

	select {
	case <-requestDone:
		assert.Error(t, requestErr)
	case <-time.After(time.Second):
		t.Fatal("profiler did not close the active request")
	}
	release()
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("timed out releasing profiler handler")
	}

	restartedAddress, err := profiler.EnableProfiler(port)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, profilerAddress, restartedAddress)
	waitForProfiler(t, func() bool {
		return profilerServing(port)
	})
}

func TestProfilerBindFailure(t *testing.T) {
	occupied, err := net.Listen("tcp", ":0")
	if !assert.NoError(t, err) {
		return
	}
	t.Cleanup(func() {
		_ = occupied.Close()
	})

	port := int32(occupied.Addr().(*net.TCPAddr).Port)
	profiler := newProfilerServer(t)
	address, err := profiler.EnableProfiler(port)
	if !assert.Error(t, err) {
		return
	}
	assert.ErrorIs(t, err, syscall.EADDRINUSE)

	shownAddress, showErr := profiler.ShowProfiler()
	if !assert.NoError(t, showErr) {
		return
	}
	assert.Equal(t, address, shownAddress)

	if !assert.NoError(t, occupied.Close()) {
		return
	}
	_, err = profiler.EnableProfiler(port)
	if !assert.NoError(t, err) {
		return
	}
	_, err = profiler.DisableProfiler()
	assert.NoError(t, err)
}

func TestProfilerUnexpectedServeError(t *testing.T) {
	profiler := newProfilerServer(t)
	port := availableProfilerPort(t)
	profilerAddress, err := profiler.EnableProfiler(port)
	if !assert.NoError(t, err) {
		return
	}

	waitForProfiler(t, func() bool {
		return profilerServing(port)
	})

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), time.Second)
	if !assert.NoError(t, err) {
		return
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	if !assert.NoError(t, conn.SetDeadline(time.Now().Add(time.Second))) {
		return
	}

	reader := bufio.NewReader(conn)
	_, err = fmt.Fprintf(
		conn,
		"GET /debug/pprof/ HTTP/1.1\r\nHost: 127.0.0.1:%d\r\nConnection: keep-alive\r\n\r\n",
		port,
	)
	if !assert.NoError(t, err) {
		return
	}
	response, err := http.ReadResponse(reader, nil)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.False(t, response.Close) {
		_ = response.Body.Close()
		return
	}
	if !assert.Equal(t, http.StatusOK, response.StatusCode) {
		_ = response.Body.Close()
		return
	}
	if _, err = io.Copy(io.Discard, response.Body); !assert.NoError(t, err) {
		_ = response.Body.Close()
		return
	}
	if !assert.NoError(t, response.Body.Close()) {
		return
	}
	if !assert.NoError(t, conn.SetDeadline(time.Time{})) {
		return
	}

	profiler.lock.Lock()
	if profiler.listener == nil {
		profiler.lock.Unlock()
		t.Fatal("profiler listener is nil after enabling")
	}
	err = profiler.listener.Close()
	profiler.lock.Unlock()
	if !assert.NoError(t, err) {
		return
	}

	if !assert.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second))) {
		return
	}
	var readBuffer [1]byte
	_, readErr := conn.Read(readBuffer[:])
	if !assert.Error(t, readErr) {
		return
	}
	var networkErr net.Error
	if errors.As(readErr, &networkErr) && networkErr.Timeout() {
		t.Errorf("profiler keep-alive connection remained open after unexpected Serve error: %v", readErr)
		return
	}

	var shownAddress string
	var showErr error
	waitForProfiler(t, func() bool {
		shownAddress, showErr = profiler.ShowProfiler()
		return showErr == nil && shownAddress != "" && shownAddress != profilerAddress
	})
	if !assert.NoError(t, showErr) {
		return
	}
	assert.NotEmpty(t, shownAddress)
	assert.NotEqual(t, profilerAddress, shownAddress)

	restartedAddress, err := profiler.EnableProfiler(port)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, profilerAddress, restartedAddress)
	waitForProfiler(t, func() bool {
		return profilerServing(port)
	})
}

func waitForProfiler(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for profiler state")
}

func profilerServing(port int32) bool {
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()

	client := http.Client{Transport: transport, Timeout: time.Second}
	response, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/debug/pprof/", port))
	if err != nil {
		return false
	}
	return response.Body.Close() == nil && response.StatusCode == http.StatusOK
}

func newProfilerServer(t *testing.T) *Server {
	t.Helper()

	profiler := NewServer("test")
	t.Cleanup(func() {
		_, err := profiler.DisableProfiler()
		assert.NoError(t, err)
	})
	return profiler
}

func availableProfilerPort(t *testing.T) int32 {
	t.Helper()

	listener, err := net.Listen("tcp", ":0")
	if !assert.NoError(t, err) {
		return 0
	}
	port := int32(listener.Addr().(*net.TCPAddr).Port)
	if !assert.NoError(t, listener.Close()) {
		return 0
	}
	return port
}

func connected(port int32) bool {
	targetAddr := fmt.Sprintf("127.0.0.1:%d", port)
	conn, err := net.DialTimeout("tcp", targetAddr, time.Second)
	if err != nil {
		return false
	}
	return conn.Close() == nil
}

// start server and return client
func server(t *testing.T) *Client {
	t.Helper()

	// We do not use a fake connection because we want to test NewClient.
	listen, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
		return nil
	}

	grpcServer := grpc.NewServer()
	profilerServer := NewServer("test")
	profilerrpc.RegisterProfilerServer(grpcServer, profilerServer)
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- grpcServer.Serve(listen)
	}()
	t.Cleanup(func() {
		grpcServer.Stop()
		_ = listen.Close()
		if err := <-serveErr; err != nil &&
			!errors.Is(err, grpc.ErrServerStopped) &&
			!errors.Is(err, net.ErrClosed) {
			t.Errorf("failed to stop profiler service: %v", err)
		}
	})

	client, err := NewClient(utils.GetGRPCAddress(listen.Addr().String()), "test")
	if err != nil {
		t.Fatalf("failed to create profiler client: %v", err)
		return nil
	}
	t.Cleanup(func() {
		assert.NoError(t, client.Close())
	})
	t.Cleanup(func() {
		_, err := profilerServer.DisableProfiler()
		assert.NoError(t, err)
	})
	return client
}
