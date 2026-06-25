## What & why

<!-- What does this change and why? Link any issue. -->

## Checklist

- [ ] Conventional Commit title (`feat:`, `fix:`, `docs:`, …)
- [ ] `go build ./... && go vet ./... && go test ./...` pass
- [ ] Docs updated in this PR if commands/flags/output changed (`SKILL.md`, docs site, landing)
- [ ] Schema golden regenerated if the grammar changed (`VABC_UPDATE_GOLDEN=1 go test ./internal/cli/`)
- [ ] Signed off (`git commit -s`, DCO)
