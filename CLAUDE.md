# CLAUDE.md — omnisignal

## PRISM Control

This repo's roadmap items are tracked in [prism-control](https://github.com/ProductBuildersHQ/prism-control). Use `prismctl work ready --repo github.com/plexusone/omnisignal` to find claimable work, and carry the `Refs: RMI-OMNISIGNAL-<NNN>` trailer on every commit.

## Ent Codegen (store/sqlite)

`store/sqlite/ent` is Ent-generated from `store/sqlite/ent/schema/*.go`. After editing a schema file, regenerate with:

```bash
go generate ./store/sqlite/ent/...
```

Never hand-edit generated files under `store/sqlite/ent/` (everything except `ent/schema/`, `generate.go`, and `tools.go`); commit generated output separately from schema changes, as `chore(codegen)`.
