# FTP/SFTP Destination Implementation Plan

## Scope

Extend the existing `gosync watch <local-source> <destination>` flow with an FTP or SFTP second argument for configured `unidirectional` and `backup` modes. Keep local-to-local behavior, configuration schemas, local backup storage, and CLI-only operation unchanged. A successful remote mirror is an exact, source-authoritative replica; a remote backup is a verified ZIP in a source-bound, globally owned managed history that may coexist with unrelated entries. Unsupported server guarantees cause an explanatory failure, not weaker synchronization. This plan covers RF-01 through RF-37 of specification 005.

## Constitution Alignment

| Principle | Planned compliance | RF coverage |
| --- | --- | --- |
| Go and standard library unless a dependency is justified | Use the standard library for URL validation, FTP control/data connections, ZIP creation, hashing, terminal I/O, scheduling, and tests. The standard library has no SSH/SFTP client: justify narrowly scoped, pinned Go SSH/SFTP dependencies for authenticated SFTP and host verification rather than implementing SSH cryptography or invoking an external program. Review the dependency choice before implementation. | RF-01, RF-04 through RF-08, RF-18, RF-23, RF-28 |
| CLI without graphical interfaces or web services | Extend only the two-argument `watch` command and interactive username/password prompt; add no GUI, API server, or remote configuration service. | RF-01 through RF-06, RF-29 |
| Propagate every I/O, network, and configuration error | Distinguish input, authentication, capability, transfer, state, confirmation, retention, logging, close, and ownership-release failures; retain the primary cause alongside cleanup failures. | RF-06, RF-09 through RF-11, RF-16, RF-21 through RF-28, RF-30, RF-37 |
| Small, deterministic, English-named functions | Isolate parsing, protocol adapters, inventory, ownership, state, verification, retention, logging, and cycle orchestration; inject network sessions, trust, clocks, persistence, and waits. | RF-01 through RF-37 |
| Tests for every changed behavior and successful `go test` | Map each RF to unit or isolated FTP/SFTP, process, CLI, and failure-injection tests; run `go test` and `go test ./...` before integration. | RF-01 through RF-37 |

## Modules

These are responsibilities within the existing `src/` package and `test/` suite, not a new command or configuration schema.

| Module | Responsibility and integration boundary | RF coverage |
| --- | --- | --- |
| Command argument and endpoint parser | Reject remote sources and malformed destination URLs before configuration access; preserve local-root parsing and represent one explicit below-root server path without userinfo or path escape. | RF-01 through RF-04 |
| Remote watch policy selector | Load the existing configuration snapshot, reject remote bidirectional mode before authentication, and dispatch local paths to existing code and remote paths to mode-specific coordinators. | RF-01 through RF-03, RF-05, RF-12, RF-29, RF-31 |
| Interactive remote credentials and SSH trust | Obtain one execution-scoped username/password; reject input/output and authentication errors; check the user's established SSH host-and-port trust before SFTP credentials cross the connection. | RF-05 through RF-08, RF-30, RF-37 |
| Remote destination contract | Define typed listing, safe path resolution, binary read/write, create, deletion, publication, close, and capability outcomes independently of protocol; prohibit operations outside the selected remote directory. | RF-04, RF-09 through RF-11, RF-13, RF-18, RF-26, RF-28 |
| FTP and SFTP adapters | Translate the common destination contract to ordinary FTP and password-authenticated SFTP respectively; reject inadequate listings, entry typing, name preservation, verification, or ownership semantics instead of guessing. | RF-07, RF-08, RF-10, RF-11, RF-25, RF-26, RF-28 |
| Source and remote inventory | Reuse the existing logical local source inventory; inspect remote entries without following links, distinguish unsupported and unrelated items by mode, and compare actual file bytes rather than timestamps. | RF-09, RF-11 through RF-16, RF-21, RF-28, RF-34, RF-36 |
| Destination protection and ownership | Recognize backup-managed entries before a mirror; coordinate backup ownership across computers and URL aliases using an actual shared destination identity, refusing mutation if exclusivity or safe classification cannot be established. | RF-19 through RF-22, RF-25, RF-32, RF-33, RF-36 |
| Remote mirror planner and executor | Reuse source-authoritative action planning, but execute through the remote destination contract; validate both sides before deletion and keep the source read-only. | RF-09 through RF-17, RF-28, RF-32, RF-34 |
| Remote mirror verifier and confirmed-state store | Verify the final remote tree against a stable local source; preserve previous state on failure and reassess the actual destination on the next attempt, including across URL aliases. | RF-12 through RF-17, RF-27, RF-34, RF-36, RF-37 |
| Remote backup-set catalog | Maintain shared, durable source binding, archive integrity, confirmation order, and pending retention outcomes visible from every client; distinguish managed records, unconfirmed artifacts, and unrelated entries. | RF-18 through RF-22, RF-24, RF-25, RF-33, RF-35 |
| Archive creation, transfer, and read-back verifier | Reuse the local ZIP payload contract; associate an archive with a remote destination without changing the local archive format; verify transferred bytes and stable source contents before confirmation. | RF-18, RF-20 through RF-23, RF-27, RF-28, RF-35 |
| Backup confirmation, recovery, and retention | Commit only verified, owned versions; recover identifiable unconfirmed work without promoting it; remove only expired confirmed managed versions and preserve the newest after pruning errors. | RF-19 through RF-27, RF-33, RF-35 through RF-37 |
| Remote cycle and watch coordinator | Order preflight, authorization, inspection, verification, mutation, confirmation, cleanup, logging, and resource release; stop after remote errors but schedule another cycle for a local source change. | RF-01, RF-06, RF-09, RF-23 through RF-31, RF-34, RF-37 |
| Logging and protected-data integration | Reuse the existing logging destinations and fallback policy; sanitize URL, server replies, credentials, errors, and user-facing remote failures. | RF-05, RF-06, RF-08, RF-26, RF-30, RF-31, RF-37 |

## Data Model

### Remote Endpoint and Watch Policy

- Transport: exact `ftp` or `sftp`; server and port; absolute server-visible directory strictly below `/`.
- Validated destination identity and a safe display form without userinfo, secret, query, fragment, encoded separators, or escape segments.
- Canonical local source path; immutable configured mode, interval, and optional backup retention for the current execution.
- Resolved actual remote-directory identity for backup ownership and alias comparisons, distinct from the URL spelling; an unknown identity is a blocking outcome for ownership-sensitive operations.

The existing local `RootPaths` and configuration JSON remain local-only or unchanged; a URL is never normalized as a Windows/local path. RF coverage: RF-01 through RF-05, RF-12, RF-20, RF-25, RF-29, RF-32.

### Execution-Scoped Authentication and Trust

- Username and password reside in memory only during one remote execution; neither is placed in a URL, persistent state, log, or error message.
- SFTP trust outcome for the selected server **and port**: recognized, absent, changed, or unavailable.
- Prompt, authentication, connection, and close outcomes remain distinct so a failure can be reported without exposing a secret.

RF coverage: RF-05 through RF-08, RF-26, RF-30, RF-37.

### Remote Capability and Entry Inventory

- Capabilities: complete typed enumeration; nonredirecting directory access; lossless distinct relative names; binary content round trips; independently readable result; and, for remote backup, exclusive cross-client ownership and durable confirmation.
- Entry: relative path, type (`regular`, `directory`, `link`, `special`, `unknown`), and a content digest only after reading file bytes. Metadata excluded by existing synchronization/backup contracts is not used as proof of equality.
- Snapshot: selected-root validity, all scoped entries, unsupported paths, any managed backup footprint, and whether the inventory can be considered complete.

Unknown types or incomplete listings are not silently treated as regular files or an empty directory. A foreign BACKUP entry may remain untouched; a mirror requires complete preflight. RF coverage: RF-09 through RF-16, RF-19, RF-28, RF-32, RF-34, RF-36.

### Remote Mirror Cycle and Confirmed State

- Initial and final local logical inventories; initial and final remote inventories; ordered source-authoritative copy, replacement, and deletion actions confined to the selected directory.
- Locally retained confirmed state scoped to the canonical local source and actual remote destination, never to credentials or the URL spelling alone.
- Outcome: confirmed, local-source-changed (next interval), or remote/local-storage failure (stop); primary error and any later cleanup/release error.

Partially changed remote contents remain observable as unconfirmed work and are reassessed from the source next time. No persistent GoSync entry may remain in a successfully mirrored remote directory. RF coverage: RF-12 through RF-17, RF-26 through RF-29, RF-32, RF-34, RF-36, RF-37.

### Shared Remote Backup History

- One managed namespace in the actual remote directory, distinguishable from unrelated entries without treating every ZIP or file as GoSync-owned.
- Durable destination and canonical-source association, versioned history, unique archive identities, ordered confirmed records, whole-archive and logical-content integrity, and any pending removal transaction.
- Candidate lifecycle: identifiable but unconfirmed; verified and published but still unconfirmed; durably confirmed; or confirmed and pending retention removal. Unknown transfer/acknowledgment outcomes are classified from durable proof, not filename or readability.
- Ownership outcome: exclusively held by this execution, held by another, absent and safely claimable, stale but safely recoverable, or uncertain/unsupported; an uncertain claim cannot authorize a mutation.
- Inspection outcome separates confirmed, corrupted/missing, owned unconfirmed, conflicting managed, unrelated, and unidentifiable entries.

The shared remote history is authoritative across clients; existing per-user *local* backup-state and OS-lock formats remain authoritative only for local backups. A local cache cannot authorize remote publication, deletion, or adoption of a history from another source. RF coverage: RF-18 through RF-27, RF-32, RF-33, RF-35 through RF-37.

### Remote Backup Cycle Result

- Existing destination or destination newly created and eligible for empty first-cycle rollback.
- Initial ZIP and source inventory, transferred candidate identity, read-back verification, durable confirmation outcome, and retention outcome.
- Primary failure plus independent cleanup, connection-close, state, logging, and ownership-release failures.

A confirmed version with failed retention remains confirmed; uncertain confirmation never becomes success merely because an archive is readable. RF coverage: RF-10, RF-21 through RF-27, RF-30, RF-35, RF-37.

## Key Decisions and Rejected Alternatives

### Validate Remote Arguments Before Configuration

Decision: classify the two arguments first; reject a remote source or ambiguous remote URL before loading configuration, then reject remote bidirectional mode after loading the startup snapshot and before credential input.

Why: invalid input cannot trigger configuration changes or connect to an unintended directory. Rejected alternative: pass every argument through the existing local-path normalizer and detect a URL later; it could turn a URL into a filesystem path or prompt for unusable settings. RF coverage: RF-01 through RF-04, RF-29.

### Standard-Library FTP and Justified SFTP Dependencies

Decision: use Go's standard networking facilities for ordinary FTP, subject to complete typed-listing and binary-verification capability checks; for SFTP, evaluate and pin maintained Go SSH and SFTP libraries narrowly for host authentication and protocol operations.

Why: SSH/SFTP is absent from the standard library, while a home-grown SSH implementation would add cryptographic and protocol risk; plain FTP must retain its specified unencrypted behavior. Rejected alternative: shell out to `ssh`/`sftp` or hand-write SSH and SFTP. Shelling out makes portable password prompting and error/secret handling unreliable; hand-writing SSH is disproportionate and unsafe. RF coverage: RF-05 through RF-08, RF-26, RF-28, RF-30.

### Fail Closed on Trust, Login, and Server Capabilities

Decision: obtain one username/password interactively per execution; validate pre-existing user SSH trust for the host and port before sending SFTP credentials; prove typed listings, safe names, and byte-preserving verification before trusting remote results.

Why: an unknown SFTP server or insufficient remote observability cannot support a safe mirror or archive confirmation. Rejected alternative: accept unknown host identities, parse human-readable FTP listings as authoritative, or assume successful upload responses prove correct contents. These choices could leak secrets or misclassify or delete entries. RF coverage: RF-05 through RF-11, RF-16, RF-23, RF-28, RF-30.

### Keep Remote Destinations Behind a Protocol-Neutral Contract

Decision: separate shared inventory, planning, and lifecycle decisions from FTP/SFTP commands, with mode-specific preflight and protocol-specific capability failures. Reuse existing source inventories, ZIP content, logging, scheduler, and local configuration without routing URL paths into local file operations.

Why: a single safety contract makes both transports and old local behavior testable without duplicating deletion/retention policy. Rejected alternative: simulate a remote folder as a mounted local root for the existing executor. Filesystem atomicity, link handling, and process locks would be incorrectly assumed to exist on FTP/SFTP. RF coverage: RF-01 through RF-03, RF-09 through RF-18, RF-28 through RF-31.

### Prove Backup Absence Before an Exact Mirror

Decision: inspect the selected remote directory for confirmed/unconfirmed managed backup entries and active backup ownership before a mirror mutation; refuse uncertain classification. Avoid leaving any persistent GoSync control entry in a successful exact mirror, and do no mirror writes after releasing any protective ownership needed for final verification.

Why: source-authoritative deletion must not erase backup history, and control files would break the exact-replica contract. Rejected alternative: exempt backup-named files from mirror deletion. That would neither create an exact mirror nor protect aliases or unrecognized managed state. RF coverage: RF-11, RF-13 through RF-17, RF-25, RF-32, RF-36.

### Shared Remote History, Not Per-Computer Authority

Decision: make durable backup source binding, confirmation order, ownership, and integrity visible from the actual remote directory to every client, including through different URL aliases. Reject a history that is present but cannot be attributed or verified; keep unrelated files untouched.

Why: local per-user state cannot establish cross-computer ownership or discover a missing confirmed archive from another client. Rejected alternative: reuse only the local destination-keyed backup store and OS lock, or infer state from `.zip` filenames. Separate computers or URL spellings could then publish into the same history, erase unrelated files, or lose confirmation order. RF coverage: RF-18 through RF-22, RF-24, RF-25, RF-32, RF-33, RF-35.

### Require Reliable Cross-Computer Exclusion

Decision: accept remote BACKUP only when the server can support a shared, exclusive ownership claim and a provable release/recovery outcome for the *actual* destination; any alias must observe the same owner, and uncertainty after a crash blocks further mutation. Keep mirror/backup protection coordinated without leaving control entries in a confirmed mirror.

Why: a process-local lock, client clock, or best-effort marker cannot guarantee the specification's cross-computer exclusivity. Rejected alternative: use only a local OS lock, timestamp expiry, or a nonexclusive FTP upload as the ownership signal. Competing or stale owners could both delete or confirm versions. RF coverage: RF-20, RF-21, RF-25, RF-26, RF-32, RF-33, RF-36.

### Verify Before Durable Confirmation and Reconcile Uncertainty

Decision: reuse complete ZIP creation locally, transfer a distinguishable candidate, read it back for ZIP and byte/content verification, check source stability, and classify it as confirmed only when durable shared history proves ownership, integrity, and order. On lost acknowledgments, reread durable proof; otherwise treat the candidate as unconfirmed and recover only identifiable owned work.

Why: successful transfer, published name, or readable ZIP alone cannot prove a confirmed backup after a disconnected response. Rejected alternative: confirm on upload completion or retrospectively adopt any readable ZIP. Both could promote partial or wrongly bound history. RF coverage: RF-18, RF-20 through RF-23, RF-26 through RF-28, RF-33, RF-35.

### Retain Only Source-Bound Confirmed Versions

Decision: apply existing count-based confirmation order, integrity checks, and resumable pending-removal semantics to shared remote history; never remove unrelated or unconfirmed entries as retention candidates. A failed prune leaves the newly confirmed version valid and blocks the next publication until safe compliance returns.

Why: retention must bound managed history without erasing unrelated content or obscuring a partial deletion. Rejected alternative: sort all ZIPs by name/date or prune before confirmation. A clock change or foreign ZIP could cause data loss, and a failed upload could erase the last usable backup. RF coverage: RF-19, RF-21, RF-22, RF-24, RF-26, RF-27, RF-33, RF-35.

### Separate Local Source Instability From Remote Errors

Decision: reuse immediate, completion-based scheduling and startup snapshots; treat a changed local source as an unconfirmed cycle eligible for the next interval, while network, remote capability, authentication, state, or ownership errors terminate remote `watch`. Report cleanup, close, release, and logging errors without hiding the primary cause.

Why: remote failures leave the destination uncertain; a changing local source can be re-evaluated without silently committing a mixed state. Rejected alternative: retry remote I/O indefinitely within one cycle or swallow remote BACKUP cycle errors in the existing backup watch loop. Either would hide failures or prevent predictable scheduling. RF coverage: RF-06, RF-16, RF-23, RF-26, RF-27, RF-29 through RF-31, RF-34, RF-37.

## Execution Sequences

### Remote Watch Startup

1. Validate argument count, local source role, and remote URL before configuration access; preserve the existing local-to-local route. RF-01 through RF-04.
2. Load the existing configuration snapshot or finish the existing first-run configure flow; reject remote bidirectional mode before requesting credentials. RF-01 through RF-03, RF-29.
3. Establish logging and local-source guards, prompt for ephemeral username/password, and validate SFTP host trust before sending credentials. RF-05 through RF-09, RF-30, RF-31.
4. Inspect server capabilities and destination-root safety; for BACKUP, establish shared ownership for the execution; for mirror, prove backup protection and destination compatibility before mutation. RF-10, RF-11, RF-19 through RF-22, RF-25, RF-28, RF-32, RF-33.
5. Execute a cycle immediately, apply the configured completion-based wait where allowed, and release resources with all failures reported. RF-26, RF-29, RF-30, RF-34, RF-37.

### Unidirectional Remote Cycle

1. Inventory the existing local source and safely inspect the entire remote directory or classify it as missing; reject unsafe entries, ambiguous names, managed backup footprints, and unsupported server behavior before destructive work. RF-09 through RF-11, RF-28, RF-32.
2. Create a missing remote directory if valid; plan copies, replacements, and deletions from the local source without reading remote differences back into the source. RF-10, RF-12 through RF-15.
3. Execute remote actions, compare the final remote contents and a stable local source, and confirm state only after exact equivalence; keep the previous record and retry later only for a local source change. RF-13 through RF-17, RF-27, RF-34, RF-36.
4. On any remote or local-state error, report all causes, stop `watch`, and let the next execution reassess actual contents; leave no successful-mirror control files. RF-17, RF-26, RF-27, RF-30, RF-32, RF-37.

### Remote BACKUP Cycle

1. Under shared destination ownership, inspect the source, selected root, existing managed history, and unrelated entries without mutation; reject unidentifiable or corrupt confirmed history. RF-09 through RF-11, RF-19 through RF-21, RF-25, RF-28, RF-33.
2. Restore pending retention compliance and remove only identifiable unconfirmed managed artifacts; fail without new publication if cleanup is blocked. RF-19, RF-21, RF-22, RF-24, RF-35.
3. Create the missing destination if needed; produce the existing complete ZIP from a stable source; transfer as an identifiable unconfirmed version and read back its bytes, contents, and identity. RF-10, RF-18, RF-20, RF-23, RF-28, RF-35.
4. Durably confirm a verified archive with one unambiguous confirmation order and bound source; never infer confirmation from a lost response. RF-20, RF-23, RF-24, RF-33, RF-35.
5. Remove only expired confirmed managed versions after confirmation; record partial retention, preserve the newest version, and stop on a remote failure. RF-19, RF-21, RF-24, RF-26, RF-27, RF-36.
6. On pre-confirmation failure, classify or remove only owned incomplete work, roll back a newly created empty destination where possible, and report any additional cleanup or ownership-release failure. RF-10, RF-22, RF-23, RF-26, RF-27, RF-30, RF-37.

## Test Strategy

### Deterministic Domain and Contract Tests

- URL and command tables cover supported schemes, missing host/path, root `/`, relative and dot paths, encoded separators, userinfo, queries, fragments, equivalent URL spellings, a remote source, invalid argument counts, and validation before configuration callbacks. RF-01 through RF-04, RF-20, RF-25.
- Credential and trust tests cover cancel/EOF, output failure, invalid username/password, zero persistence, absent/changed/valid host-and-port trust, and no SFTP credential transmission before verification. RF-05 through RF-08, RF-30, RF-37.
- Inventory tests cover empty sources, empty directories, zero-length files, symlinks, special/unknown entries, parent-path redirects, incomplete listings, unreadable bytes, case-folded or rewritten names, and valid metadata-only changes. RF-09 through RF-16, RF-23, RF-28, RF-34.
- Plan/result tests cover copy/update/delete/type conflicts, source-directed resolution, exact empty mirrors, stable-source comparison, state preservation on failure, compound errors, and retention ordering independent of server clocks. RF-12 through RF-17, RF-24, RF-26, RF-27, RF-34, RF-37.
- State and identity tests distinguish foreign, confirmed, unconfirmed, corrupt, missing, conflicting, wrong-source, unknown-order, pending-removal, and newly bound histories, including alias and cross-computer views. RF-19 through RF-25, RF-32, RF-33, RF-35.

### Isolated FTP and SFTP Integration Tests

- Exercise both adapters against controlled disposable servers with real network sessions and independent control/data or SSH/SFTP exchanges; verify ordinary FTP and SFTP password login, server trust, full typed listings, binary round trips, exact paths, ZIP readability, and failed capability checks. RF-04 through RF-11, RF-18, RF-23, RF-28.
- Cover a pre-existing directory with remote-only items, a newly created destination, an empty source, blocked reads/writes/deletes, malformed or unreadable remote entries, a foreign BACKUP symlink, and first-cycle destination rollback. RF-09 through RF-16, RF-19, RF-26, RF-28, RF-32.
- Transfer complete and empty ZIPs, corrupt confirmed versions, inject a dropped acknowledgment before and after durable confirmation, and verify owned-unconfirmed recovery never promotes a ZIP from readability alone. RF-18, RF-21 through RF-24, RF-27, RF-33, RF-35.
- Inject failure at remote listing, upload, read-back, publication, shared-state persistence, expiration deletion, candidate cleanup, connection close, and ownership release; assert stop/retry classification and independent reporting of all errors. RF-06, RF-10, RF-21 through RF-27, RF-30, RF-35, RF-37.
- Change the source or remote tree during a cycle and confirm no mixed-state mirror/backup is recorded; change only excluded metadata and confirm it does not invalidate a stable logical source. RF-16, RF-23, RF-28, RF-34, RF-36.

### Ownership and Mode-Transition Tests

- Run separate clients/processes against one real controlled destination, including two hostnames or URL spellings for that location; verify one source binding, cross-client ownership rejection without verification or mutation, and refusal when identity or exclusivity cannot be proven. RF-20, RF-21, RF-25, RF-26, RF-33, RF-36.
- Interrupt an owner, then test a demonstrably recoverable claim and an uncertain/stale claim; the latter must not publish, delete, or silently adopt another source's history. RF-22, RF-25, RF-26, RF-33, RF-35.
- Switch from remote BACKUP to unidirectional on the same directory, with confirmed, unconfirmed, and concurrently owned histories; assert zero mirror mutation even through an alias. RF-19, RF-25, RF-32.
- Inject an unrelated entry that resembles a managed name or changes during a cycle; assert no foreign deletion and no unsafe confirmation. RF-19, RF-21, RF-28, RF-33, RF-36.

### CLI, Logging, Scheduling, and Regression Tests

- Run first-time configuration, invalid existing configuration, remote bidirectional rejection, credential prompting, and local-to-local watch through the CLI without modifying the existing JSON schema. RF-01 through RF-06, RF-29, RF-31.
- Verify initial execution, startup snapshot, nonoverlapping cycles, a source-change retry at the next interval, and immediate watch termination for network/remote failures; release errors remain visible. RF-23, RF-26, RF-29, RF-34, RF-37.
- Exercise console/persistent-log fallback, logging failures, sanitized server responses containing secrets, and omission of credentials from logs, state, and user-facing errors. RF-05, RF-06, RF-26, RF-30, RF-31, RF-37.
- Rerun the existing local bidirectional, unidirectional, BACKUP, configuration, and logging suites unchanged. RF-03, RF-31.

### Isolation and Verification

Keep network fixtures on loopback, sources/destinations and SSH trust fixtures in temporary test-controlled locations, and application data isolated from real user files. Inject server faults, clocks, wait functions, connection-drop points, state commits, and ownership outcomes for reproducible tests; use real processes only where cross-process ownership matters. Run `go test` and `go test ./...` before integrating any implementation. RF coverage: RF-01 through RF-37.

## RF Traceability

| RF | Primary modules | Principal tests |
| --- | --- | --- |
| RF-01 | Argument parser, policy selector, cycle coordinator | Two protocols, two modes, unchanged local route |
| RF-02 | Argument parser, policy selector | Remote source before configuration; bidirectional before auth |
| RF-03 | Policy selector, local integration | Local-mode regression suite |
| RF-04 | Endpoint parser, destination contract | URL/path rejection and no side effects |
| RF-05 | Credentials and trust, logging | Prompt, ephemeral username/password, no persistence |
| RF-06 | Credentials and trust, coordinator | EOF, prompt output, auth failure, no changes |
| RF-07 | FTP adapter | Ordinary FTP transfer and protocol boundary |
| RF-08 | SSH trust, SFTP adapter | Host/port trusted, missing, changed, no credential leak |
| RF-09 | Source inventory, coordinator | Missing/unreadable/unsupported source before mutation |
| RF-10 | Destination contract, coordinator | Missing destination, invalid path, first-cycle rollback |
| RF-11 | Destination inventory, mirror, backup inspector | Mirror full preflight; foreign link in BACKUP |
| RF-12 | Mirror and backup coordinators | Unchanged local source across success/failure |
| RF-13 | Mirror planner/executor, verifier | Adds, changes, deletes, exact equivalence |
| RF-14 | Mirror planner/executor | File-content and file-directory conflicts |
| RF-15 | Mirror planner/executor | Empty source deletes remote-only entries |
| RF-16 | Source inventory, mirror verifier, state | Stable-source final verification and failure |
| RF-17 | Mirror state, coordinator | Interrupted cycle reassessment from source |
| RF-18 | ZIP transfer and verifier, backup coordinator | Full/empty readable ZIP and retention count |
| RF-19 | Backup inspector, catalog, retention | Foreign entries, collisions, no unrelated deletion |
| RF-20 | Endpoint identity, backup catalog | Canonical source and URL-alias binding |
| RF-21 | Catalog, inspector, verifier | Missing/corrupt/wrong-owner history blocks mutation |
| RF-22 | Catalog, recovery | Candidate-only recovery and cleanup failure |
| RF-23 | ZIP verifier, coordinator | Read-back, source changes, remote failure vs retry |
| RF-24 | Catalog, retention | Confirmation order, pruning failure, growth block |
| RF-25 | Ownership, backup coordinator | Cross-process/client/alias exclusion, unsupported server |
| RF-26 | Adapters, coordinator | Remote failure and close/release stop without retry |
| RF-27 | Coordinator, state | Prior state preserved; compounded cleanup failures |
| RF-28 | Adapters, capability/inventory | Listing, typing, naming, binary and read-back failures |
| RF-29 | Policy selector, watch coordinator | Immediate cycle, completion wait, immutable snapshot |
| RF-30 | Logging integration | Lifecycle, scrubbed URL/error, console report |
| RF-31 | Local guards, logging integration | Configuration/log overlap, fallback and regressions |
| RF-32 | Destination protection, mirror coordinator | Mode transition, aliases, active owner, uncertain state |
| RF-33 | Backup catalog and inspector | Unknown source/order/integrity rejects before mutation |
| RF-34 | Source inventory, mirror coordinator | Changed logical source rechecked next interval |
| RF-35 | Confirmation and recovery | Lost acknowledgments, durable proof, no ZIP adoption |
| RF-36 | Destination inspector, verifiers | Concurrent remote changes; foreign entries preserved |
| RF-37 | State, coordinator, logging | Local-state/close errors and independent causes |
