<div align="center">
  <h1>MCA-Cascade</h1>
  <p><strong>Clear assignment decisions for Cascade hosts.</strong></p>
  <p><a href="./README.zh-CN.md">简体中文</a></p>
</div>

MCA-Cascade is a standard-library-only Go decision and client library for connecting Cascade hosts with MatchCore Arena assignments. It gives host integrations a small, explicit boundary for authenticated queries and routing decisions.

## Why MCA-Cascade

- **Minimal dependencies.** The library uses only the Go standard library.
- **Authenticated by design.** Assignment requests use an OPS Bearer token.
- **Bounded input handling.** Response parsing places explicit bounds on untrusted data.
- **Pure decisions.** Assignment data and target mappings produce outcomes without host side effects.
- **Honest failures.** Unavailable services return explicit errors instead of masquerading as no assignment.
- **Separate responsibilities.** The library performs no configuration loading or file I/O.

## Learn more

- [Armorial Wiki](https://wiki.mcnb.dev/) — ecosystem documentation
- [Armorial Contract](https://github.com/CoreMatch/HA-Contract) — interface contracts
- [Cascade](https://github.com/CoreMatch/Cascade) — host proxy project
- [MatchCoreArena-VelocityGate](https://github.com/CoreMatch/MatchCoreArena-VelocityGate) — Velocity-side routing plugin

## Scope

Host wiring belongs outside this library. MCA-Cascade is not a complete Cascade plugin and is not presented as a production deployment.

## License

Licensed under [AGPL-3.0](LICENSE).
