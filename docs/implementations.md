# Implementations

Added the GoSync project constitution with seven enforceable engineering principles.
The principles define stack, application boundaries, error handling, quality, tests, and modification limits.

Added the local bidirectional synchronization specification with EARS functional requirements.
The specification defines conflict, deletion, missing-directory, and unsupported-entry behavior and its boundaries.

Revised the local synchronization specification to resolve ambiguity in deletions, conflicts, and filesystem failures.
It now defines confirmed state, recovery, retry handling, path validation, and automated test completion criteria.

Added the active specification's implementation plan in English, including module responsibilities and data models.
The plan maps each module and test scenario to the local synchronization functional requirements.

Added the active specification's ordered implementation task list in English.
Each task is estimated below 30 minutes and maps to functional requirements with a verifiable completion condition.

Added the `gosync watch <directory-a> <directory-b>` command contract to the active specification.
The plan and tasks now cover argument validation, immediate synchronization, and fixed five-second rechecks.

Completed task 1 by defining synchronization entry, comparison, action, and result domain types.
Added tests that verify the contracts represent file, directory, copy, deletion, success, and failure outcomes.

Completed task 2 by adding root path normalization and validation for equal or nested synchronization roots.
Added tests for normalized paths, identical roots, nested roots, sibling roots, and empty root arguments.

Completed task 3 by adding a read-only preflight scan for both synchronization roots.
Added tests for regular entries, missing roots, symbolic links, and unsupported filesystem entry modes.

Completed task 4 by creating missing synchronization roots while preserving existing directory contents.
Added tests for one or two missing roots and errors when a selected root is an existing file.

Completed task 5 by adding persistent confirmed-state storage keyed by the selected root pair.
Added tests for state loading, root-pair isolation, missing state, and preservation after incomplete synchronization.

Completed task 6 by adding recursive directory inventories with file content digests and modification times.
Added tests for inventory entries and comparisons that ignore metadata while detecting content and structure differences.

Completed task 7 by classifying one-sided entries as additions or deletions using confirmed state.
Added tests for both directions, unchanged entries, absent entries, and deterministic action order.

Completed task 8 by comparing both inventories when confirmed state is unavailable.
Added tests for matching directories, authoritative-side selection, existing state, and decision errors.

Completed task 9 by resolving differing file contents using modification time or an explicit user choice on ties.
Added tests for both newer-file directions, equal-time conflicts, matching content, and decision errors.

Completed task 10 by listing directory contents and requiring an explicit side selection for file-directory conflicts.
Added tests for both conflict directions, sorted listings, missing choices, and decision errors.

Completed task 11 by generating complete synchronization plans from additions, deletions, and conflicts.
Added tests confirming ordered actions, required user decisions, directory contents, and rejection of partial plans.

Completed tasks 12-15: action execution, locked-file retries, final verification/state commit, and `watch` argument parsing.
Added tests for filesystem changes, retry notifications, confirmed-state integrity, and command usage validation.

Completed tasks 16-17 by adding the immediate five-second watch loop and connecting the full CLI synchronization flow.
Added tests for periodic execution, command integration, preflight, planning, execution, verification, and state commit.

Completed tasks 18-21 with unit and filesystem integration coverage for state, classification, validation, conflicts, and watch behavior.
The complete Go test suite was executed with coverage after adding the final test scenarios.

Reorganized production code under `src/` and moved all tests under `test/` using an external test package.
Kept a minimal root entrypoint so the existing `go run .` command remains available.

Added the log system specification from the agreed scope, error handling, retention, and privacy decisions.
It defines EARS requirements, exclusions, and completion criteria without prescribing implementation details.

Resolved the high-severity log specification findings through agreed destination fallback and failure behavior.
Clarified log placement, synchronization-root overlap, storage-limit failures, and protected data in errors and paths.

Completed log system task 1 by defining the required severity levels and stable event identifiers.
Added focused domain tests for all four severities and every event in the log event catalog.

Completed log system task 2 by adding timestamped structured entries with injected-clock construction.
Added tests covering timestamp, severity, event, message, context, and single clock invocation.

Completed log system task 3 by formatting entries as deterministic structured text records.
Added tests for RFC3339Nano timestamps, severity, stable event names, quoted messages, and ordered context.

Completed log system tasks 4-6 with protected-value replacement, contextual sanitization, and a console destination.
Added tests for secret redaction, file-content exclusion, complete writes, and propagated console write failures.

Completed log system tasks 7-11 with per-user location resolution, root-overlap detection, append, rotation, and retention.
Added filesystem tests for working-directory independence, 10 MB boundaries, preserved records, and four newest archives.

Completed log system tasks 12-27 with destination coordination, fallback behavior, synchronization instrumentation, and end-to-end privacy coverage.
Added unit and filesystem integration tests for destination failures, lifecycle events, changes, conflicts, retries, rotation, retention, and protected data.

Added the log system implementation plan with modules, data models, execution flow, and justified decisions.
Mapped RF-01 through RF-14 to planned responsibilities and unit, filesystem, and synchronization test coverage.

Added the dependency-ordered log system task list with work items estimated below 30 minutes.
Each task identifies its RF coverage and includes an independently verifiable completion condition.

Added the global JSON configuration specification based on the agreed interactive workflow and failure behavior.
It defines interval validation, bidirectional and source-to-destination modes, unavailable BACKUP handling, persistence, and reload scope.

Resolved all reviewed configuration-specification findings through agreed behavioral contracts and precedence rules.
Clarified strict JSON validation, atomic replacement, mode safety, scheduling, concurrent configuration, CLI validation, storage protection, and logging.

Added the configuration system implementation plan with modules, data models, execution sequences, and justified decisions.
Mapped RF-01 through RF-32 to planned responsibilities and unit, filesystem, CLI, synchronization, and isolation tests.

Added the dependency-ordered configuration system task list with work items estimated below 30 minutes.
Each task identifies its RF coverage and a verifiable completion condition spanning configuration, scheduling, and synchronization behavior.

Completed configuration system tasks 1-10 with the versioned domain, configure parsing, interactive validation, cancellation, location, and strict JSON codec.
Added tests for valid drafts, confirmation behavior, per-user config location, exact encoding, and malformed or unsupported JSON rejection.

Completed configuration system tasks 11-20 with filesystem inspection, typed load outcomes, atomic storage, ownership, configure orchestration, recovery, cancellation, and lifecycle logging.
Added tests for unsupported entries, storage preservation, exclusive ownership, invalid-JSON repair, cancellation, and sanitized configuration events.

Completed configuration system tasks 21-30 with configuration-root guarding, watch startup snapshots, mode routing, completion-based scheduling, and unidirectional planning.
Added tests for first-run configure, invalid startup configuration, immutable snapshots, source validation, destination creation, copies, updates, and deletions.

Completed configuration system tasks 31-41 with source-authoritative conflicts, safety and retry reuse, confirmed-state verification, CLI/watch orchestration, and integration coverage.
Added tests for exact mirroring, source preservation, failure recovery, configured scheduling, startup guards, and the complete project suite.

Defined specification 004 for a strictly unidirectional compressed backup mode using complete ZIP versions and configurable count-based retention.
Specified atomic failure preservation, archive verification, dedicated destination ownership, unsupported-entry handling, configuration compatibility, and restoration exclusions.

Refined specification 004 after QA review to remove cross-spec conflicts and define conditional configuration schemas, exclusive source ownership, stable snapshots, and process concurrency.
Clarified interrupted artifacts, prior-version integrity, deterministic retention, blocked files, empty backups, metadata scope, failure rollback, and publication confirmation.

Added the compressed backup implementation plan with modular boundaries, transactional data models, execution sequences, and justified design alternatives.
Mapped RF-01 through RF-34 to configuration, archive, ownership, integrity, retention, scheduling, logging, and isolated test coverage.

Added 67 dependency-ordered compressed-backup implementation tasks, each estimated below 30 minutes with explicit RF coverage and a verifiable completion condition.
Sequenced configuration, paths, inventory, state, archive identity, ownership, ZIP handling, recovery, publication, retention, orchestration, and regression work.

Completed compressed-backup Configuration Foundation tasks 1-8 with schema version 2, conditional BACKUP retention, strict codec validation, interactive confirmation, and atomic persistence.
Added configuration tests for version compatibility, invalid schemas, retention boundaries, cancellation, and exact persisted JSON; `go test ./...` passes.

Completed compressed-backup Backup Policy And Paths tasks 9-14 with startup policy snapshots, canonical roots, source and destination validation, root overlap guards, and console-only logging overlap handling.
Added policy/path integration tests for aliases, mutation ordering, unsupported roots, configuration overlap, and logging destination behavior.

Completed compressed-backup Logical Inventory and Backup Set State tasks 15-25 with deterministic source inventories, strict state records, destination-keyed storage, typed load outcomes, and atomic replacement.
Added tests for unsupported and blocked entries, metadata-independent digests, state schema rejection, destination isolation, storage failures, and cancellation-safe persistence.
