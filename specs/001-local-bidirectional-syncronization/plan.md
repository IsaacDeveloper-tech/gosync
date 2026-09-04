# Implementation Plan

## Scope

Implement local bidirectional directory synchronization only. The implementation shall satisfy RF-01 through RF-15 and exclude remote transports, compression, GUIs, web services, special-entry replication, and version restoration.

## Modules

| Module | Responsibility | RF Coverage |
| --- | --- | --- |
| CLI and interaction | Read selected roots, show errors and notifications, and request conflict decisions. | RF-04, RF-08, RF-09, RF-13, RF-14 |
| Root validation | Normalize roots, create missing roots, and reject equal or nested roots. | RF-10, RF-11 |
| Preflight scanner | Traverse both roots and reject symbolic links and special entries before mutations. | RF-12 |
| State store | Load and save the last confirmed synchronization record only after full success. | RF-01, RF-02, RF-03 |
| Directory comparer | Compare structures, names, types, contents, and modification times where conflict resolution needs them. | RF-03, RF-07, RF-08, RF-15 |
| Synchronization planner | Classify additions, deletions, content conflicts, and file-directory conflicts before applying changes. | RF-05, RF-06, RF-07, RF-08, RF-09 |
| Operation executor | Apply the approved synchronization plan and preserve confirmed state after failures. | RF-02, RF-05, RF-06, RF-13 |
| Locked-file retry worker | Notify, retry locked files until available, and notify on completion. | RF-14 |

## Data Model

### Confirmed Synchronization Record

- Record version.
- Canonical absolute paths for both roots.
- A list of synchronized relative paths.
- Entry type for each path: file or directory.
- Completion timestamp.

The record establishes whether an item previously existed on both sides. It is sufficient to distinguish a new item from a replicated deletion.

### Runtime Comparison Data

- Relative path.
- Entry type.
- File content digest.
- File modification time.
- Presence in the left root, right root, and confirmed record.
- Planned action or required user decision.

### Storage Decision

Store the confirmed record in the operating system's per-user application-data location, keyed by the two canonical root paths.

Justification: synchronization metadata must survive restarts without becoming user content.

Discarded alternative: storing the record inside either synchronized root. It could be copied, deleted, or treated as ordinary user data.

## Key Decisions

### Confirmed State Is Written Only After Full Success

The executor saves a new record only after every planned action, including locked-file retries, completes.

Justification: a partial operation must never become the baseline for deletion detection.

Discarded alternative: update the record after every file operation. A crash could record a state that never existed in both roots.

### Recovery Requires User Authority

When confirmed state is unavailable after interruption, compare both roots. If they differ, ask which root must overwrite the other.

Justification: without history, the program cannot safely infer whether a difference is an addition or a deletion.

Discarded alternative: automatically choose the newest files. It can incorrectly delete or overwrite legitimate content.

### Preflight Before Mutations

Scan both existing roots for unsupported entries before creating, copying, or deleting synchronization content.

Justification: RF-12 requires unsupported entries to cause no directory changes.

Discarded alternative: fail when an unsupported entry is encountered during execution. Earlier operations could already have changed content.

### Content Defines Equality

Directories are synchronized when their relative paths, entry types, and file contents match. Permissions and modification times do not need to match.

Justification: this directly implements RF-15 and avoids unnecessary metadata-only changes.

Discarded alternative: compare only file names, sizes, and modification times. Different files can share those properties.

### Conflict Resolution Uses Modification Time

For two different file contents, use the most recent modification time. If times are equal, request user selection.

Justification: this follows RF-07 and avoids inventing an arbitrary winner for RF-08.

Discarded alternative: always prefer one fixed root on ties. This can silently discard valid changes.

### File-Directory Conflicts Require Explicit Choice

Before replacing a directory with a file or vice versa, display the directory contents and request user confirmation.

Justification: replacing a directory can remove several files.

Discarded alternative: automatically prefer the newest item. A directory does not provide a single meaningful modification time for its contents.

### Locked Files Are Retried

Treat a recognized locked-file error as pending work: notify the user, retry automatically, and notify again after successful synchronization. Other filesystem errors stop the operation.

Justification: this satisfies RF-13 and RF-14 while keeping the confirmed record trustworthy.

Discarded alternative: treat a locked file as a normal fatal error. The user would need to restart synchronization manually.

## Execution Sequence

1. Read and normalize the selected root paths.
2. Reject identical or nested roots.
3. Scan existing roots for unsupported entries.
4. Create a missing root only after preflight succeeds.
5. Load the confirmed record.
6. If recovery is required, compare roots and request authoritative-root selection when they differ.
7. Compare both roots and construct a complete synchronization plan.
8. Request decisions for tied file conflicts and file-directory conflicts.
9. Execute approved additions, updates, and deletions.
10. Retry locked files until they succeed; stop for other filesystem errors.
11. Verify synchronized contents.
12. Save the new confirmed record only after complete success.

## Test Strategy

### Unit Tests

- Confirmed-record classification distinguishes additions from deletions. RF-01, RF-05, RF-06.
- Failed execution leaves the prior confirmed record unchanged. RF-02, RF-13.
- Missing-state recovery detects equal and different roots. RF-03, RF-04.
- Content comparison ignores permissions and modification-time-only differences. RF-15.
- Newer modification time wins; equal times require a decision. RF-07, RF-08.
- File-directory conflicts produce a decision request and directory listing. RF-09.
- Equal and nested roots are rejected. RF-11.

### Filesystem Integration Tests

- Missing root creation and initial replication. RF-10.
- Bidirectional file and directory additions. RF-05.
- Replicated file and directory deletions after confirmed synchronization. RF-06.
- Unsupported symbolic links and special entries cause no changes. RF-12.
- Read, write, deletion, and permission failures preserve confirmed state. RF-13.
- Locked-file retry notifications and eventual completion. RF-14.
- A successful run leaves equal structures, names, and contents, then persists state. RF-01, RF-15.

### Verification

Run `go test` successfully before integration, as required by the constitution.
