package grpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/longhorn/go-common-libs/grpc/profiler/proto"
	"github.com/longhorn/go-common-libs/utils"
)

// ProfilerClientContext is the context with the ProfilerClient and the connection
//
//   - conn: the connection for the gRPC server
//   - service: the ProfilerClient
type ProfilerClientContext struct {
	conn    *grpc.ClientConn
	service proto.ProfilerClient
}

// Close closes the connection for the gRPC server
func (c ProfilerClientContext) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// ProfilerServer is the gRPC server to provide the basic operations
// like (show/enable/disable) for the profiler
//
//   - name: the name of the server
//   - errMsg: the error message current server has (if enabled failed, it will have the error message)
//   - profilerServer: the http server for the profiler
//   - profilerLock: the RW lock for the profilerServer
type ProfilerServer struct {
	name   string
	errMsg string

	profilerServer *http.Server
	profilerLock   sync.RWMutex
}

// ProfilerClient is the gRPC client to interactive with the ProfilerServer
//   - Name: the name of the client
//   - serviceURL: the URL of the gRPC server
//   - ProfilerClientContext: the context with the ProfilerClient and the connection
type ProfilerClient struct {
	Name       string
	serviceURL string
	ProfilerClientContext
}

// getProfilerServiceClient returns the ProfilerClient (internal function)
func (c *ProfilerClient) getProfilerServiceClient() proto.ProfilerClient {
	return c.service
}

// NewProfilerServer returns the gRPC service server for the profiler
func NewProfilerServer(name string) *ProfilerServer {
	return &ProfilerServer{
		name:           name,
		profilerServer: nil,
	}
}

// NewProfilerClient returns the gRPC client to interactive with the gRPC server for the profiler
func NewProfilerClient(address, name string, dialOpts ...grpc.DialOption) (*ProfilerClient, error) {
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, opt := range dialOpts {
		if opt != nil {
			logrus.Debugf("Add dial option: %v", opt)
			opts = append(opts, opt)
		}
	}
	getContext := func(serviceURL string) (ProfilerClientContext, error) {
		connection, err := grpc.Dial(serviceURL, opts...)
		if err != nil {
			return ProfilerClientContext{}, fmt.Errorf("cannot connect to ProfilerServer %v", serviceURL)
		}

		return ProfilerClientContext{
			conn:    connection,
			service: proto.NewProfilerClient(connection),
		}, nil
	}

	serviceURL := utils.GetGRPCAddress(address)
	context, err := getContext(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create ProfilerClient for %v: %v", serviceURL, err)
	}

	return &ProfilerClient{
		Name:                  name,
		serviceURL:            serviceURL,
		ProfilerClientContext: context,
	}, nil

}

// ProfilerOP is the gRPC function to provide the entry point for the profiler operations
func (s *ProfilerServer) ProfilerOP(ctx context.Context, req *proto.ProfilerOPRequest) (*proto.ProfilerOPResponse, error) {
	logrus.Infof("Profiler operation: %v, port: %v", req.RequestOp, req.PortNumber)
	var err error
	switch req.RequestOp {
	case proto.Op_SHOW:
		reply := &proto.ProfilerOPResponse{}
		reply.ProfilerAddr, err = s.ShowProfiler()
		return reply, err
	case proto.Op_ENABLE:
		reply := &proto.ProfilerOPResponse{}
		reply.ProfilerAddr, err = s.EnableProfiler(req.PortNumber)
		return reply, err
	case proto.Op_DISABLE:
		reply := &proto.ProfilerOPResponse{}
		reply.ProfilerAddr, err = s.DisableProfiler()
		return reply, err
	default:
		return nil, errors.New("invalid operation")
	}
}

// ShowProfiler returns the address of the profiler
func (s *ProfilerServer) ShowProfiler() (string, error) {
	s.profilerLock.RLock()
	defer s.profilerLock.RUnlock()

	logrus.Info("Prepareing to show the profiler address")
	if s.profilerServer != nil {
		return s.profilerServer.Addr, nil
	}
	if s.errMsg != "" {
		return s.errMsg, nil
	}
	return "", nil
}

// EnableProfiler enables the profiler with specific port number.
// It will fail if the profiler is already enabled or the port number is invalid(0).
// For the profiler server enabling failed, it will return the error message.
// And keep the above error message, then user can get the error message by op `show`.
// The normal(success) case will return the address of the profiler server.
func (s *ProfilerServer) EnableProfiler(portNumber int32) (string, error) {
	s.profilerLock.Lock()
	defer s.profilerLock.Unlock()

	logrus.Info("Prepareing to enable the profiler")

	if s.profilerServer != nil {
		return "", fmt.Errorf("profiler server is already running at %v", s.profilerServer.Addr)
	}

	profilerPort := int(portNumber)
	if profilerPort == 0 {
		s.errMsg = "enable profiler failed: invalid port number"
		return s.errMsg, fmt.Errorf("invalid port number: %v", portNumber)
	}

	profilerAddr := fmt.Sprintf(":%d", profilerPort)
	s.profilerServer = &http.Server{
		Addr:              profilerAddr,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := s.profilerServer.ListenAndServe(); err != http.ErrServerClosed {
			logrus.WithError(err).Warnf("Get error when start profiler server %v", s.profilerServer.Addr)
			s.profilerServer = nil
			s.errMsg = err.Error()
			return
		}
		logrus.Infof("Profiler server (%v) is closed", s.profilerServer.Addr)
	}()

	logrus.Infof("Waiting the profiler server(%v) to start", s.profilerServer.Addr)
	// Wait for the profiler server to start, and check the profiler server.
	retryCount := 3
	for i := 0; i < retryCount; i++ {
		conn, err := net.DialTimeout("tcp", s.profilerServer.Addr, 1*time.Second)
		if err == nil {
			conn.Close()
			break
		}
	}
	if s.profilerServer == nil {
		return s.errMsg, fmt.Errorf("failed to start profiler server(%v)", profilerAddr)
	}
	defer func() {
		s.errMsg = ""
	}()
	return s.profilerServer.Addr, nil
}

// DisableProfiler disables the profiler.
// It will return the error when we try to shutdown the profiler server.
func (s *ProfilerServer) DisableProfiler() (string, error) {
	s.profilerLock.Lock()
	defer s.profilerLock.Unlock()

	logrus.Info("Prepareing to disable the profiler")
	if s.profilerServer == nil {
		return "", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.profilerServer.Shutdown(ctx); err != nil {
		logrus.WithError(err).Warnf("Failed to shutdown the profiler %v", s.profilerServer.Addr)
		return "", err
	}

	s.profilerServer = nil
	return "", nil

}

// ProfilerOP will call the doProfilerOP for the ProfilerServer.
// It will convert the hunam readable op to internal usage and is the entry point for the profiler operations.
func (c *ProfilerClient) ProfilerOP(op string, portNumber int32) (string, error) {
	return c.doProfilerOP(proto.Op(proto.Op_value[op]), portNumber)
}

// doProfilerOP is the internal function to call the ProfilerOP for the ProfilerServer.
func (c *ProfilerClient) doProfilerOP(op proto.Op, portNumber int32) (string, error) {
	controllerServiceClient := c.getProfilerServiceClient()
	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCServiceTimeout)
	defer cancel()

	reply, err := controllerServiceClient.ProfilerOP(ctx, &proto.ProfilerOPRequest{
		RequestOp:  op,
		PortNumber: portNumber,
	})
	if err != nil {
		return "", err
	}
	return reply.ProfilerAddr, nil
}
