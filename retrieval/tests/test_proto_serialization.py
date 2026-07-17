import pytest
from datetime import datetime, timezone
from kasumi.v1 import events_pb2

def test_session_context_serialization():
    ctx = events_pb2.SessionContext(
        session_id="sess_123",
        user_id="user_abc",
        device_type="desktop_web",
        geo_region="tokyo"
    )
    ctx.timestamp.FromDatetime(datetime.now(timezone.utc))
    
    serialized = ctx.SerializeToString()
    
    parsed_ctx = events_pb2.SessionContext()
    parsed_ctx.ParseFromString(serialized)
    
    assert parsed_ctx.session_id == "sess_123"
    assert parsed_ctx.user_id == "user_abc"
    assert parsed_ctx.device_type == "desktop_web"
    assert parsed_ctx.geo_region == "tokyo"
    assert parsed_ctx.timestamp.seconds > 0

def test_click_event_serialization():
    ctx = events_pb2.SessionContext(session_id="sess_456")
    click = events_pb2.ClickEvent(
        context=ctx,
        item_id="item_999",
        source_placement="search",
        rank_position=5
    )
    
    serialized = click.SerializeToString()
    
    parsed = events_pb2.ClickEvent()
    parsed.ParseFromString(serialized)
    
    assert parsed.context.session_id == "sess_456"
    assert parsed.item_id == "item_999"
    assert parsed.source_placement == "search"
    assert parsed.rank_position == 5

def test_recommendation_request_serialization():
    req = events_pb2.RecommendationRequest(
        context=events_pb2.SessionContext(session_id="sess_req"),
        num_candidates=10,
        filter_categories=["electronics", "home"]
    )
    
    serialized = req.SerializeToString()
    
    parsed = events_pb2.RecommendationRequest()
    parsed.ParseFromString(serialized)
    
    assert parsed.context.session_id == "sess_req"
    assert parsed.num_candidates == 10
    assert list(parsed.filter_categories) == ["electronics", "home"]
