package server

import (
	"context"
	"io"

	pb "github.com/kasumi-engine/ingestion/api/kasumi/v1"
	"github.com/kasumi-engine/ingestion/internal/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"github.com/rs/zerolog"
)

type IngestionServer struct {
	pb.UnimplementedIngestionServiceServer
	cfg       *config.IngestionConfig
	eventChan chan proto.Message
	log       zerolog.Logger
}

func NewIngestionServer(cfg *config.IngestionConfig, log zerolog.Logger) *IngestionServer {
	return &IngestionServer{
		cfg:       cfg,
		eventChan: make(chan proto.Message, cfg.BackpressureBufferSize),
		log:       log,
	}
}

// Start processing events in the background (we can simulate saving/batching for now)
func (s *IngestionServer) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-s.eventChan:
				// Typically, we would write this event to Kafka/Redis or batch it.
				// For now, just log at debug level.
				s.log.Debug().Msgf("Processed event: %T", event)
			}
		}
	}()
}

// pushEvent tries to push an event onto the channel, enforcing backpressure.
func (s *IngestionServer) pushEvent(event proto.Message) error {
	select {
	case s.eventChan <- event:
		return nil
	default:
		// Channel is full: load shedding
		s.log.Warn().Msg("Ingestion queue full, dropping event")
		return status.Error(codes.ResourceExhausted, "ingestion queue full")
	}
}

func (s *IngestionServer) IngestClick(ctx context.Context, req *pb.ClickEvent) (*pb.IngestionResponse, error) {
	if err := s.pushEvent(req); err != nil {
		return nil, err
	}
	return &pb.IngestionResponse{Success: true, Message: "Click ingested"}, nil
}

func (s *IngestionServer) IngestDwell(ctx context.Context, req *pb.DwellEvent) (*pb.IngestionResponse, error) {
	if err := s.pushEvent(req); err != nil {
		return nil, err
	}
	return &pb.IngestionResponse{Success: true, Message: "Dwell ingested"}, nil
}

func (s *IngestionServer) StreamEvents(stream pb.IngestionService_StreamEventsServer) error {
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.IngestionResponse{
				Success: true,
				Message: "Stream completed successfully",
			})
		}
		if err != nil {
			return err
		}

		// Req is a StreamEventWrapper
		var event proto.Message
		switch e := req.Event.(type) {
		case *pb.StreamEventWrapper_Click:
			event = e.Click
		case *pb.StreamEventWrapper_Dwell:
			event = e.Dwell
		default:
			// Unknown event type
			return status.Error(codes.InvalidArgument, "unknown event type in stream")
		}

		if err := s.pushEvent(event); err != nil {
			return err
		}
	}
}

// HealthChecker implements the standard gRPC health check service.
type HealthChecker struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (h *HealthChecker) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (h *HealthChecker) Watch(req *grpc_health_v1.HealthCheckRequest, server grpc_health_v1.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "watch is not implemented")
}
