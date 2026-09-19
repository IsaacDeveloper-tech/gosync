# Compressed Backup Mode Implementation Plan

## Scope

Implement `BACKUP` as a third configured watch mode that treats the first root as an immutable source and the second as an exclusively owned local backup destination. Each successful cycle shall publish one independently usable ZIP, verify a stable logical source snapshot, retain 1 through 100 versions in confirmation order, recover owned interrupted work, reject foreign or corrupt destination state, and preserve the existing completion-based scheduler and logging contracts. The implementation shall satisfy RF-01 through RF-34. Restoration, incremental archives, encryption, metadata preservation, remote destinations, shared backup sets, and same-cycle locked-file retries remain excluded.

## Constitution Alignment

| Constitutional Principle | Plan Compliance | RF Coverage |
| --- | --- | --- |
| Go and standard library by default | Use `archive/zip`, strict JSON, SHA-256, cryptographic random identifiers, canonical filesystem paths, atomic same-directory replacement, and platform-specific standard-library system calls for process locks. | RF-05, RF-06, RF-13 through RF-19, RF-23 through RF-30 |
| CLI without graphical interfaces or web services | Extend only the existing `configure` questions and `gosync watch <directory-a> <directory-b>` workflow. | RF-01 through RF-04, RF-07, RF-31, RF-32 |
| Handle every I/O and configuration error | Model inspection, archive, state, ownership, cleanup, publication, verification, retention, logging, and lock failures as explicit outcomes and preserve confirmed history according to the specification. | RF-06, RF-08 through RF-17, RF-21 through RF-26, RF-29, RF-30, RF-34 |
| Small, deterministic, English-named functions | Separate configuration variants, path validation, ownership, state persistence, source inventory, ZIP writing, archive verification, destination classification, confirmation, and retention. Inject clocks, identifiers, waits, locks, filesystem failures, and logging in tests. | RF-01 through RF-34 |
| Tests for modifiable behavior | Add unit, filesystem, process-concurrency, CLI, scheduling, logging, and end-to-end backup tests for every changed contract. | RF-01 through RF-34 |
| Successful test execution | Run the complete Go test suite before integration. | RF-01 through RF-34 |

## Modules

| Module | Responsibility | RF Coverage |
| --- | --- | --- |
| Configuration domain | Add schema version 2, `backup` mode, the conditional retention field, the 1 through 100 range, and immutable backup policy snapshots while preserving valid version-1 configurations. | RF-01 through RF-06, RF-32 |
| Interactive configurator | Present BACKUP as available, request retention only for BACKUP, apply the default, repeat invalid answers, and include retention in the confirmation summary. | RF-01 through RF-04 |
| Strict configuration codec | Encode and decode the two exact version-2 property sets and retain strict version-1 decoding without accepting BACKUP in version 1. | RF-05, RF-06 |
| Synchronization policy selector | Route a valid startup snapshot to bidirectional, unidirectional, or backup behavior without changing the existing modes. | RF-01, RF-05 through RF-07, RF-31, RF-32 |
| Backup path validator | Canonicalize roots, require a readable source directory, validate destination type, reject equal or nested roots, and protect the configuration location. | RF-07 through RF-10, RF-12, RF-14, RF-21 |
| Backup logging policy integration | Apply console-only logging when the persistent log path overlaps either backup root and preserve fatal behavior when no log destination remains. | RF-11, RF-33, RF-34 |
| Destination ownership guard | Acquire one operating-system-backed exclusive lock keyed by canonical destination for the lifetime of the watch process and reject competing processes. | RF-13, RF-22, RF-24, RF-33 |
| Backup set state store | Atomically persist source binding, destination identity, confirmed archive records, confirmation order, integrity digests, and pending retention removals in GoSync application data. | RF-14 through RF-17, RF-24, RF-26, RF-28 through RF-30 |
| Source inventory builder | Produce deterministic logical inventories containing relative path, entry type, and file-content digest while rejecting unsupported entries and ignoring excluded metadata. | RF-08, RF-18 through RF-21, RF-23, RF-25 |
| Destination inspector | Classify every destination entry as confirmed, pending removal, owned unconfirmed, or foreign by reconciling filesystem entries, archive identity, and backup set state. | RF-12, RF-14 through RF-17, RF-24, RF-26 |
| Archive identity codec | Encode and strictly decode a versioned GoSync identity in each ZIP without adding a restorable source entry. | RF-14 through RF-17, RF-26 through RF-28 |
| Backup name generator | Create collision-resistant `.zip` names containing a readable UTC timestamp and an injected random identifier. | RF-27 |
| ZIP archive writer | Write sorted relative directories and DEFLATE-compressed regular files to an owned temporary archive, including valid empty ZIP output. | RF-18, RF-19, RF-21 through RF-24 |
| ZIP archive verifier | Read every ZIP entry, reject duplicate or unsafe names, verify ZIP checksums, rebuild the logical inventory, and compare archive and whole-file digests. | RF-17 through RF-19, RF-23 through RF-26 |
| Backup publisher | Coordinate temporary creation, close, archive verification, source reinspection, same-directory publication, and atomic confirmation-state commit. | RF-09, RF-16, RF-18, RF-23 through RF-27 |
| Retention planner | Select expired versions only by confirmation order, always retain the newest configured count, and never select an unconfirmed or foreign entry. | RF-28 through RF-30 |
| Retention executor | Record pending removal before deletion, complete or resume deletion transactions, report failures, and block later publication while over the configured limit. | RF-17, RF-24, RF-29, RF-30 |
| Backup cycle coordinator | Execute validation, recovery, verification, archive creation, confirmation, retention, rollback, error propagation, and lifecycle logging in the required order. | RF-07 through RF-34 |
| Watch scheduler integration | Reuse immediate first execution, completion-based waiting, startup policy snapshots, and non-overlap within one watch process. | RF-22, RF-31, RF-32 |

## Data Model

### Persisted Configuration Variants

Schema version 1 remains the existing exact object:

- `schemaVersion`: integer `1`.
- `synchronizationIntervalSeconds`: integer from `1` through `86,400`.
- `synchronizationMode`: `bidirectional` or `unidirectional`.

Schema version 2 has two strict variants:

- Common properties: `schemaVersion` fixed at `2`, `synchronizationIntervalSeconds` from `1` through `86,400`, and `synchronizationMode`.
- Non-backup variant: mode `bidirectional` or `unidirectional`, with no `backupRetentionCount` property.
- Backup variant: mode `backup`, with `backupRetentionCount` from `1` through `100` inclusive.

No additional properties are accepted in any variant. RF coverage: RF-01 through RF-06.

### Interactive Configuration Draft

- Candidate synchronization interval.
- Candidate synchronization mode.
- Optional candidate retention count present only for BACKUP.
- Validation state for each requested answer.
- Explicit summary-confirmation state.
- Cancellation state caused by rejection, absent input, interruption, or input error.

The draft cannot become active until all properties required by its selected mode are valid and explicitly confirmed. RF coverage: RF-01 through RF-05.

### Backup Policy Snapshot

- Canonical source path from the first watch root.
- Canonical destination path from the second watch root.
- Synchronization interval loaded at watch startup.
- Retention count loaded at watch startup.
- Configuration schema and mode that produced the policy.

The snapshot is immutable for one watch execution. RF coverage: RF-07, RF-10, RF-14, RF-31, RF-32.

### Logical Source Inventory

- Relative path represented in canonical ZIP path form.
- Entry type: regular file or directory.
- SHA-256 content digest for regular files.
- No timestamp, permission, owner, ACL, extended attribute, or other excluded metadata.
- Deterministic ordering and one aggregate inventory digest.

An empty source has an empty inventory and remains a valid logical state. RF coverage: RF-08, RF-18 through RF-21, RF-23, RF-25.

### Archive Identity

- Identity schema version.
- Collision-resistant archive identifier.
- Canonical source path.
- Canonical destination path.
- UTC creation time used in the visible name.
- Confirmation-order candidate.
- Aggregate logical inventory digest.

The identity is stored in ZIP-level metadata rather than as a source-tree entry, so standard extraction exposes only the backed-up hierarchy. It identifies an owned unconfirmed candidate before state commit and is cross-checked against confirmed state afterward. RF coverage: RF-14 through RF-17, RF-19, RF-26 through RF-28.

### Confirmed Archive Record

- Archive identifier and exact published filename.
- Confirmation order.
- UTC creation time.
- Whole-archive SHA-256 digest.
- Logical inventory digest.
- Lifecycle state: confirmed or pending retention removal.

The whole-archive digest detects any byte-level change after confirmation, while the inventory digest verifies logical hierarchy and contents. RF coverage: RF-17, RF-25, RF-26, RF-28 through RF-30.

### Backup Set State

- State schema version.
- Canonical destination identity used as the state key and collision check.
- Bound canonical source path.
- Next confirmation-order value.
- Ordered confirmed archive records.
- Any retention removal transaction that must be completed or rolled forward.

One strict state document is stored in GoSync's per-user application-data directory alongside, but separate from, global configuration. It is atomically replaced and never placed in the backup destination. RF coverage: RF-14 through RF-17, RF-24, RF-26, RF-28 through RF-30, RF-34.

### Destination Inspection Result

- Missing destination.
- Empty unbound destination.
- Bound destination with valid confirmed versions.
- Owned unconfirmed candidates not present in confirmed state.
- Confirmed versions pending retention removal.
- Foreign or unidentifiable entries.
- Missing, corrupt, altered, unreadable, or wrongly owned confirmed versions.
- Unsupported destination type or destination I/O failure.

Inspection is read-only and must complete before recovery, publication, or retention changes begin. RF coverage: RF-09, RF-12, RF-14 through RF-17, RF-24.

### Backup Cycle Result

- Destination-created flag for first-cycle rollback.
- Initial, archived, and final logical inventories.
- Unconfirmed archive path and identity, when created.
- Publication and confirmation outcomes.
- Retention outcome and any pending removal.
- Primary failure plus any cleanup or ownership-release failure.

The result distinguishes a confirmed backup with a later retention error from a cycle that never confirmed a version. RF coverage: RF-09, RF-16, RF-22 through RF-26, RF-29, RF-30, RF-34.

## Key Decisions

### Standard ZIP With DEFLATE

Use Go's standard ZIP support and DEFLATE for regular-file payloads; write directory entries explicitly and sort every entry by canonical relative path.

Justification: standard ZIP is interoperable, DEFLATE meets the compression requirement without a third-party dependency, explicit directories preserve empty folders, and deterministic ordering simplifies exact verification.

Discarded alternative: introduce a custom archive format or external compression library. It would reduce interoperability, add dependency and format risk, and provide capabilities outside the specification.

RF coverage: RF-18, RF-19, RF-25.

### Conditional Version-2 Configuration Schema

Decode schema version before selecting one exact property set. Persist retention only for BACKUP, while continuing to accept unchanged version-1 documents for existing modes.

Justification: the conditional schema follows the specification exactly and prevents an inactive backup policy from appearing in unrelated configurations.

Discarded alternative: require `backupRetentionCount` in every version-2 document. It would store and validate a setting that `bidirectional` and `unidirectional` never use.

RF coverage: RF-01 through RF-06.

### Immutable Startup Policy

Extend the existing watch snapshot with optional backup retention and route all cycles through the same immutable source, destination, interval, and retention policy.

Justification: one snapshot preserves the existing predictable watch behavior and prevents a live configuration edit from changing retention during a running process.

Discarded alternative: reload configuration before each cycle. A mode or retention change could alter ownership and deletion decisions within one execution.

RF coverage: RF-07, RF-31, RF-32.

### Canonical Destination-Keyed State Outside the Backup Root

Store one strict backup set state document in application data, keyed by a SHA-256 digest of the canonical destination and cross-checked with the full destination path inside the document.

Justification: external state can detect deleted or renamed confirmed ZIPs, preserve source binding, and support atomic confirmation without adding persistent control files to the user-visible archive directory. Keeping it beside configuration also makes the existing configuration-root guard protect operational state from backup inclusion.

Discarded alternative: infer ownership and history only from filenames and current ZIPs. Deletion of the newest archive or reuse by another source could not be detected reliably.

RF coverage: RF-10, RF-14 through RF-17, RF-26, RF-28 through RF-30.

### ZIP-Level Identity Plus External Confirmed State

Put versioned source, destination, archive, order, time, and inventory identity in ZIP-level metadata and record whole-archive integrity only when confirmation commits.

Justification: an interrupted candidate remains recognizable without exposing metadata as restored source content, while external confirmed state provides an independent record capable of detecting later alteration or absence.

Discarded alternative: place a manifest file inside the archived source hierarchy. Standard extraction would create a file that never existed in the source and violate exact logical representation.

RF coverage: RF-14 through RF-19, RF-25 through RF-28.

### Native Advisory Ownership Lock

Hold one operating-system advisory lock in application data, keyed by canonical destination, for the entire BACKUP watch execution. Isolate platform-specific lock calls behind build-tagged adapters and use only standard-library system-call facilities.

Justification: an OS lock rejects a live competing process and is released automatically when a process exits, avoiding permanent stale ownership after a crash.

Discarded alternative: rely only on exclusive creation of a lock file. A crash would leave a stale file with no portable, race-free way to distinguish it from a live owner.

RF coverage: RF-13, RF-22, RF-24, RF-33, RF-34.

### Three-Way Logical Stability Check

Build an initial source inventory, derive an inventory from bytes written and then read back from the ZIP, and rebuild the source inventory after archive verification. Require all three logical inventories to match exactly.

Justification: comparing names, types, hierarchy, and content digests before, during, and after creation prevents a mixed-state archive while ignoring metadata explicitly outside the contract.

Discarded alternative: compare only sizes and modification times. Filesystems can preserve or coarsen timestamps, and content can change without a reliable metadata difference.

RF coverage: RF-18 through RF-25.

### Same-Directory Temporary Publication

Create an identifiable temporary archive in the destination, close and verify it completely, verify source stability, then rename it to its unique final name in the same directory before committing confirmed state.

Justification: users never see an incomplete file under a confirmed name, same-directory rename provides the strongest available atomic publication boundary, and a crash before state commit leaves a recognizable unconfirmed candidate.

Discarded alternative: write directly to the final `.zip` path. Readers and later cycles could mistake a partially written archive for a confirmed version.

RF coverage: RF-09, RF-16, RF-18, RF-23 through RF-27.

### Confirmation Requires an Atomic State Commit

Treat publication and confirmation as distinct stages. A ZIP becomes confirmed only when the atomically replaced backup set state records its archive digest, inventory digest, ownership, and order; a published ZIP absent from that state remains unconfirmed.

Justification: one durable commit point makes interruption outcomes classifiable and allows the next cycle to remove a complete but unconfirmed ZIP safely.

Discarded alternative: equate a successful rename with confirmation. A crash before source binding or order persistence would leave an archive whose ownership and retention status were ambiguous.

RF coverage: RF-14, RF-16, RF-24 through RF-28.

### Validate All Confirmed Versions Before Mutation

Before cleanup, publication, or retention, compare every confirmed record with its file, ZIP identity, complete archive digest, ZIP readability, and logical inventory digest.

Justification: the specification requires any missing, changed, corrupt, unreadable, or wrongly owned confirmed version to block destination changes.

Discarded alternative: validate only the newest version or only archives selected for retention. Damage to another retained version could remain hidden while later mutations change the evidence available to the user.

RF coverage: RF-15 through RF-17, RF-24, RF-25.

### Owned Candidate Recovery Is State-Based

Classify a destination ZIP absent from confirmed state as an owned unconfirmed candidate only when its strict ZIP identity matches the current canonical source and destination. Validate confirmed versions first, then remove owned candidates; reject every other extra entry.

Justification: interrupted work can be recovered automatically without broad filename patterns that might delete user files.

Discarded alternative: delete every temporary-looking or unrecorded ZIP. A user-managed archive could accidentally match the pattern and be destroyed.

RF coverage: RF-15 through RF-17, RF-24, RF-26.

### Transactional Retention Removal

After confirmation, select expired records by confirmation order. Atomically mark each selected record pending removal before deleting its ZIP, then atomically remove the record after deletion; resume any pending transaction before a later publication.

Justification: the state distinguishes external deletion from an interrupted GoSync deletion and can safely roll retention forward after a crash. A failed deletion leaves the new version confirmed and blocks growth until cleanup succeeds.

Discarded alternative: delete the archive and update state afterward without a pending marker. A crash between those actions would make an intentional deletion indistinguishable from corruption.

RF coverage: RF-17, RF-24, RF-28 through RF-30.

### Confirmation Order Is Independent From Time

Assign a monotonically increasing order from the exclusively owned backup set state and use it for all retention decisions. Use UTC only for display in the filename and a cryptographically random identifier for uniqueness.

Justification: wall clocks and filesystem timestamps can repeat or move backward, while the locked state provides an unambiguous sequence.

Discarded alternative: sort filenames or modification times. Clock corrections, timezone changes, copied files, and timestamp collisions could delete the wrong version.

RF coverage: RF-27 through RF-30.

### Blocked Entries Fail the Cycle Without Immediate Retry

Classify platform lock and sharing violations as blocked-entry failures, clean unconfirmed work where possible, and let the completion-based scheduler try again after the configured interval.

Justification: BACKUP explicitly supersedes indefinite locked-file retry so one file cannot hold the whole watch process inside a cycle forever.

Discarded alternative: reuse the synchronization executor's indefinite retry loop. The cycle might never finish, preventing scheduling and a coherent stable-source check.

RF coverage: RF-22, RF-23, RF-31, RF-34.

### Roll Back a Destination Created by a Failed First Cycle

Track whether the coordinator created the destination. If no version becomes confirmed, remove owned candidates and then remove the directory only when empty; report every failed cleanup operation separately.

Justification: the failed first cycle should restore the pre-cycle state whenever the filesystem permits without hiding cleanup failures.

Discarded alternative: always leave an empty destination. That would contradict the required rollback and make a failed operation observable as successful setup.

RF coverage: RF-08, RF-09, RF-16, RF-24.

### Reuse Scheduler and Logging Contracts

Run backup through the existing immediate, completion-based watch loop and shared logging coordinator, adding backup-specific event identifiers and applying console-only mode for log overlap.

Justification: reuse preserves non-overlap, startup snapshots, protected-data handling, and destination-failure behavior consistently across all modes.

Discarded alternative: create a backup-specific timer and logger. Duplicate orchestration could diverge on intervals, overlap, sanitization, or fatal logging failures.

RF coverage: RF-11, RF-31 through RF-34.

## Execution Sequences

### Configure BACKUP

1. Parse `gosync configure`, acquire configuration ownership, and inspect any existing configuration under the existing configuration-system contract. RF-01, RF-05, RF-06.
2. Request and validate the interval and synchronization mode. RF-01.
3. If BACKUP is selected, request retention, apply 3 to an empty answer, and repeat values outside 1 through 100. RF-02, RF-03.
4. Display the mode-dependent complete summary and require explicit confirmation. RF-04.
5. Encode the exact version-2 variant and atomically replace configuration; preserve a valid previous document on every failure. RF-05, RF-06, RF-34.

### BACKUP Watch Startup

1. Validate `watch` arguments before configuration or filesystem mutation. RF-07, RF-34.
2. Resolve canonical roots and reject root equality, nesting, configuration overlap, invalid source, and unsupported existing destination types. RF-08, RF-10, RF-12.
3. Initialize logging; use console-only mode if persistent logging overlaps either root. RF-11, RF-33, RF-34.
4. Load one strict configuration snapshot and route `backup` to a backup policy containing source, destination, interval, and retention. RF-05 through RF-07, RF-32.
5. Acquire the canonical destination ownership lock for the lifetime of the process; reject a competing owner without destination changes. RF-13, RF-22, RF-33.
6. Run one cycle immediately, wait the complete configured interval after it finishes, and repeat without overlap. RF-31, RF-32.
7. Release destination ownership on normal termination and report any release error. RF-24, RF-34.

### Backup Cycle

1. Log cycle start and verify the immutable source and root relationship before destination mutation. RF-08, RF-10, RF-20, RF-21, RF-33.
2. Build the initial logical source inventory; reject unsupported, unreadable, or blocked entries. RF-08, RF-18 through RF-23.
3. Load strict backup set state and inspect the existing destination without changing it. RF-12, RF-14, RF-15, RF-24.
4. Verify source binding and classify every destination entry from state and ZIP identity. RF-14, RF-15, RF-26.
5. Validate every confirmed archive and reject any missing, altered, unreadable, corrupt, or wrongly owned version before mutation. RF-17, RF-24, RF-25.
6. Resume pending retention removals; if compliance cannot be restored, report failure and publish nothing new. RF-29, RF-30.
7. Remove owned unconfirmed candidates; reject foreign entries and stop if recovery cleanup fails. RF-15, RF-16, RF-24, RF-26.
8. Create the destination only when this is the first valid cycle and the destination is missing. RF-09.
9. Allocate confirmation order, UTC name components, and a collision-resistant archive identifier under exclusive ownership. RF-13, RF-27, RF-28.
10. Write and close one temporary ZIP from the initial inventory, including explicit empty directories or an empty archive. RF-18, RF-19, RF-21, RF-24.
11. Read the ZIP completely, verify its identity and logical inventory, and calculate its whole-archive digest. RF-17, RF-23 through RF-26.
12. Rebuild the source inventory and require initial, archived, and final logical states to match. RF-20, RF-21, RF-23, RF-25.
13. Rename the verified candidate to its unique final `.zip` name and atomically commit the confirmed archive record and source binding. RF-14, RF-24, RF-26 through RF-28.
14. Apply retention transactionally in confirmation order; retain the new version and report any cleanup failure. RF-28 through RF-30.
15. Log confirmation, retention, and final cycle outcome without file contents. RF-33, RF-34.
16. If the cycle fails before first confirmation and created the destination, remove owned work and the empty destination, reporting every rollback failure. RF-09, RF-16, RF-24.

## Test Strategy

### Configuration Unit Tests

- Mode prompting presents BACKUP as available and preserves the two existing choices. RF-01.
- Retention accepts 1, 3, and 100; an empty answer selects 3; zero, negatives, fractions, non-numeric input, 101, and unrepresentable integers repeat the question. RF-02, RF-03.
- Summary and confirmation include retention only for BACKUP and persist nothing after rejection, EOF, or interruption. RF-04.
- Strict codec tables accept exact version-1 documents, exact version-2 non-backup documents without retention, and exact version-2 backup documents with retention. RF-05, RF-06.
- Codec rejection tables cover malformed JSON, trailing values, duplicate, missing, unknown, mis-capitalized, incorrectly typed, mode-incompatible, out-of-range, and unsupported-version properties. RF-05, RF-06.
- Policy selection routes each mode correctly and retains one immutable interval and retention snapshot. RF-07, RF-32.

### Backup Domain Unit Tests

- Canonical root validation covers equal paths, nesting in both directions, aliases, missing and unreadable sources, configuration overlap, and every existing destination type. RF-08, RF-10, RF-12, RF-14.
- Logical inventory tests use canonical relative paths, deterministic ordering, directories, empty directories, empty files, Unicode names, and content digests while ignoring excluded metadata. RF-18, RF-19, RF-23, RF-25.
- Unsupported-entry tables reject symbolic links, sockets, devices, and entries introduced between initial and final inspection. RF-21.
- Archive identity codec accepts only its exact schema and rejects malformed, duplicate, missing, foreign-source, foreign-destination, and unsupported identities. RF-14 through RF-17, RF-26.
- Name generation uses UTC, remains human-readable, ends in `.zip`, and remains unique when clocks repeat. RF-27.
- Destination classification distinguishes confirmed, pending-removal, owned-unconfirmed, foreign, missing, corrupt, and wrongly owned entries without mutation. RF-14 through RF-17.
- Confirmation ordering remains deterministic across repeated or decreasing clocks and filesystem timestamp changes. RF-28.
- Retention planning covers counts 1 and 100, exact-boundary sets, multiple expired versions, pending removals, and protection of the newest confirmed version. RF-28 through RF-30.
- Cycle result tests preserve a primary failure and independently report close, cleanup, rollback, state, and ownership-release failures. RF-09, RF-16, RF-22, RF-24, RF-34.

### ZIP Unit Tests

- ZIP writing emits DEFLATE-compressed regular files, explicit empty directories, zero-length files, sorted safe relative names, and one valid empty archive for an empty source. RF-18, RF-19.
- ZIP verification reads every file to exercise CRC checks and rejects duplicates, absolute paths, parent traversal, malformed identity, unsupported entry types, and mismatched inventories. RF-17 through RF-19, RF-21, RF-25.
- Three-way inventory comparisons accept metadata-only changes and reject additions, deletions, type changes, renames, and content changes at each injected phase. RF-19, RF-23, RF-25.
- Whole-archive digest comparison detects changes to payload, headers, ZIP identity, entry order, and trailing bytes after confirmation. RF-17.

### Filesystem Integration Tests

- A missing destination is created and receives one confirmed ZIP; failure before confirmation removes the created directory. RF-09, RF-18, RF-24, RF-26.
- Failure to remove a newly created destination reports both the primary and cleanup failures without confirming a version. RF-09, RF-24.
- Existing files, links, special entries, and non-directory destinations are rejected without modification. RF-12, RF-15, RF-21.
- A first confirmed archive binds the canonical source; aliases resolving to the same path are accepted and a genuinely different path is rejected before mutation. RF-14.
- An interrupted temporary or published-but-unconfirmed ZIP is recognized, reported, and removed; a similar foreign ZIP is preserved and blocks the cycle. RF-15, RF-16, RF-26.
- Injected failures for inspect, open, read, write, close, rename, state replacement, confirmation, delete, cleanup, and directory rollback propagate and preserve the required prior state. RF-08, RF-09, RF-16, RF-22, RF-24, RF-26, RF-30, RF-34.
- Corrupt, altered, removed, renamed, unreadable, and wrong-source confirmed archives block all destination mutation. RF-17.
- Source changes during write or verification discard the candidate, preserve confirmed versions, and perform no retention deletion; metadata-only changes do not fail. RF-20, RF-21, RF-23 through RF-25.
- Empty and non-empty source archives extract to the exact required hierarchy and file contents without extra manifest entries. RF-18, RF-19, RF-25.
- Retention keeps the configured newest confirmation orders despite clock changes or modified filesystem timestamps. RF-27 through RF-29.
- A failed expired-version deletion keeps the new archive confirmed and blocks later publication until cleanup succeeds. RF-30.
- Interruption at every pending-removal transition resumes safely and never misclassifies an intentional deletion as corruption. RF-17, RF-24, RF-29, RF-30.

### Process-Concurrency Tests

- Two real processes targeting the same canonical destination demonstrate that the first owner continues and the second exits without creating, verifying, or removing destination entries. RF-13.
- Process termination releases native ownership so a later process can acquire the same destination without manual stale-lock cleanup. RF-13, RF-24.
- Path aliases map to the same ownership key and cannot bypass exclusivity. RF-13, RF-14.
- Platform-specific lock and sharing violations are classified as one-cycle blocked errors and are retried only after the fake scheduler advances. RF-22, RF-31.

### CLI and Watch Integration Tests

- A full configure session selects BACKUP, repeats invalid retention answers, displays the mode-dependent summary, confirms, and persists the exact version-2 backup document. RF-01 through RF-06.
- Existing valid version-1 configurations continue running bidirectional or unidirectional behavior unchanged. RF-06, RF-34.
- Version-2 non-backup modes preserve existing synchronization behavior and reject a retention property. RF-05, RF-06, RF-34.
- BACKUP uses the first argument only as source and the second only as destination; destination differences never modify source bytes. RF-07, RF-20.
- Persistent log overlap warns and uses console-only mode; loss of both logging destinations stops backup under the existing log contract. RF-11, RF-33, RF-34.
- Watch runs a backup immediately, waits only after complete success or failure, never overlaps cycles, and uses the startup interval and retention after configuration changes. RF-22, RF-31, RF-32.
- Lifecycle output covers start, verification, confirmation, recovery, retention, concurrency rejection, success, and failure without file contents. RF-33.

### Test Isolation

- Override application-data roots so configuration, backup state, ownership locks, and logs never touch user data. RF-05, RF-06, RF-10, RF-11, RF-13 through RF-17.
- Use temporary source and destination roots and assert source structure and bytes remain unchanged after every success and failure. RF-07 through RF-10, RF-20 through RF-24.
- Inject clocks, random identifiers, filesystem boundaries, state replacement, locks, logging, and waits so collision, interruption, blocked-file, and cleanup cases are deterministic. RF-09, RF-13, RF-16, RF-22 through RF-34.
- Use bounded fake waits rather than real synchronization intervals. RF-22, RF-31, RF-32.
- Keep platform lock contract tests shared and isolate only native acquisition and release fixtures behind platform-specific test files. RF-13, RF-22, RF-24.

### Verification

Run `go test ./...` successfully before integration, satisfying the constitution and exercising RF-01 through RF-34.

## RF Traceability

| Requirement | Planned Modules | Planned Test Areas |
| --- | --- | --- |
| RF-01 | Configuration domain, interactive configurator, policy selector | Mode prompt and CLI configuration tests |
| RF-02 | Configuration domain, interactive configurator | Retention default and boundary tests |
| RF-03 | Interactive configurator | Invalid and unrepresentable retention tests |
| RF-04 | Interactive configurator | Summary, confirmation, cancellation, and EOF tests |
| RF-05 | Configuration domain, strict configuration codec | Exact conditional schema and persistence tests |
| RF-06 | Strict configuration codec, policy selector | Version compatibility and strict rejection tables |
| RF-07 | Policy selector, backup path validator, cycle coordinator | Root-role and source-immutability integration tests |
| RF-08 | Backup path validator, source inventory builder, cycle coordinator | Missing, unreadable, and failed-inspection source tests |
| RF-09 | Backup publisher, cycle coordinator | Missing destination and first-cycle rollback tests |
| RF-10 | Backup path validator | Equal, nested, alias, and configuration-overlap tests |
| RF-11 | Backup logging policy integration | Console-only overlap and logging-destination tests |
| RF-12 | Backup path validator, destination inspector | File, link, special, and inaccessible destination tests |
| RF-13 | Destination ownership guard | Real-process exclusion, alias, release, and lock-failure tests |
| RF-14 | Path validator, state store, destination inspector, archive identity codec | First binding, same-source alias, and different-source tests |
| RF-15 | Destination inspector, state store, archive identity codec | Confirmed, unconfirmed, pending, and foreign classification tests |
| RF-16 | Destination inspector, backup publisher, cycle coordinator | Interrupted-candidate cleanup and cleanup-failure tests |
| RF-17 | Destination inspector, archive verifier, state store, retention executor | Missing, corrupt, altered, unreadable, and wrong-owner tests |
| RF-18 | Source inventory builder, ZIP writer, backup publisher | Full hierarchy, empty directory, empty file, and empty source tests |
| RF-19 | Source inventory builder, ZIP writer, ZIP verifier | DEFLATE, exact extraction, and metadata-exclusion tests |
| RF-20 | Backup cycle coordinator | Byte-for-byte source immutability tests |
| RF-21 | Source inventory builder, ZIP writer, cycle coordinator | Unsupported initial and race-introduced entry tests |
| RF-22 | Ownership guard, cycle coordinator, watch scheduler | Platform blocked-entry and next-interval tests |
| RF-23 | Source inventory builder, ZIP verifier, backup publisher | Three-way logical stability and metadata-only change tests |
| RF-24 | State store, publisher, retention executor, cycle coordinator | Full I/O, cleanup, interruption, and compound-error tests |
| RF-25 | Source inventory builder, ZIP verifier, backup publisher | Readability, CRC, logical inventory, and content tests |
| RF-26 | Archive identity codec, state store, publisher, destination inspector | Commit-boundary and every interruption-point tests |
| RF-27 | Backup name generator, publisher | UTC format, repeated clock, identifier, and collision tests |
| RF-28 | State store, retention planner | Monotonic order and clock-independent retention tests |
| RF-29 | Retention planner, retention executor | Boundary, oldest-selection, and post-confirmation tests |
| RF-30 | State store, retention executor, cycle coordinator | Failed deletion, publication blocking, and resumed cleanup tests |
| RF-31 | Watch scheduler integration, cycle coordinator | Immediate, completion-wait, and non-overlap tests |
| RF-32 | Backup policy snapshot, watch scheduler integration | Startup snapshot and next-process configuration tests |
| RF-33 | Backup logging integration, cycle coordinator | Complete lifecycle event and protected-data tests |
| RF-34 | All integration modules | Regression and exhaustive error-propagation tests |
