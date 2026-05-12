# bgbase

Shared Go types and utilities for **BGSpec** (BumbleGrid graph documents).

This module is split by concern:

| Path | Role |
|------|------|
| `floor/` | Floor index constants |
| `node/` | Node `data` shape and `bgKind` enum; Floor 0 extensions in `floor0.go` |
| `edge/` | Edge `data` shape and `bgRelation` enum; Floor 0 extensions in `floor0.go` |
| `graph/` | Root document (`bgspec`, `document`, `floors`) |
| `parser/` | Decoding (e.g. JSON) into `graph.BGSpecDocument` |
| `validate/` | Validation interfaces and rules (shared + per-floor) |

Legacy types may still live at the module root under `package document` until migration is finished; new code should prefer the packages above.
