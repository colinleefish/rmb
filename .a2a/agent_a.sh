#!/bin/bash
# Agent A - sends tasks and receives results from Agent B
A2A_DIR="/Users/liguanghui/Virginia/colinleefish/mypast/.a2a"
OUTBOX="$A2A_DIR/a_to_b.json"
INBOX="$A2A_DIR/b_to_a.json"

rm -f "$OUTBOX" "$INBOX"

echo "╔══════════════════════════════════════════╗"
echo "║  AGENT A (Task Sender) — A2A Protocol   ║"
echo "╚══════════════════════════════════════════╝"
echo ""
echo "[$(date +%H:%M:%S)] Agent A online. Waiting for Agent B..."
echo ""

# Wait for Agent B to be ready
sleep 2

# Send task 1
echo "[$(date +%H:%M:%S)] → SENDING task #1 to Agent B"
echo '{"jsonrpc":"2.0","id":1,"method":"tasks/send","params":{"id":"task-001","message":{"role":"user","parts":[{"type":"text","text":"What is the capital of France?"}]}}}' > "$OUTBOX"
cat "$OUTBOX" | python3 -m json.tool 2>/dev/null || cat "$OUTBOX"
echo ""

# Wait for response
echo "[$(date +%H:%M:%S)] ⏳ Waiting for Agent B's response..."
while [ ! -f "$INBOX" ]; do sleep 0.5; done
echo "[$(date +%H:%M:%S)] ← RECEIVED from Agent B:"
cat "$INBOX" | python3 -m json.tool 2>/dev/null || cat "$INBOX"
echo ""
rm -f "$INBOX"

sleep 1

# Send task 2
echo "[$(date +%H:%M:%S)] → SENDING task #2 to Agent B"
echo '{"jsonrpc":"2.0","id":2,"method":"tasks/send","params":{"id":"task-002","message":{"role":"user","parts":[{"type":"text","text":"Now calculate: 15 * 37"}]}}}' > "$OUTBOX"
cat "$OUTBOX" | python3 -m json.tool 2>/dev/null || cat "$OUTBOX"
echo ""

# Wait for response
echo "[$(date +%H:%M:%S)] ⏳ Waiting for Agent B's response..."
while [ ! -f "$INBOX" ]; do sleep 0.5; done
echo "[$(date +%H:%M:%S)] ← RECEIVED from Agent B:"
cat "$INBOX" | python3 -m json.tool 2>/dev/null || cat "$INBOX"
echo ""
rm -f "$INBOX"

sleep 1

# Send task 3 - collaborative task
echo "[$(date +%H:%M:%S)] → SENDING task #3 (collaborative) to Agent B"
echo '{"jsonrpc":"2.0","id":3,"method":"tasks/send","params":{"id":"task-003","message":{"role":"user","parts":[{"type":"text","text":"List the Go files in internal/service/ directory"}]}}}' > "$OUTBOX"
cat "$OUTBOX" | python3 -m json.tool 2>/dev/null || cat "$OUTBOX"
echo ""

# Wait for response
echo "[$(date +%H:%M:%S)] ⏳ Waiting for Agent B's response..."
while [ ! -f "$INBOX" ]; do sleep 0.5; done
echo "[$(date +%H:%M:%S)] ← RECEIVED from Agent B:"
cat "$INBOX" | python3 -m json.tool 2>/dev/null || cat "$INBOX"
echo ""
rm -f "$INBOX"

echo ""
echo "[$(date +%H:%M:%S)] ═══ A2A SESSION COMPLETE ═══"
echo "Agent A: All 3 tasks completed successfully."
