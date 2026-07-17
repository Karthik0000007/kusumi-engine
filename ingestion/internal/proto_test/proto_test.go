package proto_test

import (
	"testing"
	"time"

	pb "github.com/kasumi-engine/ingestion/api/kasumi/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestSessionContextSerialization(t *testing.T) {
	ctx := &pb.SessionContext{
		SessionId:  "sess_123",
		UserId:     "user_abc",
		DeviceType: "mobile_ios",
		GeoRegion:  "osaka",
		Timestamp:  timestamppb.New(time.Now()),
	}

	data, err := proto.Marshal(ctx)
	if err != nil {
		t.Fatalf("Failed to marshal SessionContext: %v", err)
	}

	parsed := &pb.SessionContext{}
	err = proto.Unmarshal(data, parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal SessionContext: %v", err)
	}

	if parsed.SessionId != "sess_123" {
		t.Errorf("Expected SessionId 'sess_123', got '%s'", parsed.SessionId)
	}
	if parsed.GeoRegion != "osaka" {
		t.Errorf("Expected GeoRegion 'osaka', got '%s'", parsed.GeoRegion)
	}
	if parsed.Timestamp.AsTime().IsZero() {
		t.Errorf("Expected non-zero Timestamp")
	}
}

func TestDwellEventSerialization(t *testing.T) {
	dwell := &pb.DwellEvent{
		Context: &pb.SessionContext{
			SessionId: "sess_456",
		},
		ItemId:     "item_888",
		DurationMs: 45000,
	}

	data, err := proto.Marshal(dwell)
	if err != nil {
		t.Fatalf("Failed to marshal DwellEvent: %v", err)
	}

	parsed := &pb.DwellEvent{}
	err = proto.Unmarshal(data, parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal DwellEvent: %v", err)
	}

	if parsed.Context.SessionId != "sess_456" {
		t.Errorf("Expected Context.SessionId 'sess_456', got '%s'", parsed.Context.SessionId)
	}
	if parsed.ItemId != "item_888" {
		t.Errorf("Expected ItemId 'item_888', got '%s'", parsed.ItemId)
	}
	if parsed.DurationMs != 45000 {
		t.Errorf("Expected DurationMs 45000, got %d", parsed.DurationMs)
	}
}

func TestStreamEventWrapperSerialization(t *testing.T) {
	wrapper := &pb.StreamEventWrapper{
		Event: &pb.StreamEventWrapper_Click{
			Click: &pb.ClickEvent{
				ItemId:        "item_click",
				RankPosition:  1,
			},
		},
	}

	data, err := proto.Marshal(wrapper)
	if err != nil {
		t.Fatalf("Failed to marshal StreamEventWrapper: %v", err)
	}

	parsed := &pb.StreamEventWrapper{}
	err = proto.Unmarshal(data, parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal StreamEventWrapper: %v", err)
	}

	click, ok := parsed.Event.(*pb.StreamEventWrapper_Click)
	if !ok {
		t.Fatalf("Expected event to be of type Click")
	}

	if click.Click.ItemId != "item_click" {
		t.Errorf("Expected ItemId 'item_click', got '%s'", click.Click.ItemId)
	}
}
