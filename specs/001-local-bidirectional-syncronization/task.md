# Implementation Tasks

- [x] **1. Define synchronization domain types** (20 min)
  RF: RF-01, RF-02, RF-05, RF-06
  Done when: Entry, comparison, action, and result types express every required synchronization outcome.

- [x] **2. Validate selected root paths** (20 min)
  RF: RF-11
  Done when: Identical and nested roots are rejected with explanatory errors.

- [ ] **3. Scan roots before modifications** (25 min)
  RF: RF-12
  Done when: Symbolic links and special entries are detected before any content change.

- [ ] **4. Create missing root directories** (15 min)
  RF: RF-10
  Done when: A missing selected root is created only after successful preflight validation.

- [ ] **5. Implement confirmed-state storage** (25 min)
  RF: RF-01, RF-02
  Done when: State loads by root pair and is written only after an explicitly successful run.

- [ ] **6. Build directory inventories and content comparison** (25 min)
  RF: RF-03, RF-07, RF-15
  Done when: Inventories detect path, type, content, and modification-time differences while ignoring permissions.

- [ ] **7. Classify additions and deletions from confirmed state** (25 min)
  RF: RF-05, RF-06
  Done when: An item existing on one side is correctly classified as either an addition or deletion.

- [ ] **8. Implement missing-state recovery flow** (20 min)
  RF: RF-03, RF-04
  Done when: Absent state triggers comparison and requests an authoritative root only when directories differ.

- [ ] **9. Implement file-content conflict selection** (20 min)
  RF: RF-07, RF-08
  Done when: Newer content wins automatically and equal modification times require user selection.

- [ ] **10. Implement file-directory conflict selection** (20 min)
  RF: RF-09
  Done when: The directory contents are listed and no replacement occurs without user choice.

- [ ] **11. Generate a complete synchronization action plan** (25 min)
  RF: RF-05, RF-06, RF-07, RF-08, RF-09
  Done when: All required copies, deletions, and user decisions are known before execution starts.

- [ ] **12. Execute approved file and directory actions** (25 min)
  RF: RF-05, RF-06, RF-07, RF-13
  Done when: Approved additions, updates, and deletions are applied, while non-lock errors stop execution.

- [ ] **13. Add locked-file retry and notifications** (25 min)
  RF: RF-14
  Done when: Locked files notify the user, retry automatically, and notify again after synchronization succeeds.

- [ ] **14. Verify final contents and commit confirmed state** (20 min)
  RF: RF-01, RF-02, RF-15
  Done when: State is saved only after both roots have equal structure, names, and contents.

- [ ] **15. Parse the watch command and its roots** (20 min)
  RF: RF-16
  Done when: `gosync watch <directory-a> <directory-b>` accepts exactly two roots and invalid argument counts show usage information.

- [ ] **16. Add the five-second watch loop** (20 min)
  RF: RF-17
  Done when: The command synchronizes immediately and initiates another check every five seconds while running.

- [ ] **17. Connect the CLI orchestration flow** (25 min)
  RF: RF-04, RF-08, RF-09, RF-10, RF-11, RF-13, RF-14, RF-16, RF-17
  Done when: The CLI runs command parsing, validation, preflight, planning, decisions, execution, notifications, state handling, and periodic rechecks in order.

- [ ] **18. Add unit tests for state and classification** (25 min)
  RF: RF-01, RF-02, RF-03, RF-05, RF-06
  Done when: Tests cover confirmed-state persistence, interruption preservation, additions, and deletions.

- [ ] **19. Add unit tests for validation, conflicts, and watch behavior** (25 min)
  RF: RF-07, RF-08, RF-09, RF-11, RF-12, RF-15, RF-16, RF-17
  Done when: Tests cover root overlap, unsupported entries, content equality, timestamp ties, type conflicts, command arguments, immediate synchronization, and five-second rechecks.

- [ ] **20. Add filesystem integration tests** (25 min)
  RF: RF-05 through RF-17
  Done when: Temporary-directory tests verify successful synchronization, recovery, failures, locked-file completion, and watch polling.

- [ ] **21. Run the complete test suite** (10 min)
  RF: All
  Done when: `go test` passes with coverage for every functional requirement and defined error behavior.
