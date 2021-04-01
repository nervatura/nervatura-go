//+build grpc all

package app

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	ntura "github.com/nervatura/nervatura-go/pkg/nervatura"
	pb "github.com/nervatura/nervatura-go/pkg/proto"
	srv "github.com/nervatura/nervatura-go/pkg/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type rpcServer struct {
	app        *App
	service    srv.RPCService
	result     string
	server     *grpc.Server
	tlsEnabled bool
}

func init() {
	registerService("grpc", &rpcServer{})
}

// StartService - Start Nervatura RPC server
func (s *rpcServer) StartService() error {
	s.service = srv.RPCService{
		GetNervaStore: s.app.GetNervaStore,
	}

	var cred credentials.TransportCredentials
	if ntura.GetEnvValue("bool", os.Getenv("NT_GRPC_TLS_ENABLED")).(bool) {
		if os.Getenv("NT_TLS_CERT_FILE") != "" && os.Getenv("NT_TLS_KEY_FILE") != "" {
			cert, err := tls.LoadX509KeyPair(os.Getenv("NT_TLS_CERT_FILE"), os.Getenv("NT_TLS_KEY_FILE"))
			if err != nil {
				s.app.errorLog.Printf("failed to load key pair: %v\n", err)
				os.Exit(2)
			}
			cred = credentials.NewServerTLSFromCert(&cert)
			s.tlsEnabled = true
		}
	}

	addr := fmt.Sprintf(":%d", ntura.GetEnvValue("int", os.Getenv("NT_GRPC_PORT")).(int))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.app.errorLog.Printf("grpc server: failed to listen %v\n", err)
		os.Exit(2)
	}

	s.server = grpc.NewServer(
		grpc.Creds(cred),
		// MaxConnectionAge is just to avoid long connection, to facilitate load balancing
		// MaxConnectionAgeGrace will torn them, default to infinity
		grpc.KeepaliveParams(keepalive.ServerParameters{MaxConnectionAge: 2 * time.Minute}),
		grpc.UnaryInterceptor(s.tokenAuth),
	)

	pb.RegisterAPIServer(s.server, &s.service)

	s.app.infoLog.Printf("GRPC server serving at: %s. SSL/TLS authentication: %v.\n",
		os.Getenv("NT_GRPC_PORT"), s.tlsEnabled)

	return s.server.Serve(ln)
}

// StopService - Stop Nervatura RPC server
func (s *rpcServer) StopService(interface{}) error {
	if s.server != nil {
		s.app.infoLog.Println("stopping GRPC server")
		s.server.GracefulStop()
	}
	return nil
}

func (s *rpcServer) ConnectApp(app interface{}) {
	s.app = app.(*App)
}

func (s *rpcServer) Results() string {
	return s.result
}

func (s *rpcServer) tokenAuth(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	switch info.FullMethod {
	case "/nervatura.API/UserLogin", "/nervatura.API/TokenDecode":
		return handler(ctx, req)
	default:
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata")
		}

		if info.FullMethod == "/nervatura.API/DatabaseCreate" {
			ictx, err := s.service.ApiKeyAuth(md["x-api-key"], ctx)
			if err != nil {
				return nil, status.Errorf(codes.Unauthenticated, "missing api key")
			}
			return handler(ictx, req)
		}

		ictx, err := s.service.TokenAuth(md["authorization"], ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid token")
		}
		return handler(ictx, req)
	}
}
