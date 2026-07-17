#!/usr/bin/env python3
"""
Kasumi Engine — Synthetic Catalog Generator

Generates a synthetic item catalog for local development and benchmarking.
Produces a Parquet or JSONL file with item metadata.
"""

import argparse
import json
import random
import uuid
from datetime import datetime, timedelta
from pathlib import Path

try:
    import pandas as pd
    import numpy as np
except ImportError:
    print("Warning: pandas and numpy are required for Parquet output.")
    pd = None
    np = None

CATEGORIES = [
    "electronics", "clothing", "home", "beauty", 
    "sports", "toys", "books", "automotive", 
    "grocery", "health"
]

def generate_catalog(num_items: int, output_path: str, format: str) -> None:
    """Generate a synthetic catalog and save to disk."""
    print(f"Generating {num_items} synthetic items...")
    
    items = []
    base_time = datetime.now() - timedelta(days=365)
    
    for _ in range(num_items):
        item_id = f"item_{uuid.uuid4().hex[:12]}"
        category = random.choice(CATEGORIES)
        price = round(random.uniform(5.0, 500.0), 2)
        
        # Synthetic popularity score (power law distribution approximation)
        popularity = min(1.0, max(0.0, random.betavariate(1, 5)))
        
        created_offset = random.randint(0, 365 * 24 * 60)
        created_at = base_time + timedelta(minutes=created_offset)
        
        items.append({
            "item_id": item_id,
            "category": category,
            "price": price,
            "popularity_score": popularity,
            "embedding_version": "v1",
            "created_at": created_at.isoformat()
        })
        
    out_path = Path(output_path)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    
    if format == "jsonl":
        print(f"Saving to {out_path} as JSON Lines...")
        with open(out_path, "w") as f:
            for item in items:
                f.write(json.dumps(item) + "\n")
    elif format == "parquet" and pd is not None:
        print(f"Saving to {out_path} as Parquet...")
        df = pd.DataFrame(items)
        df.to_parquet(out_path, index=False)
    else:
        if format == "parquet":
            print("Pandas not available. Falling back to JSON Lines.")
            out_path = out_path.with_suffix(".jsonl")
        
        print(f"Saving to {out_path} as JSON Lines...")
        with open(out_path, "w") as f:
            for item in items:
                f.write(json.dumps(item) + "\n")
                
    print(f"Successfully generated catalog at {out_path}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate synthetic item catalog.")
    parser.add_argument("--count", type=int, default=100000, help="Number of items to generate")
    parser.add_argument("--output", type=str, default="data/catalog.jsonl", help="Output file path")
    parser.add_argument("--format", type=str, choices=["jsonl", "parquet"], default="jsonl", help="Output format")
    
    args = parser.parse_args()
    generate_catalog(args.count, args.output, args.format)
