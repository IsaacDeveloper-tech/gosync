# Configuration System Implementation Tasks

- [ ] **1. Define configuration domain types and constants** (15 min)
  RF: RF-03, RF-09, RF-10, RF-18, RF-19
  Done when: Tests represent schema version 1, the inclusive interval range, both persisted modes, and immutable configuration snapshots.

- [ ] **2. Extend command parsing for configure** (20 min)
  RF: RF-01, RF-02, RF-23
  Done when: Tests accept only argument-free `gosync configure` and prove invalid `configure` or `watch` arguments invoke no configuration access.

- [ ] **3. Validate interactive interval answers** (20 min)
  RF: RF-03, RF-04
  Done when: Tests accept `1` and `86400` and reject out-of-range, fractional, empty, and non-numeric answers with another request.

- [ ] **4. Validate interactive mode answers** (20 min)
  RF: RF-05, RF-06
  Done when: Tests accept exact bidirectional and unidirectional values, label BACKUP unavailable, and request another mode after BACKUP or unknown input.

- [ ] **5. Build the interactive configuration draft** (25 min)
  RF: RF-01, RF-03 through RF-06
  Done when: Injected-stream tests collect one complete valid interval-and-mode draft after any invalid answers.

- [ ] **6. Add summary confirmation and cancellation** (25 min)
  RF: RF-07, RF-08
  Done when: Tests confirm only an accepted summary can proceed and rejection, EOF, or interruption returns cancellation without persistence.

- [ ] **7. Resolve the global config.json location** (20 min)
  RF: RF-18, RF-21, RF-26
  Done when: Tests locate `config.json` under GoSync's per-user application-data directory independently from the working directory and return location errors.

- [ ] **8. Encode the exact version-1 JSON document** (20 min)
  RF: RF-18
  Done when: Tests encode exactly `schemaVersion`, `synchronizationIntervalSeconds`, and `synchronizationMode` with their required types and values.

- [ ] **9. Decode valid configuration JSON strictly** (25 min)
  RF: RF-18, RF-19
  Done when: Tests decode only a complete version-1 object with the exact required properties, capitalization, interval, and persisted mode.

- [ ] **10. Reject ambiguous and unsupported JSON** (25 min)
  RF: RF-19
  Done when: Table-driven tests reject malformed or trailing JSON, duplicate, unknown or missing properties, wrong capitalization or types, unsupported versions or modes, and invalid intervals.

- [ ] **11. Inspect the configuration filesystem entry** (20 min)
  RF: RF-22, RF-26, RF-27
  Done when: Tests distinguish a missing path and regular file and reject symbolic links, directories, special entries, and inspection failures.

- [ ] **12. Load typed configuration outcomes** (25 min)
  RF: RF-19, RF-21, RF-22, RF-25 through RF-27
  Done when: Tests return distinct valid, missing, invalid, unsupported-entry, and I/O-failure outcomes without modifying `config.json`.

- [ ] **13. Write and close a complete candidate document** (25 min)
  RF: RF-18, RF-20, RF-21
  Done when: Temporary-directory tests produce a fully encoded and closed candidate while candidate creation, write, and close failures leave the active JSON unchanged.

- [ ] **14. Replace config.json atomically** (25 min)
  RF: RF-20, RF-21
  Done when: Tests show successful replacement exposes the complete new JSON and each simulated interruption exposes either the complete previous or complete new JSON, never partial data.

- [ ] **15. Propagate configuration storage failures** (20 min)
  RF: RF-21, RF-26
  Done when: Tests return explanatory location, read, serialization, create, write, replace, and close errors while preserving the last complete valid configuration.

- [ ] **16. Add exclusive configure ownership** (25 min)
  RF: RF-28
  Done when: Tests allow one configuration owner, reject a concurrent owner without storage changes, and release ownership after success, cancellation, or failure.

- [ ] **17. Orchestrate a successful direct configure flow** (25 min)
  RF: RF-01, RF-03 through RF-08, RF-18, RF-20, RF-21, RF-28
  Done when: A service test acquires ownership, collects and confirms valid answers, stores the exact complete JSON, and reports success.

- [ ] **18. Allow configure to repair invalid JSON** (20 min)
  RF: RF-19, RF-25
  Done when: Tests report existing invalid JSON and replace it only after a new valid interactive session is explicitly confirmed.

- [ ] **19. Preserve configuration on cancelled configure** (20 min)
  RF: RF-08, RF-20, RF-21
  Done when: Tests prove rejection, absent input, interruption, and input errors leave the previous complete JSON unchanged.

- [ ] **20. Log configuration lifecycle outcomes** (20 min)
  RF: RF-32
  Done when: Tests emit start, success, cancellation, and failure events without including answers, interval, mode, or serialized JSON.

- [ ] **21. Reject configuration paths inside synchronization roots** (20 min)
  RF: RF-31
  Done when: Canonical-path tests reject config locations equal to or contained in either root and accept paths outside both roots.

- [ ] **22. Start configure when watch has no configuration** (25 min)
  RF: RF-22, RF-23, RF-24, RF-28
  Done when: Tests start configuration only after valid watch arguments and continue toward synchronization only after confirmed persistence.

- [ ] **23. Handle invalid or inaccessible configuration at watch startup** (20 min)
  RF: RF-21, RF-25 through RF-27
  Done when: Tests stop watch with explanatory errors for invalid, unreadable, symbolic-link, directory, or special-entry configuration without changing user files.

- [ ] **24. Keep one configuration snapshot per watch process** (20 min)
  RF: RF-09, RF-10, RF-30
  Done when: Tests prove an active watch policy does not change after the stored JSON changes and a new process loads the updated complete snapshot.

- [ ] **25. Select synchronization behavior from the configured mode** (20 min)
  RF: RF-09, RF-10, RF-30
  Done when: Tests route bidirectional snapshots to existing behavior and unidirectional snapshots to source-authoritative behavior without reloading configuration.

- [ ] **26. Replace fixed ticker scheduling with completion-based waiting** (25 min)
  RF: RF-03, RF-29, RF-30
  Done when: Fake-wait tests synchronize immediately, wait the configured duration only after completion, and never overlap synchronization callbacks.

- [ ] **27. Validate unidirectional source and roots** (20 min)
  RF: RF-10, RF-11, RF-15
  Done when: Tests assign the first root as source, reject missing sources and equal or nested roots, and leave the destination unchanged after rejection.

- [ ] **28. Create a missing unidirectional destination** (20 min)
  RF: RF-12, RF-15
  Done when: Temporary-root tests create the destination only after source and preflight validation succeed.

- [ ] **29. Plan unidirectional copies and updates** (25 min)
  RF: RF-13, RF-14
  Done when: Tests plan destination copies for source-only and differing entries and never plan a destination-to-source action.

- [ ] **30. Plan deletion of destination-only entries** (20 min)
  RF: RF-13, RF-14
  Done when: Tests plan removal of files and directories absent from the source while leaving the source inventory unchanged.

- [ ] **31. Resolve unidirectional conflicts to the source** (25 min)
  RF: RF-13, RF-14, RF-16
  Done when: Tests choose source content and type for file-content and file-directory conflicts without requesting user decisions.

- [ ] **32. Reuse preflight and filesystem error protections** (25 min)
  RF: RF-11, RF-12, RF-15
  Done when: Tests reject unsupported entries before mutation and propagate non-lock filesystem errors without modifying the source.

- [ ] **33. Reuse locked-file retry behavior** (20 min)
  RF: RF-15
  Done when: Tests verify locked destination actions notify, retry until available, and complete without weakening source authority.

- [ ] **34. Verify unidirectional results and confirmed state** (25 min)
  RF: RF-13, RF-17
  Done when: Tests commit confirmed state only after exact source-destination equivalence and preserve the previous state after any failed cycle.

- [ ] **35. Connect configured policy to watch orchestration** (25 min)
  RF: RF-09 through RF-17, RF-22 through RF-24, RF-29 through RF-31
  Done when: An orchestration test runs startup validation, one configuration snapshot, selected synchronization mode, final verification, and completion-based waiting in the required order.

- [ ] **36. Add configuration filesystem integration coverage** (25 min)
  RF: RF-18 through RF-21, RF-25 through RF-28
  Done when: Temporary application-data tests verify strict loading, complete replacement, invalid-file recovery, entry-type rejection, failure preservation, and exclusive ownership.

- [ ] **37. Add configure CLI integration coverage** (25 min)
  RF: RF-01 through RF-08, RF-18, RF-20, RF-21, RF-25, RF-28, RF-32
  Done when: CLI tests cover valid configuration, repeated invalid answers, BACKUP rejection, confirmation, cancellation, repair, competing sessions, and sanitized lifecycle logs.

- [ ] **38. Add watch configuration startup integration coverage** (25 min)
  RF: RF-22 through RF-27, RF-30, RF-31
  Done when: Tests cover invalid arguments, first-run configuration, invalid or inaccessible JSON, unsupported entries, immutable startup snapshots, and root overlap.

- [ ] **39. Add unidirectional filesystem integration coverage** (25 min)
  RF: RF-10 through RF-17
  Done when: Temporary-root tests cover missing roots, exact mirroring, additions, changes, deletions, type conflicts, source preservation, retries, failures, verification, and confirmed state.

- [ ] **40. Add configured scheduler integration coverage** (20 min)
  RF: RF-03, RF-09, RF-29, RF-30
  Done when: Tests prove both modes run immediately, wait after completion using the startup interval, avoid overlap, and apply changed settings only on a new run.

- [ ] **41. Run the complete project test suite** (10 min)
  RF: RF-01 through RF-32
  Done when: `go test ./...` passes with automated coverage for every configuration-system requirement and defined error behavior.
