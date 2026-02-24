# Logging Code Review Report

**Date**: 2026-02-24
**Reviewer**: Claude Opus 4.6 (Code Review Agent)
**Scope**: Logging quality, security, performance, and consistency across core operator files

---

## Files Reviewed

| File | Lines | Verdict |
|------|-------|---------|
| `controllers/networkintent_controller.go` | 755 | PASS with minor notes |
| `pkg/oran/a1/a1_adaptor.go` | 2287 | FAIL -- multiple gaps |
| `internal/loop/watcher.go` | ~2500+ | PASS with medium issues |
| `cmd/intent-ingest/main.go` | 216 | PASS with minor issues |
| `internal/ingest/handler.go` | 406 | PASS |
| `pkg/logging/logger.go` | 305 | PASS with design notes |

---

## 1. Completeness -- Error Path Logging

### PASS: controllers/networkintent_controller.go

Every error return in Reconcile, updateStatus, createA1Policy, deleteA1Policy, and
SetupWithManager is preceded by a `logger.ErrorEvent()` call. The finalizer cleanup
path at lines 206-219 correctly logs both the error and the retry count before
requeueing. The A1 endpoint missing-configuration path at line 531 logs before
returning.

### FAIL: pkg/oran/a1/a1_adaptor.go

**Finding LR-001 [HIGH]**: Six methods have zero structured logging on any path:

- `ListPolicyTypes` (line 546) -- no logger instantiated, no error-path logging.
- `DeletePolicyType` (line 591) -- no logger, no logging on error or success.
- `GetPolicyInstance` (line 686) -- no logger, errors returned silently.
- `ListPolicyInstances` (line 739) -- no logger, errors returned silently.
- `GetPolicyStatus` (line 859) -- no logger, errors returned silently.
- `DiscoverServices` (line 1251) -- no logger, errors returned silently.
- `NotifyPolicyEvent` (line 1287) -- no logger, errors returned silently.

These methods return `fmt.Errorf(...)` without any accompanying log statement.
When called from other logged methods (e.g., `ApplyPolicy` does log on
`GetPolicyStatus` failure), the caller partially compensates. However, when called
directly (e.g., from orchestrator event handlers), errors are swallowed silently.

**Finding LR-002 [MEDIUM]**: `UpdatePolicyInstance` (line 784) delegates to
`CreatePolicyInstance` but has no logging of its own. An error during
`json.Unmarshal` at line 788 returns silently.

**Finding LR-003 [MEDIUM]**: `createPolicyTypeWithRetry` (line 2184) and
`createPolicyInstanceWithRetry` (line 2225) perform HTTP operations with retry
logic but have zero logging of retry attempts, failures, or successes. This makes
debugging retry behavior in production impossible.

**Finding LR-004 [LOW]**: The `executeWithRetry` method (line 2058) does not log
individual retry attempts or the final failure after all retries are exhausted.
Operators will see no output explaining why a circuit breaker might be open.

### PASS: internal/loop/watcher.go (with notes)

Every significant error path in the watcher has either `logger.ErrorEvent` or
`logger.WarnEvent`. File processing results are consistently logged via
`IntentFileProcessed`. Worker lifecycle events are logged at debug level.

### PASS: cmd/intent-ingest/main.go

All startup errors call `logger.ErrorEvent` before `os.Exit(1)`. Health and intent
endpoints log request start, completion, and errors.

### PASS: internal/ingest/handler.go

The HandleIntent method logs at every decision point: method validation, content
type validation, body size validation, provider parsing, regex parsing, validation,
file write, and response encoding. The createPorchPackage helper logs both success
and failure.

---

## 2. Context -- Sufficient Identifying Information

### PASS: controllers/networkintent_controller.go

The `ReconcileStart` call at line 148 seeds the logger with `namespace` and `name`,
which propagate to all subsequent log calls in the reconciliation. HTTP requests
include `url`, `statusCode`, and `durationSeconds`. A1 policy operations include
`policyInstanceID`, `policyTypeID`, and `endpoint`.

### FAIL: pkg/oran/a1/a1_adaptor.go

**Finding LR-005 [MEDIUM]**: Each method that does have logging creates a new
`logging.NewLogger(logging.ComponentA1)` on every invocation (e.g., lines 430, 493,
616, 806, 892, 1037, ...). This is inefficient and means no request-scoped context
(e.g., correlation ID) flows through A1 operations. The logger should either be
stored on the `A1Adaptor` struct or passed via context.

**Finding LR-006 [MEDIUM]**: `handlePolicyCreate` (line 1427) calls
`o.registry.NotifyPolicyEvent()` on failure (line 1513) but does not log the
notification failure. If the SMO notification itself fails, both the original
A1 error and the notification error are lost.

### PASS: internal/loop/watcher.go

Worker ID, filename, attempt number, file size, and modification time are
consistently included. The shutdown sequence logs stage numbers (0-3) and grace
period values.

### PASS: internal/ingest/handler.go

Request IDs propagate via `WithRequestID`. Intent fields (type, target, namespace)
are included in acceptance logs. Correlation IDs are conditionally included.

---

## 3. Performance -- Hot Path Logging

### PASS: controllers/networkintent_controller.go

Debug-level logging uses `DebugEvent` (backed by `V(1).Info`) which is gated by the
global log level. The reconciliation duration is tracked via `time.Since(start)`.
The `defer` block on line 151-156 always runs but produces only one Info-level log.

### MEDIUM: pkg/oran/a1/a1_adaptor.go

**Finding LR-007 [MEDIUM]**: `NewLogger(logging.ComponentA1)` is called at the top
of every method body (e.g., lines 430, 493, 616, 806, 892, 1037, 1185, 1341, 1372,
1428, 1560, 1609, 1675, 1708). Each call constructs a new `Logger` struct. While
individually cheap, under high throughput with many policy operations, this adds
allocations per request. A struct-level logger field would eliminate this.

### PASS: internal/loop/watcher.go

File-level logging in `handleIntentFile` and `processIntentFileWithContext` uses
`DebugEvent` for frequent operations (file detection, debounce skips) and
`InfoEvent` only for significant state transitions. The `updateMetrics` method runs
on a 10-second ticker, not per-file.

### PASS: cmd/intent-ingest/main.go

The `/healthz` endpoint logs at Debug level (line 131), avoiding noise in production.
The `/intent` endpoint logs at Info level which is appropriate for business events.

---

## 4. Security -- Sensitive Data in Logs

### PASS: controllers/networkintent_controller.go

No passwords, tokens, or credentials are logged. The LLM processor URL is logged at
startup (line 726) which is acceptable for configuration visibility. The SSRF
validator is applied before URLs are used.

### FAIL: pkg/oran/a1/a1_adaptor.go

**Finding LR-008 [HIGH]**: The `SMOServiceRegistry` struct stores `APIKey` (line
167) and sends it via `X-API-Key` header (lines 1214, 1263, 1302). While the API
key is not currently logged in any log statement, the `SMOServiceRegistry.URL` field
is logged alongside operations. The struct field is exported (`APIKey`) and could be
inadvertently logged if the struct is ever passed to a `%+v` format string or a
debug dump. The field should be redacted or the struct should implement a custom
`String()` method that masks the key.

### MEDIUM: internal/loop/watcher.go

**Finding LR-009 [MEDIUM]**: The `Config` struct contains `MetricsUser` and
`MetricsPass` fields (lines 63-65). These are validated at line 172 and compared at
line 941 but never logged. However, the struct is serializable via JSON tags
(`json:"metrics_user"`, `json:"metrics_pass"`). If Config is ever marshaled for
debug logging, credentials would leak. Consider adding `json:"-"` to the password
field or implementing `MarshalJSON` to redact it.

### PASS: internal/ingest/handler.go

The request body is not logged verbatim. Only metadata (content type, body length,
intent type, target, namespace) appears in logs.

---

## 5. Consistency -- Logging Pattern Uniformity

### PASS: controllers/networkintent_controller.go

Consistently uses `logger.InfoEvent`, `logger.ErrorEvent`, `logger.DebugEvent` for
all operations. HTTP calls use `logger.HTTPRequest` and `logger.HTTPError`
symmetrically. Duration tracking uses `time.Since(start).Seconds()` throughout.

### FAIL: pkg/oran/a1/a1_adaptor.go

**Finding LR-010 [HIGH]**: The file exhibits an inconsistent pattern where some
methods have full structured logging (CreatePolicyType, CreatePolicyInstance,
DeletePolicyInstance, ApplyPolicy, RemovePolicy, RegisterService, processEvent,
executeWorkflow) while others have zero logging (ListPolicyTypes, DeletePolicyType,
GetPolicyInstance, ListPolicyInstances, GetPolicyStatus, DiscoverServices,
NotifyPolicyEvent). There is no systematic rule for which methods log.

**Finding LR-011 [MEDIUM]**: HTTP call logging is inconsistent. Some methods use
both `logger.HTTPError` and `logger.HTTPRequest` (e.g., CreatePolicyType at lines
461, 467), while others return bare `fmt.Errorf` (e.g., ListPolicyTypes at line
556). Duration tracking is present in some methods but absent in others.

### FAIL: internal/loop/watcher.go

**Finding LR-012 [MEDIUM]**: The file uses a mix of `log.Printf` (standard library)
and `w.logger.*` (structured logger). There are 12 occurrences of `log.Printf`:

- `Config.Validate()` (lines 77, 84, 96, 102, 119, 123, 127) -- 7 occurrences
- `NewWatcherWithConfig` (lines 621, 627, 645, 721) -- 4 occurrences
- `Close()` nil check (line 2070) -- 1 occurrence

The comments state "Use standard library log here since we don't have a logger
instance in Config.Validate()" which is a valid justification for the `Validate()`
method. However, lines 621, 627, 645, and 721 occur during `NewWatcherWithConfig`
where the structured logger is being constructed. These could be deferred and logged
after the logger is initialized.

### MEDIUM: cmd/intent-ingest/main.go

**Finding LR-013 [LOW]**: Line 183 uses `fmt.Printf` for the "Ready to accept
intents" message instead of the structured logger. While this is cosmetically
intentional (user-facing startup banner), it breaks structured log parsing in
production log aggregators.

---

## 6. Testing -- Logging Code Coverage

### pkg/logging/logger.go Test Coverage

The test file `pkg/logging/logger_test.go` (130 lines) covers:

| Method | Covered | Notes |
|--------|---------|-------|
| `NewLogger` | Yes | |
| `NewLoggerWithLevel` | Yes | |
| `WithValues` | Yes | |
| `WithName` | No | Not tested |
| `WithRequestID` | Yes | Via chaining test |
| `WithNamespace` | Yes | Via chaining test |
| `WithResource` | Yes | |
| `WithIntent` | Yes | |
| `WithError` | No | Not tested (nil and non-nil cases) |
| `InfoEvent` | Yes | |
| `ErrorEvent` | Yes | Via ReconcileError |
| `DebugEvent` | Yes | |
| `WarnEvent` | No | Not directly tested |
| `ReconcileStart` | Yes | |
| `ReconcileSuccess` | Yes | |
| `ReconcileError` | Yes | |
| `HTTPRequest` | Yes | |
| `HTTPError` | Yes | |
| `A1PolicyCreated` | Yes | |
| `A1PolicyDeleted` | Yes | |
| `IntentFileProcessed` | Yes | |
| `PorchPackageCreated` | Yes | |
| `ScalingExecuted` | Yes | |
| `InitGlobalLogger` | Yes | Implicitly via NewLogger |
| `GetLogLevel` | Yes | Comprehensive env var test |
| `isDevelopment` | No | Not tested |

**Finding LR-014 [LOW]**: `WithName`, `WithError`, `WarnEvent`, and `isDevelopment`
are not directly tested. While these are trivial wrappers, `WithError` has a nil
guard that should be verified.

---

## 7. Additional Checks

### Are all fmt.Printf calls replaced?

**Finding LR-015 [MEDIUM]**: Within the reviewed files:

| File | `fmt.Printf` / `fmt.Println` Count | Location |
|------|-------------------------------------|----------|
| `controllers/networkintent_controller.go` | 0 | Clean |
| `pkg/oran/a1/a1_adaptor.go` | 0 | Clean |
| `internal/loop/watcher.go` | 0 | Clean (uses `fmt.Sprintf` for string formatting only) |
| `cmd/intent-ingest/main.go` | 1 | Line 183: `fmt.Printf("\nReady to accept intents...")` |
| `internal/ingest/handler.go` | 0 | Clean |
| `pkg/logging/logger.go` | 2 | Lines 228-230: `fmt.Fprintf(os.Stderr, ...)` -- justified (bootstrapping) |

The `cmd/intent-ingest/main.go` instance is the only unjustified usage.

### Are all panics preceded by logging?

**Finding LR-016 [PASS]**: The only panic in the reviewed files is in
`pkg/logging/logger.go` line 231 (`InitGlobalLogger`). It is preceded by three
`fmt.Fprintf(os.Stderr, ...)` calls that output the error, log level, and config.
This is acceptable because the logger itself is unavailable at this point.

### Are all error returns logged?

**Summary by file**:

| File | Error returns logged | Error returns unlogged | Compliance |
|------|---------------------|----------------------|------------|
| `controllers/networkintent_controller.go` | 15/15 | 0 | 100% |
| `pkg/oran/a1/a1_adaptor.go` | ~25/65 | ~40 | ~38% |
| `internal/loop/watcher.go` | ~30/35 | ~5 (validation helpers) | ~86% |
| `cmd/intent-ingest/main.go` | 5/5 | 0 | 100% |
| `internal/ingest/handler.go` | 8/8 | 0 | 100% |
| `pkg/logging/logger.go` | N/A | N/A | N/A |

The A1 adaptor at 38% compliance is the primary concern.

### Are durations tracked for major operations?

| Operation | Duration Tracked | File |
|-----------|-----------------|------|
| Reconciliation | Yes (start/defer) | controller line 147/151 |
| LLM HTTP call | Yes (llmStart/llmDuration) | controller line 379/381 |
| A1 policy create | Yes (a1Start/a1Duration) | controller line 581/583 |
| A1 policy delete | Yes (deleteStart/deleteDuration) | controller line 660/662 |
| A1 CreatePolicyType | Yes | a1_adaptor line 431/458 |
| A1 CreatePolicyInstance | Yes | a1_adaptor line 617/649 |
| A1 DeletePolicyInstance | Yes | a1_adaptor line 807/828 |
| A1 GetPolicyType | Yes | a1_adaptor line 494/510 |
| A1 ListPolicyTypes | No | a1_adaptor line 546 |
| A1 DeletePolicyType | No | a1_adaptor line 591 |
| A1 GetPolicyInstance | No | a1_adaptor line 686 |
| A1 ListPolicyInstances | No | a1_adaptor line 739 |
| A1 GetPolicyStatus | No | a1_adaptor line 859 |
| A1 ApplyPolicy | Yes | a1_adaptor line 893 |
| A1 RemovePolicy | Yes | a1_adaptor line 1039 |
| A1 RegisterService | Yes | a1_adaptor line 1186 |
| A1 DiscoverServices | No | a1_adaptor line 1251 |
| A1 NotifyPolicyEvent | No | a1_adaptor line 1287 |
| Watcher file processing | Yes | watcher line 1777/1874 |
| Watcher polling scan | No (per-scan) | watcher line 1567 |
| Intent ingest request | Yes | main.go lines 128, 149 |
| Intent handler | Yes (IntentFileProcessed) | handler.go line 274 |

---

## Findings Summary

| ID | Severity | File | Description |
|----|----------|------|-------------|
| LR-001 | HIGH | a1_adaptor.go | 7 methods have zero structured logging |
| LR-002 | MEDIUM | a1_adaptor.go | UpdatePolicyInstance has no logging |
| LR-003 | MEDIUM | a1_adaptor.go | Retry helper methods have no logging |
| LR-004 | LOW | a1_adaptor.go | executeWithRetry does not log attempts |
| LR-005 | MEDIUM | a1_adaptor.go | Logger created per-call, no request context |
| LR-006 | MEDIUM | a1_adaptor.go | SMO notification failure not logged |
| LR-007 | MEDIUM | a1_adaptor.go | Per-method logger allocation overhead |
| LR-008 | HIGH | a1_adaptor.go | APIKey in exported struct risks log exposure |
| LR-009 | MEDIUM | watcher.go | MetricsPass in JSON-serializable Config struct |
| LR-010 | HIGH | a1_adaptor.go | Inconsistent logging: some methods logged, others not |
| LR-011 | MEDIUM | a1_adaptor.go | HTTP call logging inconsistent across methods |
| LR-012 | MEDIUM | watcher.go | 12 `log.Printf` calls mixed with structured logger |
| LR-013 | LOW | main.go | 1 `fmt.Printf` for startup banner |
| LR-014 | LOW | logger_test.go | WithName, WithError, WarnEvent untested |
| LR-015 | MEDIUM | main.go | fmt.Printf not fully replaced |
| LR-016 | PASS | logger.go | Panic preceded by stderr output |

### Severity Distribution

- **HIGH**: 3 findings (LR-001, LR-008, LR-010)
- **MEDIUM**: 9 findings (LR-002, LR-003, LR-005, LR-006, LR-007, LR-009, LR-011, LR-012, LR-015)
- **LOW**: 3 findings (LR-004, LR-013, LR-014)
- **PASS**: 1 finding (LR-016)

---

## Recommendations

### Priority 1 -- Address HIGH findings

1. **Add structured logging to all A1 adaptor methods** (LR-001, LR-010):
   Store a `logging.Logger` field on the `A1Adaptor` struct, initialized in
   `NewA1Adaptor`. Replace all per-method `logging.NewLogger(logging.ComponentA1)`
   calls with `a.logger`. Add logging to the 7 unlogged methods.

2. **Protect APIKey from accidental log exposure** (LR-008):
   Implement a `String()` method on `SMOServiceRegistry` that masks the `APIKey`
   field, or change the JSON tag to `json:"-"` to prevent marshaling.

### Priority 2 -- Address MEDIUM findings

3. **Add logging to retry helpers** (LR-003, LR-004):
   Log each retry attempt with attempt number, delay, and error in
   `executeWithRetry`. Log the final failure with total attempts.

4. **Propagate request context through A1 operations** (LR-005):
   Accept a logger parameter or store a per-request logger on the adaptor that
   carries correlation IDs.

5. **Protect MetricsPass from JSON serialization** (LR-009):
   Change the `MetricsPass` JSON tag to `json:"-"`.

6. **Convert remaining `log.Printf` to structured logging** (LR-012):
   For `NewWatcherWithConfig` (lines 621, 627, 645, 721), accumulate warnings
   during construction and log them via the structured logger after it is
   initialized. For `Config.Validate()`, the standard library usage is acceptable
   as documented.

7. **Replace `fmt.Printf` in main.go** (LR-013, LR-015):
   Convert `fmt.Printf` at line 183 of `cmd/intent-ingest/main.go` to
   `logger.InfoEvent`.

8. **Log SMO notification failures** (LR-006):
   In `handlePolicyCreate`, log errors returned from `NotifyPolicyEvent`.

### Priority 3 -- Address LOW findings

9. **Add missing logger test cases** (LR-014):
   Add tests for `WithName`, `WithError` (both nil and non-nil), `WarnEvent`,
   and `isDevelopment`.

10. **Add duration tracking to remaining A1 methods** (LR-011):
    Add `start := time.Now()` and duration logging to ListPolicyTypes,
    DeletePolicyType, GetPolicyInstance, ListPolicyInstances, GetPolicyStatus,
    DiscoverServices, and NotifyPolicyEvent.

---

## Design Observations

### Logger Architecture -- Strengths

The `pkg/logging/logger.go` design is well-structured:

- Uses `logr` interface with `zap` backend, following Kubernetes controller-runtime
  conventions.
- Component constants provide consistent identification across modules.
- Specialized methods (`ReconcileStart`, `HTTPRequest`, `A1PolicyCreated`, etc.)
  enforce consistent field names and reduce ad-hoc key-value pairs.
- `WithRequestID` and `WithResource` enable request-scoped context propagation.
- Debug vs Info level distinction prevents noise in production while enabling
  verbose output for troubleshooting.

### Logger Architecture -- Weaknesses

- **No `Warn` level in logr**: The `WarnEvent` method (line 137) prepends
  `"WARNING: "` to the message string as a workaround. This means warning-level
  logs cannot be filtered separately from info-level logs in log aggregation
  systems. Consider using `V(0).Info` with a `"level": "warn"` key-value pair
  instead, or switching to a logger that natively supports warn level.

- **No sampling or rate limiting**: Under extreme load (e.g., thousands of intent
  files per second), error logging in hot paths could itself become a bottleneck.
  The zap backend supports sampling, but it is not configured in `getZapConfig`.

- **No caller information at Info level**: The zap config includes `CallerKey` but
  callers may not appear at all verbosity levels depending on the zap level. This
  is fine for production but could be enhanced for debug builds.

---

**Review Status**: Complete
**Next Action**: Address HIGH findings (LR-001, LR-008, LR-010) before next release
