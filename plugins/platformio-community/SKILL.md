# SKILL: platformio-community

## Description
Curated MCP adapter for PlatformIO: provides device discovery, target environment inspection, build, flash preflight check, and bounded serial monitor.

## Safety Guidelines
- **Flash Preflight**: Always verify serial port, target board, and binary SHA-256 before flashing.
- **Fail-Safe**: Hardware write failures must never auto-retry.
- **Bounded Monitor**: Serial monitor enforces duration timeouts and max lines limits.
