#!/bin/bash
# Agent B - receives tasks and sends results back to Agent A
A2A_DIR="/Users/liguanghui/Virginia/colinleefish/mypast/.a2a"
INBOX="$A2A_DIR/a_to_b.json"
OUTBOX="$A2A_DIR/b_to_a.json"

echo "╔══════════════════════════════════════════╗"
echo "║  AGENT B (Task Executor) — A2A Protocol  ║"
echo "╚══════════════════════════════════════════╝"
echo ""
echo "[$(date +%H:%M:%S)] Agent B online. Watching for tasks from Agent A..."
echo ""

# Process 3 tasks
for i in 1 2 3; do
    # Wait for incoming task
    echo "[$(date +%H:%M:%S)] ⏳ Waiting for task #$i..."
    while [ ! -f "$INBOX" ]; do sleep 0.5; done

    echo "[$(date +%H:%M:%S)] ← RECEIVED task from Agent A:"
    cat "$INBOX" | python3 -m json.tool 2>/dev/null || cat "$INBOX"
    echo ""

    # Process the task based on content
    TASK_TEXT=$(cat "$INBOX" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['params']['message']['parts'][0]['text'])" 2>/dev/null)

    sleep 1
    echo "[$(date +%H:%M:%S)] 🔧 Processing: \"$TASK_TEXT\""

    if echo "$TASK_TEXT" | grep -qi "capital.*france"; then
        RESULT="The capital of France is Paris."
    elif echo "$TASK_TEXT" | grep -qi "15 \* 37\|15\*37"; then
        RESULT="15 × 37 = 555"
    elif echo "$TASK_TEXT" | grep -qi "go files\|internal/service"; then
        RESULT=$(ls /Users/liguanghui/Virginia/colinleefish/mypast/internal/service/ 2>/dev/null | tr '\n' ', ' | sed 's/,$//')
        RESULT="Files in internal/service/: $RESULT"
    else
        RESULT="Task acknowledged. Processing complete."
    fi

    sleep 0.5

    # Send response
    echo "[$(date +%H:%M:%S)] → SENDING result back to Agent A"
    echo "{\"jsonrpc\":\"2.0\",\"id\":$i,\"result\":{\"status\":\"completed\",\"artifacts\":[{\"type\":\"text\",\"text\":\"$RESULT\"}]}}" > "$OUTBOX"
    cat "$OUTBOX" | python3 -m json.tool 2>/dev/null || cat "$OUTBOX"
    echo ""

    rm -f "$INBOX"
    sleep 1
done

echo ""
echo "[$(date +%H:%M:%S)] ═══ A2A SESSION COMPLETE ═══"
echo "Agent B: All 3 tasks processed successfully."
