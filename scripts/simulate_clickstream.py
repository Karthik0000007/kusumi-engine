#!/usr/bin/env python3
"""
Kasumi Engine — Synthetic Clickstream Simulator

Simulates real-time user traffic (clicks and dwells) and serializes
events using the Protobuf schema.
"""

import argparse
import random
import time
import uuid
from datetime import datetime, timezone
import json
import sys
from pathlib import Path

# Add retrieval directory to path so we can import the generated protobufs
# In a real environment, the bindings would be built and installed properly
sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "retrieval"))

try:
    from kasumi.v1 import events_pb2
    HAS_PROTO = True
except ImportError:
    print("Warning: Protobuf bindings not found. Run 'buf generate' in proto/ first.")
    HAS_PROTO = False


def generate_session() -> dict:
    """Generate a synthetic user session context."""
    return {
        "session_id": f"sess_{uuid.uuid4().hex}",
        "user_id": f"usr_{uuid.uuid4().hex}" if random.random() > 0.3 else "",
        "device_type": random.choice(["mobile_ios", "mobile_android", "desktop_web"]),
        "geo_region": random.choice(["tokyo", "osaka", "fukuoka", "sapporo"]),
        "timestamp": datetime.now(timezone.utc)
    }


def simulate_stream(duration_sec: int, qps: int, catalog_file: str) -> None:
    """Simulate a stream of click/dwell events."""
    
    # Load some item IDs from catalog to use
    item_ids = [f"item_{i}" for i in range(100)] # fallback
    
    catalog_path = Path(catalog_file)
    if catalog_path.exists():
        print(f"Loading item IDs from {catalog_path}...")
        try:
            with open(catalog_path, "r") as f:
                item_ids = []
                for i, line in enumerate(f):
                    if i >= 10000:  # Just load a subset for simulation
                        break
                    item_ids.append(json.loads(line)["item_id"])
        except Exception as e:
            print(f"Failed to load catalog: {e}. Using synthetic IDs.")

    print(f"Starting simulation: {qps} QPS for {duration_sec} seconds")
    
    start_time = time.time()
    events_generated = 0
    
    active_sessions = [generate_session() for _ in range(50)]
    
    sleep_time = 1.0 / qps
    
    while time.time() - start_time < duration_sec:
        loop_start = time.time()
        
        # Session churn
        if random.random() < 0.1:
            active_sessions.remove(random.choice(active_sessions))
            active_sessions.append(generate_session())
            
        session = random.choice(active_sessions)
        item_id = random.choice(item_ids)
        
        is_click = random.random() > 0.2
        
        if HAS_PROTO:
            # Build proto message
            ctx = events_pb2.SessionContext(
                session_id=session["session_id"],
                user_id=session["user_id"],
                device_type=session["device_type"],
                geo_region=session["geo_region"]
            )
            ctx.timestamp.FromDatetime(datetime.now(timezone.utc))
            
            if is_click:
                event = events_pb2.ClickEvent(
                    context=ctx,
                    item_id=item_id,
                    source_placement=random.choice(["homepage_feed", "search", "related"]),
                    rank_position=random.randint(1, 50)
                )
            else:
                event = events_pb2.DwellEvent(
                    context=ctx,
                    item_id=item_id,
                    duration_ms=random.randint(1000, 120000)
                )
        
        events_generated += 1
        
        # Pace the loop to hit target QPS
        elapsed = time.time() - loop_start
        if elapsed < sleep_time:
            time.sleep(sleep_time - elapsed)
            
        if events_generated % (qps * 5) == 0:
            print(f"Generated {events_generated} events...")
            
    print(f"Simulation complete. Generated {events_generated} total events.")


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Simulate clickstream traffic.")
    parser.add_argument("--duration", type=int, default=10, help="Duration in seconds")
    parser.add_argument("--qps", type=int, default=100, help="Queries per second")
    parser.add_argument("--catalog", type=str, default="data/catalog.jsonl", help="Catalog file to draw items from")
    
    args = parser.parse_args()
    simulate_stream(args.duration, args.qps, args.catalog)
