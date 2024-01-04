package profiler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/generated/profilerpb"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/utils"
)

const (
	opSHOW    = "SHOW"
	opENABLE  = "ENABLE"
	opDISABLE = "DISABLE"
)

func (s TestSuite) TestProfilerServiceOperations(c *C) {
	type testCase struct {
		op         string
		portNumber int32

		expectRet bool
	}

	testCases := map[string]testCase{
		"ProfilerOP(...): show": {
			op:         opSHOW,
			portNumber: 0,
			expectRet:  true,
		},
		"ProfilerOP(...): enable/disable": {
			op:         opENABLE,
			portNumber: 55555,
			expectRet:  true,
		},
		"ProfilerOP(...): invalidate op": {
			op:         "INVALID",
			portNumber: 0,
			expectRet:  false,
		},
	}

	grpcServerPort, err := utils.GenerateRandomNumber(50000, 55000)
	if err != nil {
		c.Fatalf("Failed to generate random number: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := server(ctx, grpcServerPort)

	for testName, testCase := range testCases {
		c.Logf("testing grpc.%v", testName)

		if testCase.expectRet == false {
			_, err := client.ProfilerOP(testCase.op, testCase.portNumber)
			c.Assert(err, NotNil, Commentf(test.ErrResultFmt, testName))
			continue
		}

		_, err := client.ProfilerOP(testCase.op, testCase.portNumber)
		c.Assert(err, IsNil, Commentf(test.ErrResultFmt, testName))

		// test the Op_ENABLE we should also test the Op_DISABLE
		if testCase.op == opENABLE {
			c.Assert(connected(testCase.portNumber), Equals, true, Commentf(test.ErrResultFmt, testName))
			// Then, we disable it.
			_, err := client.ProfilerOP(opDISABLE, testCase.portNumber)
			c.Assert(err, IsNil, Commentf(test.ErrResultFmt, testName))
			c.Assert(connected(testCase.portNumber), Equals, false, Commentf(test.ErrResultFmt, testName))
		}
	}
}

func connected(port int32) bool {
	targetAddr := fmt.Sprintf(":%d", port)
	retryCount := 3
	connected := false
	for i := 0; i < retryCount; i++ {
		conn, err := net.DialTimeout("tcp", targetAddr, 1*time.Second)
		if err == nil {
			connected = true
			_ = conn.Close()
			break
		}
	}
	return connected
}

// start server and return client
func server(_ context.Context, grpcServerPort int64) *Client {

	// we do not use fake connection because we want to test the `NewPorfilerClient` function
	port := fmt.Sprintf(":%d", grpcServerPort)
	listen, err := net.Listen("tcp", port)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	server := NewServer("test")
	profilerpb.RegisterProfilerServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(listen); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to start profiler server: %v\n", err)
		}
	}()

	client, err := NewClient(utils.GetGRPCAddress(listen.Addr().String()), "test", nil)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to dial: %v\n", err)
	}

	return client
}
