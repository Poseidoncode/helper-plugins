# Helper Official Plugins & Marketplace

This repository hosts the official curated plugins, skills, and MCP adapter configurations for the [Helper](https://github.com/Poseidoncode/Helper) AI assistant ecosystem.

## 📦 Included Plugins

| Plugin ID | Type | Description |
|---|---|---|
| [`firmware-release-evidence`](./plugins/firmware-release-evidence) | Skill | Deterministic release evidence generator (Git state, build artifacts, checksums). |
| [`platformio-community`](./plugins/platformio-community) | MCP | Curated PlatformIO adapter for discovery, build, flash preflight, and bounded serial monitor. |
| [`firmware-sbom`](./plugins/firmware-sbom) | MCP | CycloneDX & SPDX SBOM generator with vulnerability scanning. |
| [`firmware-secret-scan`](./plugins/firmware-secret-scan) | MCP | Read-only credential & secret scanner with automatic finding redaction. |
| [`github-release-workflow`](./plugins/github-release-workflow) | Skill | Drafts structured release notes and creates safe GitHub Release drafts. |

---

## 🚀 How to Use

In your Helper client:
1. Open **Marketplace** (Settings / Marketplace).
2. The official catalog `marketplace.json` is subscribed by default.
3. Select any plugin to install with one click.

To manually subscribe to this repository:
```text
https://raw.githubusercontent.com/Poseidoncode/helper-plugins/main/marketplace.json
```

---

## 🛠️ Contributing

1. Fork this repository.
2. Add your plugin under `plugins/<your-plugin-id>/` with a valid `helper.json` and `SKILL.md`.
3. Register your plugin entry in `marketplace.json`.
4. Open a Pull Request!

To create a new standalone plugin repository, use the [helper-plugin-template](https://github.com/Poseidoncode/helper-plugin-template).

## 📄 License
[Apache License 2.0](./LICENSE)
