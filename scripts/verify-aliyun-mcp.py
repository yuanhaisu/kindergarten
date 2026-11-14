#!/usr/bin/env python3
"""Verify the user-level Alibaba Cloud MCP server starts within Claude Code's handshake timeout.

Reads the MCP entry from ~/.claude.json, spawns it, performs the MCP
initialize + tools/list handshake, and prints startup time and tool names.

Exit codes:
  0  server started within --timeout and registered at least one tool
  1  startup exceeded --timeout, or no tools were registered, or handshake failed
"""

import argparse
import json
import os
import queue
import subprocess
import sys
import threading
import time


def load_entry(name: str) -> dict:
    path = os.path.expanduser("~/.claude.json")
    with open(path, encoding="utf-8") as f:
        config = json.load(f)
    servers = config.get("mcpServers", {})
    if name not in servers:
        raise SystemExit(f"MCP server '{name}' not found in {path}")
    return servers[name]


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--name", default="alibaba-cloud-ops-mcp-server")
    parser.add_argument("--timeout", type=float, default=30.0,
                        help="Claude Code handshake timeout in seconds (default: 30)")
    args = parser.parse_args()

    entry = load_entry(args.name)
    command = [entry["command"], *entry.get("args", [])]
    env = os.environ.copy()
    env.update(entry.get("env", {}))

    started = time.time()
    proc = subprocess.Popen(
        command,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        encoding="utf-8",
        bufsize=1,
        env=env,
    )
    stdout_queue: "queue.Queue[str]" = queue.Queue()
    stderr_lines: list[str] = []

    threading.Thread(
        target=lambda: [stdout_queue.put(line) for line in proc.stdout],
        daemon=True,
    ).start()
    threading.Thread(
        target=lambda: [stderr_lines.append(line.rstrip()) for line in proc.stderr],
        daemon=True,
    ).start()

    def send(payload: dict) -> None:
        proc.stdin.write(json.dumps(payload) + "\n")
        proc.stdin.flush()

    def wait_for(request_id: int, timeout: float) -> dict:
        deadline = time.time() + timeout
        while time.time() < deadline:
            try:
                line = stdout_queue.get(timeout=1)
            except queue.Empty:
                continue
            try:
                message = json.loads(line)
            except json.JSONDecodeError:
                continue
            if message.get("id") == request_id:
                return message
        raise TimeoutError(
            f"no response for id={request_id} within {timeout:.0f}s; "
            f"stderr tail={stderr_lines[-6:]}"
        )

    try:
        send({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "verify-aliyun-mcp", "version": "0"},
            },
        })
        init = wait_for(1, args.timeout)
        elapsed = time.time() - started
        server_info = init.get("result", {}).get("serverInfo", {})

        send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})
        send({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
        tools_result = wait_for(2, args.timeout)
        tools = [tool["name"] for tool in tools_result.get("result", {}).get("tools", [])]

        print(f"server: {server_info.get('name')} {server_info.get('version')}")
        print(f"startup: {elapsed:.1f}s (limit {args.timeout:.0f}s)")
        print(f"tools: {len(tools)}")
        for name in tools:
            print(f"  - {name}")

        if elapsed > args.timeout:
            print(f"FAIL: startup exceeded {args.timeout:.0f}s", file=sys.stderr)
            return 1
        if not tools:
            print("FAIL: no tools registered", file=sys.stderr)
            return 1
        return 0
    except TimeoutError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        return 1
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


if __name__ == "__main__":
    sys.exit(main())
