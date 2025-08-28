# /self-check — Contract & build gate

Run a quick, read-only audit to verify we didn’t break interfaces.

Steps (ask before running tools):
- ajv: validate JSON contracts in docs/contracts/*.json
- go: `go build ./...`
- controller-gen: (if API changed) regenerate & diff CRDs under deployments/crds
- Optional: `golangci-lint run` if config present

Report:
- Summary of passes/fails
- If failures: list exact files and the minimal fix plan (do NOT apply unless asked)
