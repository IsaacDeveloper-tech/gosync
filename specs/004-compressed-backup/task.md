# Compressed Backup Mode Implementation Tasks

## Configuration Foundation

- [ ] **1. Extend the configuration domain to schema version 2** (20 min)
  RF: RF-01, RF-02, RF-05, RF-06
  Done when: Tests represent version-1 compatibility and both exact version-2 variants, with retention present only for BACKUP and constrained to 1 through 100.

- [ ] **2. Present BACKUP as an available mode** (15 min)
  RF: RF-01
  Done when: Interactive tests offer bidirectional, unidirectional, and BACKUP as selectable modes without the previous unavailable warning.

- [ ] **3. Validate interactive retention answers** (20 min)
  RF: RF-02, RF-03
  Done when: Tests accept 1, 3, and 100, map an empty answer to 3, and repeat zero, negative, fractional, non-numeric, 101, and unrepresentable values.

- [ ] **4. Add retention to the BACKUP draft and summary** (20 min)
  RF: RF-02, RF-04
  Done when: Tests include retention in a BACKUP summary only and prove rejection, EOF, or interruption persists neither mode nor retention.

- [ ] **5. Encode exact version-2 configuration variants** (20 min)
  RF: RF-05
  Done when: Tests encode exactly three properties for non-backup modes and exactly four for BACKUP, using lowercase `backup` and no additional properties.

- [ ] **6. Decode valid version-1 and version-2 configurations** (25 min)
  RF: RF-05, RF-06
  Done when: Table-driven tests load existing valid version-1 modes and both valid version-2 property sets into immutable snapshots.

- [ ] **7. Reject invalid conditional configuration schemas** (25 min)
  RF: RF-05, RF-06
  Done when: Tests reject malformed, duplicate, unknown, missing, mis-capitalized, incorrectly typed, out-of-range, mode-incompatible, and unsupported-version JSON.

- [ ] **8. Persist confirmed BACKUP configuration** (25 min)
  RF: RF-01 through RF-06, RF-34
  Done when: A service test confirms BACKUP, atomically stores its exact version-2 document, preserves the previous document on cancellation or failure, and leaves version-1 loading unchanged.

## Backup Policy And Paths

- [ ] **9. Route BACKUP from the startup configuration snapshot** (20 min)
  RF: RF-05 through RF-07, RF-32
  Done when: Policy tests route `backup` to a snapshot containing source, destination, interval, and retention without changing bidirectional or unidirectional routing.

- [ ] **10. Derive canonical backup root identities** (25 min)
  RF: RF-07, RF-10, RF-14
  Done when: Tests assign the first root as source, the second as destination, resolve path aliases, and produce stable absolute canonical identities.

- [ ] **11. Validate the source before destination access** (20 min)
  RF: RF-08, RF-20, RF-24
  Done when: Tests reject missing, non-directory, unreadable, or uninspectable sources before invoking any destination mutation.

- [ ] **12. Validate existing destination entry types** (20 min)
  RF: RF-09, RF-12, RF-24
  Done when: Tests distinguish a missing or regular-directory destination and reject files, symbolic links, special entries, and inspection failures unchanged.

- [ ] **13. Reject unsafe root and configuration overlap** (20 min)
  RF: RF-10
  Done when: Canonical-path tests reject equal roots, nesting in both directions, and configuration locations equal to or contained within either root before mutation.

- [ ] **14. Apply console-only logging for backup-root overlap** (20 min)
  RF: RF-11, RF-34
  Done when: Tests warn and disable persistent logging when its location is inside either root while retaining existing one-destination and no-destination failure behavior.

## Logical Inventory

- [ ] **15. Define deterministic logical inventory types** (15 min)
  RF: RF-18, RF-19, RF-23, RF-25
  Done when: Tests represent canonical relative paths, file and directory kinds, file-content digests, deterministic ordering, and an empty inventory without metadata fields.

- [ ] **16. Build inventories for regular files and directories** (25 min)
  RF: RF-08, RF-18, RF-19, RF-25
  Done when: Temporary-tree tests inventory nested files, empty directories, and zero-length files with exact relative paths and SHA-256 content digests.

- [ ] **17. Calculate deterministic aggregate inventory digests** (20 min)
  RF: RF-17, RF-23, RF-25
  Done when: Tests produce equal aggregate digests for equal logical trees regardless of traversal order and different digests after any name, type, hierarchy, or content change.

- [ ] **18. Reject unsupported and blocked source entries** (25 min)
  RF: RF-08, RF-21, RF-22, RF-24
  Done when: Tests identify symbolic links and available platform special entries, classify lock or sharing violations as blocked, and return explanatory errors without destination changes.

- [ ] **19. Ignore excluded metadata in logical comparisons** (15 min)
  RF: RF-19, RF-23
  Done when: Tests keep inventories equivalent after timestamp or permission-only changes while detecting every required logical change.

## Backup Set State

- [ ] **20. Define backup set and archive record states** (20 min)
  RF: RF-14, RF-17, RF-26, RF-28 through RF-30
  Done when: Tests represent destination and source identity, next confirmation order, archive and inventory digests, and confirmed or pending-removal records.

- [ ] **21. Resolve destination-keyed state and lock locations** (20 min)
  RF: RF-10, RF-13, RF-14, RF-34
  Done when: Tests derive stable collision-checked application-data paths from canonical destinations and keep state and lock files outside both backup roots.

- [ ] **22. Encode and decode backup set state strictly** (25 min)
  RF: RF-14, RF-17, RF-24, RF-26, RF-28 through RF-30
  Done when: Tests round-trip the exact state schema and reject malformed, duplicate, unknown, missing, mis-capitalized, invalid-order, and invalid-lifecycle data.

- [ ] **23. Load typed backup state outcomes** (20 min)
  RF: RF-14 through RF-17, RF-24
  Done when: Tests distinguish missing, valid, invalid, unsupported-entry, and inaccessible state without changing application data or the destination.

- [ ] **24. Replace backup set state atomically** (25 min)
  RF: RF-24, RF-26, RF-28 through RF-30
  Done when: Interruption tests expose either the previous complete state or the new complete state after candidate creation, write, close, and replacement boundaries.

- [ ] **25. Propagate backup state storage failures** (20 min)
  RF: RF-24, RF-34
  Done when: Tests return location, inspection, read, serialization, create, write, close, replace, and cleanup errors while preserving the last committed state.

## Archive Identity And Naming

- [ ] **26. Encode strict ZIP-level archive identity** (25 min)
  RF: RF-14 through RF-17, RF-26 through RF-28
  Done when: Tests round-trip one exact identity containing schema, archive ID, canonical roots, UTC time, confirmation order, and inventory digest without creating a source-tree entry.

- [ ] **27. Reject foreign or invalid archive identity** (20 min)
  RF: RF-14 through RF-17, RF-26
  Done when: Tests reject malformed, duplicate, missing, unsupported, wrong-source, wrong-destination, wrong-order, and mismatched-inventory identity values.

- [ ] **28. Generate unique UTC archive names** (20 min)
  RF: RF-27
  Done when: Injected-clock and identifier tests produce readable `.zip` names with UTC date and time and no collisions when the clock repeats.

## Destination Ownership

- [ ] **29. Define destination ownership locking contract** (15 min)
  RF: RF-13, RF-22, RF-24
  Done when: Contract tests acquire one canonical-destination owner, reject a second owner, release safely, and expose acquisition and release errors.

- [ ] **30. Add native Windows destination locking** (25 min)
  RF: RF-13, RF-22, RF-24
  Done when: Windows tests hold an OS-backed lock across handles, reject another process, and permit reacquisition after release or owner termination.

- [ ] **31. Add native non-Windows destination locking** (25 min)
  RF: RF-13, RF-22, RF-24
  Done when: Non-Windows contract tests provide the same acquisition, exclusion, automatic process-release, and error semantics behind build-tagged code.

- [ ] **32. Key ownership by canonical destination aliases** (20 min)
  RF: RF-13, RF-14
  Done when: Tests prove path aliases contend for the same lock while distinct canonical destinations can be owned independently.

## ZIP Creation And Verification

- [ ] **33. Write valid empty ZIPs and directory entries** (20 min)
  RF: RF-18, RF-19
  Done when: Tests open a valid empty-source ZIP and find explicit sorted entries for nested and empty directories in non-empty sources.

- [ ] **34. Write DEFLATE-compressed regular files** (25 min)
  RF: RF-18, RF-19, RF-21
  Done when: Tests read exact regular and zero-length file contents from safe relative entries and verify regular payloads use ZIP DEFLATE.

- [ ] **35. Propagate ZIP creation and close failures** (20 min)
  RF: RF-22, RF-24
  Done when: Injected create, header, read, copy, write, and close failures return their cause and never expose a confirmed archive.

- [ ] **36. Reject unsafe or ambiguous ZIP structures** (25 min)
  RF: RF-17, RF-19, RF-21, RF-25
  Done when: Verifier tests reject absolute paths, parent traversal, backslash ambiguity, duplicate names, unsupported entry kinds, and malformed ZIPs.

- [ ] **37. Verify ZIP checksums and logical inventory** (25 min)
  RF: RF-17 through RF-19, RF-23, RF-25
  Done when: Tests read every payload, exercise ZIP checksum validation, rebuild the exact logical inventory, and reject any hierarchy or content mismatch.

- [ ] **38. Verify whole-archive integrity digests** (20 min)
  RF: RF-17, RF-24, RF-25
  Done when: Tests detect changes to payloads, headers, ZIP identity, entry order, trailing bytes, and unreadable confirmed archives against stored SHA-256 digests.

## Destination Inspection And Recovery

- [ ] **39. Inspect missing, empty, and state-bound destinations** (20 min)
  RF: RF-09, RF-12, RF-14, RF-15
  Done when: Read-only tests return distinct missing, empty-unbound, and bound-directory outcomes without creating or deleting entries.

- [ ] **40. Enforce canonical source binding** (20 min)
  RF: RF-14, RF-24
  Done when: Tests bind the first confirmed source, accept aliases of that canonical path, and reject a different source before destination mutation.

- [ ] **41. Classify confirmed and pending-removal versions** (25 min)
  RF: RF-15, RF-17, RF-26, RF-29, RF-30
  Done when: Tests reconcile state records with exact destination filenames and distinguish confirmed records from in-progress retention transactions.

- [ ] **42. Classify owned candidates and foreign entries** (25 min)
  RF: RF-15, RF-16, RF-26
  Done when: Tests recognize only unrecorded ZIPs with matching strict identity as owned candidates and classify every other extra entry as foreign without mutation.

- [ ] **43. Reject damaged confirmed backup history** (25 min)
  RF: RF-17, RF-24, RF-25
  Done when: Tests block mutation for missing, renamed, altered, unreadable, corrupt, or wrongly owned confirmed versions and identify the affected archive.

- [ ] **44. Recover owned unconfirmed candidates** (20 min)
  RF: RF-15, RF-16, RF-24, RF-26
  Done when: Tests validate confirmed history first, report and remove every owned candidate, preserve foreign entries, and stop publication if candidate cleanup fails.

## Publication And Confirmation

- [ ] **45. Create identifiable temporary archives** (20 min)
  RF: RF-16, RF-18, RF-24, RF-26
  Done when: Tests create candidates only inside the validated destination with identity sufficient for later recovery and remove them after pre-publication failure when possible.

- [ ] **46. Enforce three-way source stability** (25 min)
  RF: RF-20, RF-21, RF-23, RF-25
  Done when: Tests require initial, archived, and final inventories to match, accept metadata-only changes, and reject additions, deletions, renames, type changes, or content changes.

- [ ] **47. Publish verified archives atomically** (20 min)
  RF: RF-24 through RF-27
  Done when: Tests rename only a closed and verified candidate to its unique final name and leave failed or interrupted publication unconfirmed.

- [ ] **48. Commit archive confirmation state** (25 min)
  RF: RF-14, RF-24, RF-26 through RF-28
  Done when: Tests classify a ZIP as confirmed only after atomic state records source binding, filename, identity, archive and inventory digests, and confirmation order.

- [ ] **49. Roll back a failed first destination** (25 min)
  RF: RF-08, RF-09, RF-16, RF-24
  Done when: Tests remove owned work and the newly created empty destination before confirmation, preserve pre-existing destinations, and report primary and rollback failures separately.

## Retention

- [ ] **50. Plan retention by confirmation order** (20 min)
  RF: RF-28, RF-29
  Done when: Tests select only the oldest confirmed records beyond retention counts 1 through 100, independent of filenames, clocks, and filesystem timestamps.

- [ ] **51. Mark expired versions before deletion** (20 min)
  RF: RF-24, RF-29, RF-30
  Done when: Tests atomically persist pending removal before deleting an expired ZIP and never select the newest retained version.

- [ ] **52. Complete and resume pending removals** (25 min)
  RF: RF-17, RF-24, RF-29, RF-30
  Done when: Interruption tests resume before later publication whether the pending ZIP still exists or its deletion completed before state finalization.

- [ ] **53. Block publication after retention failure** (20 min)
  RF: RF-29, RF-30
  Done when: Tests keep the new ZIP confirmed, report deletion failure, publish nothing further while over limit, and resume publication only after compliance is restored.

## Backup Cycle And Watch Orchestration

- [ ] **54. Order backup-cycle validation and recovery** (25 min)
  RF: RF-07 through RF-17, RF-21, RF-24
  Done when: An orchestration test runs source validation, state load, read-only destination inspection, confirmed-history verification, pending retention, and candidate cleanup before creation or publication.

- [ ] **55. Complete one successful backup cycle** (25 min)
  RF: RF-18 through RF-20, RF-23 through RF-29
  Done when: An end-to-end service test creates, verifies, publishes, confirms, and retains exactly one independent ZIP without changing source bytes.

- [ ] **56. Preserve state across backup-cycle failures** (25 min)
  RF: RF-08, RF-09, RF-16, RF-17, RF-20 through RF-24, RF-30, RF-34
  Done when: Failure-injection tests preserve confirmed history, apply no premature retention, clean candidates where possible, and return primary plus cleanup errors.

- [ ] **57. Add backup lifecycle logging events** (20 min)
  RF: RF-11, RF-33, RF-34
  Done when: Tests log cycle start, verification, confirmation, recovery, retention, ownership rejection, success, and failure without file contents or unsanitized protected data.

- [ ] **58. Hold destination ownership for the watch lifetime** (20 min)
  RF: RF-13, RF-24, RF-33
  Done when: Tests acquire before the first cycle, retain ownership during waits, reject a competing owner, and release after normal stop or failure with release errors propagated.

- [ ] **59. Connect BACKUP to completion-based scheduling** (25 min)
  RF: RF-22, RF-31, RF-32
  Done when: Fake-wait tests run immediately, wait only after each completed success or failure, never overlap cycles, and retain the startup interval and retention snapshot.

## Integration And Regression

- [ ] **60. Add configure CLI integration coverage** (25 min)
  RF: RF-01 through RF-06, RF-34
  Done when: CLI tests cover BACKUP selection, repeated invalid retention, default 3, summary, confirmation, cancellation, exact version-2 persistence, and version-1 compatibility.

- [ ] **61. Add ZIP filesystem integration coverage** (25 min)
  RF: RF-18, RF-19, RF-23, RF-25 through RF-27
  Done when: Temporary-root tests extract empty and non-empty backups to the exact source hierarchy, verify DEFLATE payloads, and find no extra source-tree manifest entry.

- [ ] **62. Add destination integrity and recovery coverage** (25 min)
  RF: RF-12, RF-14 through RF-17, RF-24, RF-26
  Done when: Filesystem tests cover binding, foreign entries, owned candidates, missing and corrupted archives, unsupported destination types, and failed recovery cleanup without unsafe deletion.

- [ ] **63. Add transactional retention integration coverage** (25 min)
  RF: RF-24, RF-28 through RF-30
  Done when: Tests cover limits 1 and 100, repeated clocks, multiple expirations, every pending-removal interruption, deletion failure, publication blocking, and resumed cleanup.

- [ ] **64. Add real-process ownership coverage** (25 min)
  RF: RF-13, RF-14, RF-22, RF-24
  Done when: Process tests reject a concurrent destination owner through path aliases and prove owner termination permits later acquisition without manual stale-lock cleanup.

- [ ] **65. Add configured watch and logging coverage** (25 min)
  RF: RF-07, RF-11, RF-20, RF-22, RF-31 through RF-34
  Done when: Integration tests prove source immutability, console-only log overlap, complete lifecycle events, immediate cycles, post-completion waits, non-overlap, and startup snapshot retention.

- [ ] **66. Add existing-mode and exhaustive error regressions** (25 min)
  RF: RF-06, RF-24, RF-34
  Done when: Tests keep version-1, bidirectional, and unidirectional behavior unchanged and cover every planned configuration, lock, state, ZIP, publication, deletion, logging, and cleanup failure boundary.

- [ ] **67. Run the complete project test suite** (10 min)
  RF: RF-01 through RF-34
  Done when: `go test ./...` passes with automated coverage for every compressed-backup requirement and defined error behavior.
