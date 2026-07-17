package server

import (
	"context"
	"net"
	"sync"
	"time"

	pb "github.com/kasumi-engine/ingestion/api/kasumi/v1"
	"github.com/kasumi-engine/ingestion/internal/config"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"github.com/rs/zerolog"
)

type Interceptors struct {
	cfg        *config.IngestionConfig
	limiterMap sync.Map
	log        zerolog.Logger
}

func NewInterceptors(cfg *config.IngestionConfig, log zerolog.Logger) *Interceptors {
	return &Interceptors{
		cfg: cfg,
		log: log,
	}
}

func (i *Interceptors) getLimiter(ip string) *rate.Limiter {
	limiter, exists := i.limiterMap.Load(ip)
	if !exists {
		// allow burst of 2x the RPS
		newLimiter := rate.NewLimiter(rate.Limit(i.cfg.RateLimitRPS), i.cfg.RateLimitRPS*2)
		limiter, _ = i.limiterMap.LoadOrStore(ip, newLimiter)
	}
	return limiter.(*rate.Limiter)
}

func extractIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
		return tcpAddr.IP.String()
	}
	return p.Addr.String()
}

func validateContext(sessionCtx *pb.SessionContext) error {
	if sessionCtx == nil {
		return status.Error(codes.InvalidArgument, "missing session context")
	}
	if sessionCtx.SessionId == "" {
		return status.Error(codes.InvalidArgument, "missing session_id")
	}
	if sessionCtx.Timestamp == nil {
		return status.Error(codes.InvalidArgument, "missing timestamp")
	}

	ts := sessionCtx.Timestamp.AsTime()
	now := time.Now()
	// allow up to 24h past and 1h future
	if ts.After(now.Add(time.Hour)) || ts.Before(now.Add(-24*time.Hour)) {
		return status.Error(codes.InvalidArgument, "timestamp out of acceptable bounds")
	}
	return nil
}

func validateRequest(req interface{}) error {
	switch v := req.(type) {
	case *pb.ClickEvent:
		return validateContext(v.Context)
	case *pb.DwellEvent:
		return validateContext(v.Context)
	}
	return nil
}

func (i *Interceptors) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ip := extractIP(ctx)
		limiter := i.getLimiter(ip)

		if !limiter.Allow() {
			i.log.Warn().Str("ip", ip).Msg("Rate limit exceeded")
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		if err := validateRequest(req); err != nil {
			i.log.Warn().Str("ip", ip).Err(err).Msg("Validation failed")
			return nil, err
		}

		start := time.Now()
		resp, err := handler(ctx, req)
		i.log.Info().
			Str("method", info.FullMethod).
			Str("ip", ip).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Handled unary request")

		return resp, err
	}
}

func (i *Interceptors) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()
		ip := extractIP(ctx)
		limiter := i.getLimiter(ip)

		if !limiter.Allow() {
			i.log.Warn().Str("ip", ip).Msg("Rate limit exceeded for stream")
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		start := time.Now()
		err := handler(srv, ss)
		i.log.Info().
			Str("method", info.FullMethod).
			Str("ip", ip).
			Dur("duration", time.Since(start)).
			Err(err).
			Msg("Handled stream request")

		return err
	}
}
