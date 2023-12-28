package grpc

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	. "gopkg.in/check.v1"

	"github.com/longhorn/go-common-libs/grpc/profiler/proto"
	"github.com/longhorn/go-common-libs/test"
	"github.com/longhorn/go-common-libs/utils"
)

func (s TestSuite) TestProfilerServiceOperations(c *C) {
	type testCase struct {
		op         string
		portNumber int32

		expect_ret bool
	}

	testCases := map[string]testCase{
		"ProfilerOP(...): show": {
			op:         "show",
			portNumber: 0,
			expect_ret: true,
		},
		"ProfilerOP(...): enable": {
			op:         "enable",
			portNumber: 55555,
			expect_ret: true,
		},
		"ProfilerOP(...): disable": {
			op:         "disable",
			portNumber: 0,
			expect_ret: true,
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

		_, err := client.ProfilerOP(testCase.op, testCase.portNumber)
		c.Assert(err, IsNil, Commentf(test.ErrResultFmt, testName))
	}
}

// start server and return client
func server(ctx context.Context, grpcServerPort int64) *ProfilerClient {

	// we do not use fake connection because we want to test the `NewPorfilerClient` function
	port := fmt.Sprintf(":%d", grpcServerPort)
	listen, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to listen: %v\n", err)
	}

	grpcServer := grpc.NewServer()
	server := NewProfilerServer("test")
	proto.RegisterProfilerServer(grpcServer, server)
	go func() {
		if err := grpcServer.Serve(listen); err != nil && err != grpc.ErrServerStopped {
			fmt.Fprintf(os.Stderr, "Failed to start profiler server: %v\n", err)
		}
	}()

	client, err := NewProfilerClient(utils.GetGRPCAddress(listen.Addr().String()), "test", nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to dial: %v\n", err)
	}

	return client
}
