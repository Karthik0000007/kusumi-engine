package server

import (
	"context"
	"net"
	"testing"

	pb "github.com/kasumi-engine/ingestion/api/kasumi/v1"
	"github.com/kasumi-engine/ingestion/internal/config"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
	"github.com/rs/zerolog"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

func setupServer(t *testing.T, cfg *config.IngestionConfig) (*grpc.Server, pb.IngestionServiceClient, func()) {
	lis = bufconn.Listen(bufSize)
	log := zerolog.Nop()

	interceptors := NewInterceptors(cfg, log)
	s := grpc.NewServer(
		grpc.UnaryInterceptor(interceptors.UnaryInterceptor()),
		grpc.StreamInterceptor(interceptors.StreamInterceptor()),
	)

	ingestionServer := NewIngestionServer(cfg, log)
	// Do not start the background processor so we can test backpressure filling up
	pb.RegisterIngestionServiceServer(s, ingestionServer)

	go func() {
		if err := s.Serve(lis); err != nil {
			t.Logf("Server exited with error: %v", err)
		}
	}()

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}

	client := pb.NewIngestionServiceClient(conn)

	cleanup := func() {
		conn.Close()
		s.Stop()
	}

	return s, client, cleanup
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestIngestClick_Success(t *testing.T) {
	cfg := config.Defaults().Ingestion
	cfg.BackpressureBufferSize = 10

	_, client, cleanup := setupServer(t, &cfg)
	defer cleanup()

	ctx := context.Background()
	req := &pb.ClickEvent{
		Context: &pb.SessionContext{
			SessionId: "sess_1",
			Timestamp: timestamppb.Now(),
		},
		ItemId: "item_1",
	}

	resp, err := client.IngestClick(ctx, req)
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestIngestClick_ValidationFailed(t *testing.T) {
	cfg := config.Defaults().Ingestion
	_, client, cleanup := setupServer(t, &cfg)
	defer cleanup()

	ctx := context.Background()
	req := &pb.ClickEvent{
		// Missing Context
		ItemId: "item_1",
	}

	_, err := client.IngestClick(ctx, req)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestBackpressure(t *testing.T) {
	cfg := config.Defaults().Ingestion
	cfg.BackpressureBufferSize = 2 // Small buffer

	_, client, cleanup := setupServer(t, &cfg)
	defer cleanup()

	ctx := context.Background()
	req := &pb.ClickEvent{
		Context: &pb.SessionContext{
			SessionId: "sess_1",
			Timestamp: timestamppb.Now(),
		},
		ItemId: "item_1",
	}

	// Fill the buffer
	_, err := client.IngestClick(ctx, req)
	assert.NoError(t, err)
	_, err = client.IngestClick(ctx, req)
	assert.NoError(t, err)

	// Third request should fail due to backpressure (ResourceExhausted)
	_, err = client.IngestClick(ctx, req)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Contains(t, st.Message(), "ingestion queue full")
}

func TestRateLimiter(t *testing.T) {
	cfg := config.Defaults().Ingestion
	cfg.RateLimitRPS = 1 // 1 request per second

	_, client, cleanup := setupServer(t, &cfg)
	defer cleanup()

	ctx := context.Background()
	req := &pb.ClickEvent{
		Context: &pb.SessionContext{
			SessionId: "sess_1",
			Timestamp: timestamppb.Now(),
		},
		ItemId: "item_1",
	}

	// The limiter allows burst of 2x RPS, so 2 requests should succeed.
	_, err := client.IngestClick(ctx, req)
	assert.NoError(t, err)

	_, err = client.IngestClick(ctx, req)
	assert.NoError(t, err)

	// Third request should be rate limited (ResourceExhausted)
	_, err = client.IngestClick(ctx, req)
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.ResourceExhausted, st.Code())
	assert.Contains(t, st.Message(), "rate limit exceeded")
}
