# Log System Implementation Plan

## Scope

Implement structured operational logging for GoSync with simultaneous console and persistent-file output, severity classification, protected-data sanitization, bounded file rotation, and defined destination-failure behavior. The implementation shall satisfy RF-01 through RF-14 and shall not add remote collection, graphical viewers, metrics, tracing, alerting, log compression, encryption, archival backup, or user-configurable logging policies.

## Constitution Alignment

| Constitutional Principle | Plan Compliance | RF Coverage |
| --- | --- | --- |
| Go and standard library by default | Build formatting, application-data lookup, filesystem output, and rotation with Go standard-library packages only. | RF-03, RF-05, RF-08, RF-09 |
| CLI without graphical interfaces or web services | Keep all immediate log visibility in the existing command-line process. | RF-03, RF-07, RF-12, RF-13, RF-14 |
| Handle every I/O error | Return destination failures to the coordinator, disable failed file output, preserve an available destination, and fail synchronization when neither destination remains. | RF-07, RF-13, RF-14 |
| Small, deterministic, English-named functions | Separate entry creation, sanitization, formatting, destination coordination, location validation, and rotation. Inject time and I/O boundaries in tests. | RF-01 through RF-14 |
| Tests for modifiable behavior | Add unit and integration coverage for every logger behavior and each synchronization event integration point. | RF-01 through RF-14 |
| Successful test execution | Run the complete Go test suite before integration. | RF-01 through RF-14 |

## Modules

| Module | Responsibility | RF Coverage |
| --- | --- | --- |
| Log domain | Define severity levels, event identifiers, structured entries, safe context, and destination state. | RF-01, RF-02, RF-04, RF-05, RF-06, RF-10, RF-11 |
| Entry builder | Create entries for operation boundaries, synchronized changes, conflicts, retries, warnings, and errors using an injected clock. | RF-01, RF-02, RF-04, RF-05 |
| Protected-data sanitizer | Replace registered protected values and sensitive path or error fragments before an entry reaches any destination; reject file-content fields from log context. | RF-06, RF-10, RF-11 |
| Structured-text formatter | Convert sanitized entries into one consistent text record containing date and time, severity, event, and message. | RF-05 |
| Console destination | Write formatted records to one console stream and surface every write failure to the destination coordinator. | RF-03, RF-07, RF-12, RF-13, RF-14 |
| Application-data locator | Resolve GoSync's standard per-user application-data log location and determine whether it equals or falls within a selected synchronization root. | RF-03, RF-12 |
| Rotating file destination | Create and append to the active log, rotate at the size boundary, retain at most five total files, and return every creation, write, rotation, or retention failure. | RF-03, RF-07, RF-08, RF-09, RF-13, RF-14 |
| Destination coordinator | Send each formatted entry to every available destination, track destination availability, apply fallback behavior, and return a fatal logging error when no destination remains. | RF-03, RF-07, RF-12, RF-13, RF-14 |
| Synchronization instrumentation | Emit domain events from command startup, synchronization planning, execution, retries, conflicts, warnings, successful completion, and failure without changing existing user-facing errors. | RF-01, RF-02, RF-06 |

## Data Model

### Log Entry

- Timestamp supplied by the entry builder.
- Severity: `DEBUG`, `INFO`, `WARN`, or `ERROR`.
- Stable event identifier.
- Human-readable message.
- Sanitized contextual fields needed to identify the operation or affected path.

The entry is the single representation shared by both destinations so console and file output describe the same event. RF coverage: RF-01, RF-02, RF-04, RF-05.

### Event Catalog

- Synchronization started.
- Synchronization completed with its outcome.
- Change synchronized.
- Conflict detected.
- Retry started and retry completed.
- Warning raised.
- Operation failed with a sanitized cause.
- Persistent destination failed or was disabled.
- Console destination failed.

Stable event identifiers allow tests and users to distinguish event kinds without depending only on prose. RF coverage: RF-01, RF-02, RF-06, RF-07, RF-12, RF-13, RF-14.

### Protected Data

- Registered credentials, authentication secrets, and cryptographic keys that must be replaced wherever they occur.
- Sanitized path or name context containing no registered protected value.
- Sanitized error classification and context containing no file content or registered protected value.
- A fixed replacement marker that communicates omission without reproducing the protected value.

Raw file contents are not part of the logging data model and cannot be accepted as contextual fields. RF coverage: RF-06, RF-10, RF-11.

### Destination State

- Console availability.
- Persistent-file availability.
- Reason the persistent destination was disabled, retained only long enough to report it through an available destination.
- Fatal state when neither destination remains available.

Destination state lasts for one command execution. A disabled persistent destination is not retried during that execution. RF coverage: RF-07, RF-12, RF-13, RF-14.

### File Retention Policy

- Active log file in GoSync's per-user application-data location.
- Maximum active-file boundary of 10,000,000 bytes.
- Maximum of five files in total, including the active file.
- Ordered archive generations that identify the oldest retained file deterministically.

The fixed policy bounds storage while keeping the latest diagnostic history. RF coverage: RF-03, RF-08, RF-09.

## Key Decisions

### Synchronous Logging

Process each log entry synchronously before the producing operation continues.

Justification: destination failures must be observed immediately to apply RF-07, RF-13, and RF-14 deterministically, and ordered synchronous writes avoid loss during normal command termination.

Discarded alternative: an asynchronous queue and background writer. It would introduce queue saturation, shutdown flushing, event-loss, and delayed-error semantics not defined by the specification.

RF coverage: RF-03, RF-07, RF-13, RF-14.

### One Entry and One Formatter for Both Destinations

Build, sanitize, and format an entry once before sending the same record to console and file.

Justification: a shared record keeps timestamps, severity, event identity, message, and protected-data handling equivalent across destinations.

Discarded alternative: destination-specific entry construction or formatting. It could expose different information or produce inconsistent records for the same event.

RF coverage: RF-03, RF-04, RF-05, RF-10.

### Standard Per-User Application-Data Location

Resolve a dedicated GoSync log directory from the operating system's per-user application-data convention and keep its location independent from the process working directory.

Justification: logs must persist across invocations without becoming ordinary project or synchronization content.

Discarded alternative: store logs in the current working directory. Their location would depend on command invocation and could place them inside a synchronized root.

RF coverage: RF-03, RF-12.

### Console-Only Mode for Root Overlap

Compare the resolved log location with both canonical synchronization roots before persistent logging begins. If it equals or is contained by either root, report the condition and keep only console logging for that execution.

Justification: writing logs within a watched root could generate additional synchronization activity and an unbounded feedback cycle.

Discarded alternative: silently exclude the log directory from synchronization. It would change synchronization semantics and conflict with the expectation that selected roots have equivalent contents.

RF coverage: RF-12.

### Permanent File Disablement After a Persistent Failure

After any file creation, opening, writing, rotation, or retention failure, report the failure through the console and mark persistent logging unavailable for the rest of the execution.

Justification: this preserves synchronization while preventing repeated failures and avoiding file growth beyond the retention policy when rotation cannot complete.

Discarded alternative: continue appending after rotation or retention failure. It could exceed the specified size or file-count boundaries.

RF coverage: RF-07, RF-08, RF-09.

### Independent Destination Failure States

Treat console and file availability independently. Continue with either remaining destination, but propagate a logging error to stop synchronization when both are unavailable.

Justification: one destination can preserve observability after the other fails, while continuing with neither would make activity and failures unobservable.

Discarded alternative: stop after the first destination failure. It would interrupt synchronization despite another valid destination being available.

RF coverage: RF-07, RF-13, RF-14.

### Allowlisted Context and Registered Protected Values

Let event producers provide only defined contextual fields, register known credentials and secrets with the sanitizer, replace those values wherever they appear, sanitize paths and errors before formatting, and never pass file contents to the logger.

Justification: controlling admitted fields and known protected values provides a consistent privacy boundary across messages, errors, paths, and names.

Discarded alternative: write arbitrary error strings and use only pattern matching afterward. Patterns can miss secrets, cannot reliably distinguish file content, and can expose data before all cases are recognized.

RF coverage: RF-06, RF-10, RF-11.

### Decimal Rotation Boundary and Five Total Files

Interpret 10 MB as 10,000,000 bytes and count the active file among the five retained files. Rotate before an entry would cross the boundary when the active file already contains data; retain one active file and up to four archives.

Justification: decimal megabytes match the specification's `MB` unit, and counting the active file preserves the stated approximate 50 MB total.

Discarded alternative: retain five archives plus the active file. That would produce six files and exceed RF-09's total-file limit.

RF coverage: RF-08, RF-09.

### Preserve Existing User-Facing Errors

Add logging at operation boundaries without consuming, replacing, or changing errors already returned or displayed by synchronization behavior.

Justification: logging supplies diagnostic history but must not alter the established command failure contract.

Discarded alternative: replace operation errors with generic logger messages. It would remove actionable information from existing CLI behavior.

RF coverage: RF-01, RF-02, RF-06.

## Execution Sequence

1. Resolve the standard per-user GoSync log location. RF-03.
2. Compare the log location with both canonical synchronization roots. RF-12.
3. Initialize console output and, when no overlap exists, initialize the rotating file destination. RF-03, RF-07, RF-12.
4. Register known protected values before operational events can be emitted. RF-06, RF-10, RF-11.
5. Build each event with a timestamp, severity, stable identifier, message, and allowlisted context. RF-01, RF-02, RF-04, RF-05.
6. Sanitize the complete entry and format it as one structured text record. RF-05, RF-06, RF-10, RF-11.
7. Write the record synchronously to each available destination. RF-03.
8. If persistent output fails, report it through the console, disable the file destination, and continue synchronization. RF-07.
9. If console output fails while file output remains, record that failure in the file and continue synchronization. RF-13.
10. If neither destination remains available, propagate a logging error and stop synchronization. RF-14.
11. Before a file write crosses the size boundary, rotate archives and remove the oldest generation when required. RF-08, RF-09.

## Test Strategy

### Unit Tests

- Entry construction emits deterministic timestamps, stable event identifiers, messages, and all four severity values. RF-01, RF-02, RF-04, RF-05.
- Structured formatting emits one complete text record with date and time, severity, event, and message. RF-05.
- Registered credentials, authentication secrets, cryptographic keys, sensitive path segments, and sensitive error fragments are replaced before formatting. RF-06, RF-10, RF-11.
- File contents cannot enter the accepted contextual data and do not appear in formatted output. RF-10.
- Canonical path comparison detects a log location equal to or contained within either synchronization root. RF-12.
- The destination coordinator writes the same formatted record to both healthy destinations. RF-03.
- File destination failure reports to console, permanently disables file output for the execution, and leaves synchronization able to continue. RF-07.
- Console failure is recorded through a healthy file destination and leaves synchronization able to continue. RF-13.
- Simultaneous or sequential loss of both destinations returns a fatal logging error. RF-14.
- Rotation occurs at the 10,000,000-byte boundary and maintains one active file plus no more than four archives. RF-08, RF-09.
- Creation, opening, writing, rotation, and retention errors are returned to the coordinator. RF-07.

### Filesystem Integration Tests

- Normal logging creates the GoSync log directory and active file beneath an isolated per-user application-data location. RF-03.
- Console and file capture equivalent records for one synchronization event. RF-03, RF-05.
- Writes around the size boundary rotate without losing the triggering record. RF-08.
- Repeated rotations preserve the five newest files and remove the oldest generation. RF-09.
- A persistent path inside either test synchronization root produces a console warning and no log file writes. RF-12.
- Filesystem failures during initialization, append, rotation, and retention trigger console-only behavior for the remainder of the execution. RF-07.

### Synchronization Integration Tests

- Synchronization startup and successful or failed completion emit boundary and outcome events. RF-01, RF-06.
- Synchronized changes, conflicts, locked-file retries, warnings, and errors emit their corresponding events. RF-02.
- Logged operation failures retain sanitized diagnostic context while the existing user-facing error remains unchanged. RF-06, RF-10, RF-11.
- Loss of one destination does not stop synchronization; loss of both returns a logging failure and stops it. RF-07, RF-13, RF-14.

### Test Isolation

- Inject time so formatting and event-order assertions are deterministic. RF-01, RF-05.
- Use in-memory writers for destination-state unit tests and temporary directories for real rotation tests. RF-03, RF-07, RF-08, RF-09, RF-13, RF-14.
- Override the application-data base only within tests so no test reads, rotates, or deletes real user logs. RF-03, RF-08, RF-09, RF-12.
- Run protected-data cases as table-driven tests covering messages, errors, paths, names, and every output destination. RF-06, RF-10, RF-11.

### Verification

Run `go test ./...` successfully before integration, satisfying the constitution's test requirement and exercising RF-01 through RF-14.

## RF Traceability

| Requirement | Planned Modules | Planned Test Areas |
| --- | --- | --- |
| RF-01 | Log domain, entry builder, synchronization instrumentation | Entry construction and synchronization integration |
| RF-02 | Log domain, entry builder, synchronization instrumentation | Event catalog and synchronization integration |
| RF-03 | Console destination, application-data locator, rotating file destination, destination coordinator | Dual-destination unit and filesystem integration |
| RF-04 | Log domain, entry builder | Severity unit tests |
| RF-05 | Log domain, entry builder, structured-text formatter | Formatting unit and dual-destination integration |
| RF-06 | Protected-data sanitizer, synchronization instrumentation | Sanitized-error and existing-behavior integration |
| RF-07 | Console destination, rotating file destination, destination coordinator | Persistent-failure unit and filesystem integration |
| RF-08 | Rotating file destination | Size-boundary unit and filesystem integration |
| RF-09 | Rotating file destination | Retention-order unit and filesystem integration |
| RF-10 | Log domain, protected-data sanitizer | Protected-data unit and end-to-end output tests |
| RF-11 | Log domain, protected-data sanitizer | Sanitized path, name, and error tests |
| RF-12 | Application-data locator, console destination, destination coordinator | Root-overlap unit and filesystem integration |
| RF-13 | Console destination, rotating file destination, destination coordinator | Console-failure unit and synchronization integration |
| RF-14 | Console destination, rotating file destination, destination coordinator | Total-destination-failure unit and synchronization integration |
