package server

import (
	"context"

	api "github.com/kyu08/distributed-services-with-go/api/v1"
)

type Config struct {
	CommitLog CommitLog
}

type CommitLog interface {
	Append(*api.Record) (uint64, error)
	Read(uint64) (*api.Record, error)
}

var _ api.LogServer = (*grpcServer)(nil)

type grpcServer struct {
	api.UnimplementedLogServer
	*Config
}

func newgrpcServer(config *Config) (srv *grpcServer, err error) {
	srv = &grpcServer{
		Config: config,
	}

	return srv, nil
}

func (s *grpcServer) Produce(ctx context.Context, req *api.ProduceRequest) (*api.ProduceResponse, error) {
	offset, err := s.CommitLog.Append(req.Record)
	if err != nil {
		return nil, err
	}
	return &api.ProduceResponse{Offset: offset}, nil
}

func (s *grpcServer) Consume(ctx context.Context, req *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	record, err := s.CommitLog.Read(req.Offset)
	if err != nil {
		return nil, err
	}

	return &api.ConsumeResponse{Record: record}, nil
}