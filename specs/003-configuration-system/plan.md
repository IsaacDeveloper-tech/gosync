# Configuration System Implementation Plan

## Scope

Implement `gosync configure`, strict global JSON configuration, safe configuration replacement, single-process configuration ownership, startup configuration loading, configurable non-overlapping watch intervals, and bidirectional or source-authoritative unidirectional synchronization. The implementation shall satisfy RF-01 through RF-32. BACKUP execution, configuration profiles, non-interactive configuration, live reload, remote synchronization settings, and logging policy settings remain excluded.

## Constitution Alignment

| Constitutional Principle | Plan Compliance | RF Coverage |
| --- | --- | --- |
| Go and standard library by default | Use Go standard-library facilities for JSON, terminal streams, filesystem inspection, application-data lookup, timing, and process-safe storage coordination. | RF-01 through RF-32 |
| CLI without graphical interfaces or web services | Keep configuration in the `gosync configure` terminal workflow and the existing `watch` command. | RF-01, RF-02, RF-04 through RF-08, RF-22 through RF-25 |
| Handle every I/O and configuration error | Distinguish absent, invalid, unsupported, inaccessible, interrupted, and conflicting configuration states and propagate each defined failure. | RF-08, RF-19 through RF-28, RF-31 |
| Small, deterministic, English-named functions | Separate parsing, prompting, validation, storage, ownership, policy selection, scheduling, and synchronization planning; inject streams, waits, filesystem boundaries, and logging in tests. | RF-01 through RF-32 |
| Tests for modifiable behavior | Add focused unit tests and filesystem, CLI, scheduling, and synchronization integration tests for every changed behavior. | RF-01 through RF-32 |
| Successful test execution | Run the complete project test suite before integration. | RF-01 through RF-32 |

## Modules

| Module | Responsibility | RF Coverage |
| --- | --- | --- |
| Command parser | Recognize `configure`, reject additional arguments, and validate `watch` arguments before configuration access. | RF-01, RF-02, RF-23 |
| Configuration domain | Define schema version, interval range, persisted synchronization modes, configuration snapshots, and load outcomes. | RF-03, RF-09, RF-10, RF-18, RF-19, RF-29, RF-30 |
| Interactive configurator | Ask for interval and mode, identify BACKUP as unavailable, repeat invalid questions, display a summary, request confirmation, and report cancellation. | RF-01, RF-03 through RF-08 |
| Application-data locator | Resolve the per-user GoSync `config.json` path and report location failures. | RF-18, RF-21, RF-26, RF-31 |
| Strict JSON codec | Encode the exact version-1 document and reject malformed, duplicate, unknown, missing, mis-capitalized, incorrectly typed, out-of-range, or unsupported values. | RF-18, RF-19, RF-25 |
| Configuration file inspector | Distinguish missing files from inaccessible paths and require an existing configuration entry to be a regular file. | RF-22, RF-25, RF-26, RF-27 |
| Atomic configuration store | Load complete configuration snapshots and replace `config.json` without exposing a partial document after interruption. | RF-18 through RF-21, RF-25 through RF-27 |
| Configuration ownership guard | Allow only one active `configure` flow to own configuration replacement and reject competing sessions without changing data. | RF-28 |
| Configuration service | Coordinate inspection, invalid-file recovery, interactive collection, confirmation, persistence, and error propagation for direct and watch-triggered configuration. | RF-07, RF-08, RF-20 through RF-26, RF-28 |
| Configuration root guard | Compare the canonical configuration path with both synchronization roots and reject equal or contained locations. | RF-31 |
| Synchronization policy selector | Route a valid startup snapshot to bidirectional or unidirectional behavior without reloading it during the process. | RF-09, RF-10, RF-15, RF-16, RF-29, RF-30 |
| Unidirectional planner | Require an existing source and produce source-authoritative copy, replacement, and deletion actions for an exact destination replica. | RF-10 through RF-16 |
| Synchronization executor integration | Reuse preflight, root safety, locked-file retry, action execution, final verification, and confirmed-state commit rules. | RF-11 through RF-17 |
| Watch scheduler | Synchronize immediately, wait after completion for the configured interval, and prevent cycle overlap. | RF-03, RF-29, RF-30 |
| Configuration logging integration | Emit configuration start, success, cancellation, and failure events without responses or JSON contents. | RF-32 |

## Data Model

### Persisted Configuration

- `schemaVersion`: integer fixed at `1`.
- `synchronizationIntervalSeconds`: integer from `1` through `86,400` inclusive.
- `synchronizationMode`: exact string `bidirectional` or `unidirectional`.

No additional properties are admitted. This model is the complete global JSON contract and the immutable snapshot used by one `watch` execution. RF coverage: RF-03, RF-09, RF-10, RF-18, RF-19, RF-29, RF-30.

### Interactive Configuration Draft

- Candidate synchronization interval.
- Candidate synchronization mode.
- Validation state for each answer.
- Explicit confirmation state.
- Cancellation state caused by rejection, absent input, interruption, or input error.

The draft cannot become active until every value is valid and the summary is confirmed. BACKUP exists only as an unavailable interactive choice and is not a persisted mode. RF coverage: RF-01, RF-03 through RF-08, RF-20.

### Configuration Load Outcome

- Missing: `config.json` does not exist.
- Valid: one complete version-1 configuration snapshot is available.
- Invalid: a regular file exists but violates the strict schema.
- Unsupported entry: the path is a symbolic link, directory, or special entry.
- I/O failure: the location or file cannot be inspected, read, or closed successfully.

Distinct outcomes allow `watch` to start first-time configuration, `configure` to repair invalid JSON, and all unsafe or inaccessible states to stop with the correct error. RF coverage: RF-19, RF-21, RF-22, RF-25, RF-26, RF-27.

### Configuration Replacement State

- Previously committed complete document, when one exists.
- Fully encoded candidate document.
- Replacement ownership held by the active configuration process.
- Commit result: previous document retained or candidate document committed.

Only complete documents may become `config.json`; a candidate is never considered active before replacement succeeds. RF coverage: RF-08, RF-20, RF-21, RF-28.

### Synchronization Policy

- Mode loaded at `watch` startup.
- Interval loaded at `watch` startup.
- First root role: peer in bidirectional mode, source in unidirectional mode.
- Second root role: peer in bidirectional mode, destination in unidirectional mode.

The policy remains immutable for one process so file changes cannot alter an active execution. RF coverage: RF-09, RF-10, RF-14, RF-16, RF-29, RF-30.

### Unidirectional Synchronization Result

- Source validation outcome.
- Destination creation requirement.
- Ordered copy, replacement, and deletion actions.
- Final source-destination equivalence result.
- Confirmed-state commit or preservation outcome.

The result captures exact mirroring while preserving the source and the existing safety guarantees. RF coverage: RF-11 through RF-17.

## Key Decisions

### Strict Versioned JSON

Decode one JSON object with exactly the three specified properties and validate property names, duplicates, types, values, and complete input consumption before creating a configuration snapshot.

Justification: strict decoding makes typos and unsupported schema changes visible instead of silently changing synchronization behavior.

Discarded alternative: permissive decoding that ignores unknown fields and accepts convertible values. It could accept misspelled properties, duplicate keys, or ambiguous mode and interval representations.

RF coverage: RF-18, RF-19, RF-25.

### BACKUP Is an Interactive Placeholder Only

Represent BACKUP in the question catalog as explicitly unavailable, but exclude it from the persisted synchronization-mode domain.

Justification: the user can see the planned capability while every saved configuration remains executable by the current system.

Discarded alternative: persist BACKUP and reject it later in `watch`. That would deliberately create a validly stored but unusable configuration.

RF coverage: RF-05, RF-06, RF-18, RF-19.

### Complete Interactive Session Before Persistence

Collect and validate both answers, display one summary, and require explicit confirmation before attempting storage replacement. Treat rejection, EOF, or interruption before confirmation as cancellation.

Justification: the existing configuration must not change from partial answers or an unconfirmed draft.

Discarded alternative: save each answer as soon as it is accepted. Cancellation could leave a mixture of previous and new settings.

RF coverage: RF-01, RF-03 through RF-08, RF-20.

### Invalid Configuration Is Recoverable Only Through Configure

Return an invalid load outcome without modifying the file. `watch` rejects that outcome; `configure` reports it and permits a confirmed replacement.

Justification: synchronization must not infer settings from invalid data, while the configuration command needs a controlled repair path.

Discarded alternative: automatically delete or overwrite invalid JSON. It would destroy evidence of the problem without explicit user confirmation.

RF coverage: RF-19, RF-25.

### Atomic Same-Location Replacement

Prepare and close a complete candidate beside `config.json`, then commit it through an operating-system atomic replacement within the same application-data location. Treat any unsupported or failed replacement as a configuration failure.

Justification: a same-location atomic commit ensures interruption exposes either the previous complete document or the new complete document, never partially written JSON.

Discarded alternative: truncate and rewrite `config.json` in place. A write failure or interrupted process could destroy the last valid configuration.

RF coverage: RF-20, RF-21.

### Exclusive Configuration Ownership

Acquire process-level ownership before an interactive configuration session can replace global settings and hold it through cancellation or completed persistence.

Justification: one writer prevents two sessions from racing between confirmation and replacement.

Discarded alternative: allow both sessions and accept the last writer. A user could unknowingly overwrite a configuration confirmed by another process.

RF coverage: RF-28.

### Inspect Before Reading

Inspect `config.json` without following symbolic links and continue decoding only when the entry is a regular file.

Justification: configuration must not be redirected to an unintended file or sourced from unsupported filesystem entries.

Discarded alternative: open the path directly and follow symbolic links. The apparent application-data path could resolve outside GoSync's controlled location.

RF coverage: RF-21, RF-26, RF-27.

### Validate Command Shape Before Configuration Access

Parse `configure` and `watch` argument counts before locating, loading, creating, or prompting for configuration.

Justification: configuration work must not occur for a command that cannot execute and invalid `configure` invocations must not change user data.

Discarded alternative: initialize configuration first and validate command arguments afterward. It could prompt or modify files before reporting invalid usage.

RF coverage: RF-01, RF-02, RF-22, RF-23.

### One Startup Snapshot Per Watch Process

Load and validate configuration once after command and root checks, then pass the resulting immutable policy through all watch cycles.

Justification: one snapshot keeps mode and interval consistent throughout a running process and applies changes at a predictable next execution.

Discarded alternative: reload before each cycle. An edit could change direction or timing in the middle of a running synchronization session.

RF coverage: RF-09, RF-10, RF-29, RF-30.

### Completion-Based Scheduling

Run the first cycle immediately and start the configured wait only after that cycle completes; do not start a second cycle while one is active.

Justification: completion-based waits make the interval predictable and prevent concurrent mutations of the same synchronization roots.

Discarded alternative: fixed ticker events independent from synchronization completion. Long operations could queue checks or require overlapping cycles.

RF coverage: RF-03, RF-29, RF-30.

### Separate Unidirectional Planning With Shared Safety and Execution

Build an exact-mirror plan directly from source and destination inventories, always select the source for differences, and reuse existing root validation, preflight, retries, execution, verification, and state commit behavior.

Justification: direction-specific planning avoids bidirectional conflict questions while retaining established filesystem protections.

Discarded alternative: reuse the bidirectional planner unchanged. It could infer destination-to-source actions or request conflict decisions despite source authority.

RF coverage: RF-10 through RF-17.

### Existing Source Is Mandatory

Reject unidirectional synchronization before destination mutation when the source does not exist; create a missing destination only after source and preflight checks pass.

Justification: an absent source must not be interpreted as an empty authoritative tree that erases destination data.

Discarded alternative: create an empty source and mirror it. That could remove every destination item without a valid source snapshot.

RF coverage: RF-11, RF-12, RF-15.

### Configuration Must Remain Outside Synchronization Roots

Compare the canonical `config.json` location with canonical roots before synchronization and reject equal or containing root selections.

Justification: global application state must not be copied, removed, or classified as user synchronization content.

Discarded alternative: silently exclude `config.json`. Hidden exclusions would weaken the promise that selected roots represent complete synchronized contents.

RF coverage: RF-31.

### Configuration Logging Uses Event-Only Context

Emit lifecycle outcomes through the log system without passing prompt responses, serialized JSON, interval values, or mode values as log context.

Justification: operational diagnostics need to show configuration success or failure without duplicating configuration data in logs.

Discarded alternative: log complete prompts and saved JSON. It would violate RF-32 and create unnecessary copies of user configuration.

RF coverage: RF-32.

## Execution Sequences

### Direct Configure Command

1. Parse `gosync configure` and reject additional arguments before configuration access. RF-01, RF-02.
2. Initialize configuration lifecycle logging. RF-32.
3. Resolve the per-user `config.json` location and acquire exclusive configuration ownership. RF-18, RF-21, RF-28.
4. Inspect any existing path as missing, regular, unsupported, or inaccessible. RF-21, RF-26, RF-27.
5. Decode valid JSON or report invalid JSON while allowing the repair flow to continue. RF-19, RF-25.
6. Request and validate interval and mode until valid answers or cancellation. RF-03 through RF-06, RF-08.
7. Display the complete summary and require explicit confirmation. RF-07, RF-08.
8. Encode the strict version-1 document and atomically replace `config.json`. RF-18, RF-20, RF-21.
9. Report and log success, cancellation, or failure, then release ownership. RF-21, RF-28, RF-32.

### Watch Startup

1. Validate the `watch` command and exactly two root arguments. RF-23.
2. Normalize roots and initialize logging under the existing log-system contract. RF-15, RF-32.
3. Resolve the canonical configuration path and reject overlap with either synchronization root. RF-21, RF-31.
4. Inspect and load one strict configuration snapshot. RF-18, RF-19, RF-26, RF-27, RF-30.
5. If configuration is missing, acquire ownership and run the interactive flow; continue only after confirmed persistence. RF-22, RF-24, RF-28.
6. Select bidirectional or unidirectional synchronization from the startup snapshot. RF-09, RF-10, RF-30.
7. Run synchronization immediately, wait the configured interval after completion, and repeat without overlap. RF-29, RF-30.

### Unidirectional Synchronization Cycle

1. Validate that roots are distinct and non-nested and that the source exists. RF-11, RF-15.
2. Preflight the source and any existing destination for unsupported entries before mutation. RF-15.
3. Create a missing destination only after successful source and preflight checks. RF-12, RF-15.
4. Compare source and destination and generate source-authoritative copy, replacement, and deletion actions without conflict prompts. RF-13, RF-14, RF-16.
5. Execute actions with existing filesystem-error and locked-file behavior. RF-15.
6. Verify exact source-destination contents and commit confirmed state only after complete success. RF-13, RF-17.

## Test Strategy

### Unit Tests

- Command parsing accepts only argument-free `configure` and validates `watch` arguments before any configuration callback. RF-01, RF-02, RF-23.
- Interval parsing accepts `1` and `86400`, rejects every boundary violation and invalid representation, and requests another answer. RF-03, RF-04.
- Mode prompting accepts exact persisted modes, labels BACKUP unavailable, and repeats BACKUP or unknown selections. RF-05, RF-06.
- Summary confirmation saves only an accepted complete draft; rejection, EOF, and interruption cancel without persistence. RF-07, RF-08.
- Strict JSON tests accept only the exact version-1 schema and reject malformed input, trailing values, duplicates, unknown or missing fields, capitalization changes, invalid types, ranges, versions, and modes. RF-18, RF-19.
- Load-outcome tests distinguish missing, invalid, unsupported-entry, and I/O-failure states. RF-21, RF-22, RF-25, RF-26, RF-27.
- Configure orchestration permits confirmed replacement of invalid JSON while watch loading rejects it. RF-25.
- Ownership tests allow one active configuration session and reject a second without invoking storage replacement. RF-28.
- Root-guard tests reject a configuration path equal to or contained within either canonical synchronization root. RF-31.
- Scheduler tests run immediately, wait only after completion, use the configured interval, and never invoke overlapping cycles. RF-03, RF-29, RF-30.
- Policy tests retain one startup snapshot even when the underlying store changes. RF-09, RF-10, RF-30.
- Unidirectional planning tests copy source-only entries, update differing entries, delete destination-only entries, and select the source for file-content and file-directory conflicts. RF-10, RF-13, RF-14, RF-16.
- Lifecycle logging tests emit start, success, cancellation, and failure without answer or JSON fields. RF-32.

### Filesystem Integration Tests

- A confirmed configuration creates the exact `config.json` document in an isolated application-data directory and can be loaded on the next execution. RF-18, RF-20, RF-30.
- Replacing an existing configuration leaves the complete new document after success. RF-20.
- Injected interruption points around candidate creation, close, and replacement leave either the previous complete JSON or the new complete JSON. RF-20, RF-21.
- Location, inspection, read, create, write, replace, and close failures return explanatory errors and preserve the last complete configuration. RF-21, RF-26.
- Symbolic links, directories, and special entries at `config.json` are rejected without following or modifying them. RF-27.
- Concurrent configuration attempts demonstrate that only the owner can replace `config.json`. RF-28.
- A configuration path inside either synchronization root blocks watch startup before synchronization changes. RF-31.

### CLI Integration Tests

- `gosync configure` rejects extra arguments without prompting or touching configuration. RF-02.
- A complete interactive session repeats invalid interval and mode answers, labels and rejects BACKUP, displays a summary, confirms, and persists. RF-01, RF-03 through RF-07, RF-18.
- Rejection and EOF cancel direct and watch-triggered configuration without changing the previous document. RF-08, RF-24.
- Invalid `watch` arguments show usage without reading or creating configuration. RF-23.
- Valid `watch` arguments with missing configuration start configure and synchronize only after confirmed persistence. RF-22, RF-24.
- Invalid JSON blocks watch and remains unchanged; direct configure reports it and permits confirmed replacement. RF-19, RF-25.
- Configuration I/O and interactive-input errors stop the requested operation and preserve the previous configuration. RF-21, RF-26.
- Configuration lifecycle outcomes appear in logs without responses or serialized JSON. RF-32.

### Synchronization Integration Tests

- Bidirectional mode preserves the existing synchronization behavior while using the configured interval. RF-09, RF-29.
- Unidirectional mode rejects a missing source without modifying the destination. RF-11.
- Unidirectional mode creates a missing destination and initially replicates the complete source. RF-12, RF-13.
- Source additions, modifications, deletions, and type changes make the destination exact without changing the source or requesting conflict decisions. RF-13, RF-14, RF-16.
- Equal or nested roots and unsupported entries are rejected before unidirectional mutations. RF-15.
- Locked files retry, other filesystem errors stop, and failed cycles preserve confirmed state. RF-15, RF-17.
- Successful exact mirroring verifies both inventories and commits confirmed state for subsequent mode changes. RF-13, RF-17.
- Long synchronization cycles do not overlap and are followed by the complete configured delay. RF-29.
- Editing `config.json` during watch does not affect the active process, while the next process uses the new complete snapshot. RF-30.

### Test Isolation

- Inject input, output, persistence, ownership, wait, synchronization, and logging boundaries so error paths are deterministic. RF-01 through RF-08, RF-20, RF-21, RF-28 through RF-30, RF-32.
- Override the application-data base in tests so no test reads, writes, locks, or replaces the user's real `config.json`. RF-18 through RF-28, RF-31.
- Use temporary source and destination roots for every unidirectional behavior and verify the source remains byte-for-byte unchanged. RF-10 through RF-17.
- Use bounded fake waits rather than real intervals for scheduler tests. RF-03, RF-29, RF-30.

### Verification

Run `go test ./...` successfully before integration, satisfying the constitution and exercising RF-01 through RF-32.

## RF Traceability

| Requirement | Planned Modules | Planned Test Areas |
| --- | --- | --- |
| RF-01 | Command parser, interactive configurator | Command and interactive CLI tests |
| RF-02 | Command parser | Invalid configure command tests |
| RF-03 | Configuration domain, interactive configurator, watch scheduler | Interval boundary and scheduler tests |
| RF-04 | Interactive configurator | Invalid interval loop tests |
| RF-05 | Interactive configurator | Mode presentation tests |
| RF-06 | Interactive configurator | Invalid and BACKUP mode tests |
| RF-07 | Interactive configurator, configuration service | Summary and confirmation tests |
| RF-08 | Interactive configurator, configuration service, replacement state | Cancellation, EOF, and interruption tests |
| RF-09 | Synchronization policy selector | Bidirectional regression and interval tests |
| RF-10 | Configuration domain, policy selector, unidirectional planner | Root-role and policy tests |
| RF-11 | Unidirectional planner, executor integration | Missing-source integration tests |
| RF-12 | Unidirectional planner, executor integration | Missing-destination integration tests |
| RF-13 | Unidirectional planner, executor integration | Exact-mirror integration tests |
| RF-14 | Synchronization policy, unidirectional planner | Source-preservation integration tests |
| RF-15 | Policy selector, executor integration | Safety, preflight, retry, and error tests |
| RF-16 | Unidirectional planner | Content and type conflict tests |
| RF-17 | Executor integration, unidirectional result | Verification and confirmed-state tests |
| RF-18 | Configuration domain, locator, JSON codec, store | Exact JSON and persistence tests |
| RF-19 | Strict JSON codec | Schema rejection table tests |
| RF-20 | Atomic configuration store, replacement state | Successful and interrupted replacement tests |
| RF-21 | Locator, store, configuration service | Full I/O failure tests |
| RF-22 | File inspector, configuration service | Missing first-run configuration tests |
| RF-23 | Command parser | Argument-order tests |
| RF-24 | Configuration service | Watch-triggered configuration tests |
| RF-25 | JSON codec, file inspector, configuration service | Invalid watch and repair tests |
| RF-26 | Locator, file inspector, store | Location and read failure tests |
| RF-27 | Configuration file inspector | Symlink and special-entry tests |
| RF-28 | Ownership guard, configuration service | Concurrent configure tests |
| RF-29 | Policy selector, watch scheduler | Immediate, delayed, and non-overlap tests |
| RF-30 | Configuration snapshot, policy selector, watch scheduler | Startup snapshot and next-run tests |
| RF-31 | Application-data locator, configuration root guard | Equal and contained path tests |
| RF-32 | Configuration logging integration | Event and data-exclusion tests |
