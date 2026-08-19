#!/usr/bin/env python3
"""Rewrite a CometBFT config.toml / Cosmos SDK app.toml pair for one localnet validator.

Edits are section-aware so keys that appear in several sections (laddr, address,
enable) are unambiguous.
"""
import argparse
import re


def set_key(text: str, section: str, key: str, value: str) -> str:
    """Set `key = value` inside `[section]` ("" = top-level, before the first section)."""
    lines = text.splitlines()
    current = ""
    pattern = re.compile(rf"^\s*#?\s*{re.escape(key)}\s*=")
    found = False
    for idx, line in enumerate(lines):
        header = re.match(r"^\s*\[([^\]]+)\]\s*$", line)
        if header:
            current = header.group(1)
            continue
        if current == section and pattern.match(line):
            lines[idx] = f"{key} = {value}"
            found = True
            break
    if not found:
        raise SystemExit(f"key {key!r} not found in section [{section}]")
    return "\n".join(lines) + "\n"


def main() -> None:
    p = argparse.ArgumentParser()
    p.add_argument("--config", required=True)
    p.add_argument("--app", required=True)
    p.add_argument("--rpc-port", required=True)
    p.add_argument("--p2p-port", required=True)
    p.add_argument("--pprof-port", required=True)
    p.add_argument("--grpc-port", required=True)
    p.add_argument("--api-port", required=True)
    p.add_argument("--peers", default="")
    p.add_argument("--denom", default="upin")
    p.add_argument("--timeout-commit", default="1s")
    args = p.parse_args()

    cfg = open(args.config).read()
    cfg = set_key(cfg, "rpc", "pprof_laddr", f'"localhost:{args.pprof_port}"')
    cfg = set_key(cfg, "rpc", "laddr", f'"tcp://0.0.0.0:{args.rpc_port}"')
    cfg = set_key(cfg, "rpc", "cors_allowed_origins", '["*"]')
    cfg = set_key(cfg, "p2p", "laddr", f'"tcp://0.0.0.0:{args.p2p_port}"')
    cfg = set_key(cfg, "p2p", "persistent_peers", f'"{args.peers}"')
    cfg = set_key(cfg, "p2p", "addr_book_strict", "false")
    cfg = set_key(cfg, "p2p", "allow_duplicate_ip", "true")
    cfg = set_key(cfg, "consensus", "timeout_commit", f'"{args.timeout_commit}"')
    open(args.config, "w").write(cfg)

    app = open(args.app).read()
    app = set_key(app, "", "minimum-gas-prices", f'"0.001{args.denom}"')
    app = set_key(app, "api", "enable", "true")
    app = set_key(app, "api", "swagger", "true")
    app = set_key(app, "api", "address", f'"tcp://localhost:{args.api_port}"')
    app = set_key(app, "grpc", "address", f'"localhost:{args.grpc_port}"')
    open(args.app, "w").write(app)


if __name__ == "__main__":
    main()
