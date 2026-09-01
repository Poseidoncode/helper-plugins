# SKILL: firmware-release-evidence

## Description
Collects git state, hardware target board, compiler execution details, test execution results, and generates deterministic `release-evidence.json` and `release-evidence.md` reports with SHA-256 binary manifests.

## When to Use
- When preparing a firmware delivery package or staging a release.
- When verifying that the local working tree is clean and builds are reproducible.
- Trigger phrases: "產生發版證據", "驗證韌體 checksum", "generate release evidence".

## Input Requirements
- Current git commit hash, branch name, and dirty working tree status.
- Build toolchain, command, and exit status.
- Generated binary / map / elf artifact paths.

## Output Schema
Produces standard structured files in `./build/artifacts/`:
- `release-evidence.json`: Machine-readable evidence with SHA-256 checksums.
- `release-evidence.md`: Formatted human-readable report.
