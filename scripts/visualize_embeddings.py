#!/usr/bin/env python3
"""Project rmb memory embeddings to 2D and write a self-contained Plotly HTML map.

Requires: numpy, scikit-learn, umap-learn, plotly, psycopg2-binary

Example (on host with RMB_DB_URL):
  python3 scripts/visualize_embeddings.py --category entities -o /tmp/memories-map.html

  RMB_DB_URL=postgres://... python3 scripts/visualize_embeddings.py
"""

from __future__ import annotations

import argparse
import os
import re
import sys
from datetime import datetime, timezone

try:
    import numpy as np
    import psycopg2
    import plotly.graph_objects as go
    import umap
except ImportError as e:
    print(
        "Missing dependency: %s\n"
        "Install with: pip install numpy scikit-learn umap-learn plotly psycopg2-binary"
        % e,
        file=sys.stderr,
    )
    sys.exit(1)


def parse_vector(text: str) -> np.ndarray:
    text = text.strip()
    if text.startswith("[") and text.endswith("]"):
        text = text[1:-1]
    if not text:
        return np.array([], dtype=np.float32)
    return np.array([float(x) for x in text.split(",")], dtype=np.float32)


def load_memories(conn, category: str | None) -> list[dict]:
    where = ["m.superseded_at IS NULL", "m.embedding IS NOT NULL"]
    params: list[str] = []
    if category:
        where.append("m.category = %s")
        params.append(category)
    sql = f"""
        SELECT m.uri,
               m.category,
               coalesce(m.slug, '') AS slug,
               left(coalesce(m.abstract, m.body, ''), 120) AS label,
               m.embedding::text AS embedding
        FROM memories m
        WHERE {' AND '.join(where)}
        ORDER BY m.uri
    """
    with conn.cursor() as cur:
        cur.execute(sql, params)
        rows = cur.fetchall()
    out = []
    for uri, cat, slug, label, emb in rows:
        out.append(
            {
                "uri": uri,
                "category": cat,
                "slug": slug,
                "label": label.replace("\n", " "),
                "embedding": parse_vector(emb),
            }
        )
    return out


def slug_prefix(slug: str, parts: int = 2) -> str:
    bits = [b for b in re.split(r"[-_]", slug) if b]
    if not bits:
        return "(no slug)"
    return "-".join(bits[:parts])


def build_html(memories: list[dict], output: str, title: str) -> None:
    if not memories:
        raise SystemExit("No memories with embeddings matched the filter.")

    uris = [m["uri"] for m in memories]
    labels = [m["label"] or m["slug"] or m["uri"] for m in memories]
    slugs = [m["slug"] for m in memories]
    categories = [m["category"] for m in memories]
    prefixes = [slug_prefix(s) for s in slugs]

    X = np.vstack([m["embedding"] for m in memories])
    if X.shape[1] == 0:
        raise SystemExit("Embeddings are empty.")

    reducer = umap.UMAP(
        n_components=2,
        n_neighbors=15,
        min_dist=0.1,
        metric="cosine",
        random_state=42,
    )
    coords = reducer.fit_transform(X)

    # Color by slug prefix (rough visual clusters).
    unique_prefixes = sorted(set(prefixes))
    palette = [
        "#6366f1",
        "#22c55e",
        "#f97316",
        "#ec4899",
        "#14b8a6",
        "#eab308",
        "#8b5cf6",
        "#ef4444",
        "#06b6d4",
        "#84cc16",
    ]
    prefix_color = {
        p: palette[i % len(palette)] for i, p in enumerate(unique_prefixes)
    }
    colors = [prefix_color[p] for p in prefixes]

    hover = [
        f"<b>{labels[i]}</b><br>"
        f"uri: {uris[i]}<br>"
        f"category: {categories[i]}<br>"
        f"slug: {slugs[i] or '—'}"
        for i in range(len(uris))
    ]

    fig = go.Figure()
    fig.add_trace(
        go.Scatter(
            x=coords[:, 0],
            y=coords[:, 1],
            mode="markers",
            marker=dict(size=7, color=colors, opacity=0.85, line=dict(width=0.5, color="#fff")),
            text=hover,
            hoverinfo="text",
            name="memories",
        )
    )

    generated = datetime.now(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    fig.update_layout(
        title=f"{title} ({len(memories)} points, UMAP cosine)",
        template="plotly_dark",
        width=1200,
        height=800,
        legend=dict(orientation="h", yanchor="bottom", y=1.02),
        annotations=[
            dict(
                text=f"Generated {generated} · color = slug prefix",
                showarrow=False,
                xref="paper",
                yref="paper",
                x=0,
                y=-0.08,
                font=dict(size=11, color="#94a3b8"),
            )
        ],
        margin=dict(b=80),
    )

    fig.write_html(output, include_plotlyjs="cdn", full_html=True)
    print(f"Wrote {output} ({len(memories)} memories)")


def main() -> None:
    parser = argparse.ArgumentParser(description="Visualize rmb memory embeddings in 2D.")
    parser.add_argument(
        "--db-url",
        default=os.environ.get("RMB_DB_URL", ""),
        help="Postgres URL (default: RMB_DB_URL env)",
    )
    parser.add_argument(
        "--category",
        default="entities",
        help="Memory category filter (default: entities). Use 'all' for no filter.",
    )
    parser.add_argument(
        "-o",
        "--output",
        default="memory-embeddings.html",
        help="Output HTML path (default: memory-embeddings.html)",
    )
    args = parser.parse_args()

    if not args.db_url:
        print("Set RMB_DB_URL or pass --db-url", file=sys.stderr)
        sys.exit(1)

    category = None if args.category.lower() == "all" else args.category

    conn = psycopg2.connect(args.db_url)
    try:
        memories = load_memories(conn, category)
    finally:
        conn.close()

    title = f"rmb memories — {args.category}"
    build_html(memories, args.output, title)


if __name__ == "__main__":
    main()
