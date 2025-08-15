# /e2e-run — Kind based E2E

Preconditions:
- kind, kubectl available
- bin/webhook-manager(.exe) built or `go build ./cmd/webhook-manager`

Run:
- Bash("bash hack/run-e2e.sh")
- Wait until all pods Ready in the test namespace
- Port-forward if needed, collect logs to .excellence-reports/e2e/

Verify:
- CRD admission webhooks respond 200/400 as expected
- Porch package mutation applied
- Teardown only if asked

Output a short PASS/FAIL with links to logs.
