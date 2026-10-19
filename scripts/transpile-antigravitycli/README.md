# Transpile a pinned Shell bundle for AGY

Use a clean Git clone pinned to the intended Shell release. Local source validation includes tracked, untracked and ignored files; rejected dirty sources leave the destination unchanged.

```bash
# Generate into a fresh scratch directory before deployment.
go run ./scripts/transpile-antigravitycli \
  --source-dir <clean-pinned-shell-clone> \
  --output-dir <fresh-output-directory> --strict

# Check the committed bundle against its provenance pin.
go run ./scripts/transpile-antigravitycli \
  --check --strict --source-dir <clean-pinned-shell-clone> \
  --output-dir internal/assets/antigravitycli
```

`--strict` rejects unmapped tools in generation and check mode. Failed strict generation can leave partial output: do not deploy it. Non-Markdown resources retain their bytes. The compatibility projection uses AGY-native tools, foreground waiting and explicit role/phase context rather than Pi session helpers.

For a destination inside a Git repository, `--check` checks the committed tree, not dirty candidate files. A successful drift check does not prove installation, model behavior or review approval. See the [antigravity CLI documentation](../../docs/antigravity-cli/README.md) — the [source index](../../docs/antigravity-cli/10-source-index.md) for pins and evidence levels and the [verification registry](../../docs/antigravity-cli/09-verification.md) for check results and limits.
