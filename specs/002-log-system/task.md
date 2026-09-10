# Log System Implementation Tasks

- [x] **1. Define log severity and event domain types** (15 min)
  RF: RF-01, RF-02, RF-04, RF-05
  Done when: Tests verify all four severity values and every planned stable event identifier can be represented.

- [ ] **2. Build timestamped structured log entries** (20 min)
  RF: RF-01, RF-02, RF-04, RF-05
  Done when: Tests using a fixed clock produce entries with the expected timestamp, severity, event, message, and context.

- [ ] **3. Format entries as structured text** (20 min)
  RF: RF-03, RF-05
  Done when: Tests verify one deterministic text record contains the required date and time, severity, event, and human-readable message.

- [ ] **4. Replace registered protected values** (25 min)
  RF: RF-06, RF-10
  Done when: Table-driven tests confirm registered credentials, authentication secrets, and cryptographic keys never appear in sanitized messages.

- [ ] **5. Sanitize errors, paths, and names** (25 min)
  RF: RF-06, RF-10, RF-11
  Done when: Tests preserve useful error and path context while replacing protected fragments and excluding file contents.

- [ ] **6. Add the console log destination** (20 min)
  RF: RF-03, RF-13, RF-14
  Done when: Injected-writer tests verify complete record writes and return console write failures without suppressing them.

- [ ] **7. Resolve the per-user log location** (20 min)
  RF: RF-03
  Done when: Tests verify the resolved log directory is under GoSync's operating-system application-data location and is independent of the working directory.

- [ ] **8. Detect log-location and synchronization-root overlap** (20 min)
  RF: RF-12
  Done when: Tests detect equal and contained log locations for either canonical root while accepting non-overlapping locations.

- [ ] **9. Create and append to the active log file** (25 min)
  RF: RF-03, RF-07
  Done when: Temporary-directory tests verify the active file is created, existing records are preserved, new records are appended, and I/O errors are returned.

- [ ] **10. Rotate the active file at 10 MB** (25 min)
  RF: RF-08
  Done when: Boundary tests verify rotation before a new record would take a non-empty active file beyond 10,000,000 bytes without losing that record.

- [ ] **11. Retain at most five total log files** (25 min)
  RF: RF-09
  Done when: Repeated-rotation tests retain the active file and four newest archives and remove the oldest generation.

- [ ] **12. Surface every persistent logging failure** (20 min)
  RF: RF-07, RF-08, RF-09
  Done when: Tests return creation, opening, append, rotation, and retention failures to the caller that coordinates destinations.

- [ ] **13. Coordinate healthy console and file destinations** (25 min)
  RF: RF-03, RF-05
  Done when: Tests verify the same sanitized and formatted record is written synchronously to both available destinations.

- [ ] **14. Fall back to console after persistent failure** (20 min)
  RF: RF-07
  Done when: Tests verify one persistent failure is reported through the console and prevents all further file writes during that execution.

- [ ] **15. Fall back to file after console failure** (20 min)
  RF: RF-13
  Done when: Tests verify a console write failure is recorded in the file and later records continue using the file destination.

- [ ] **16. Stop logging when both destinations fail** (20 min)
  RF: RF-14
  Done when: Tests verify simultaneous and sequential destination loss returns a logging error that stops the producing operation.

- [ ] **17. Initialize logging with root-overlap fallback** (25 min)
  RF: RF-03, RF-07, RF-12, RF-14
  Done when: Tests verify normal initialization enables both destinations, overlap enables console only with a warning, and total initialization failure returns an error.

- [ ] **18. Log synchronization lifecycle events** (25 min)
  RF: RF-01, RF-04
  Done when: Tests verify every synchronization attempt records its start and its successful or failed outcome with the planned severity.

- [ ] **19. Log synchronized change events** (20 min)
  RF: RF-02, RF-04, RF-11
  Done when: Tests verify completed copy and deletion actions produce change events with sanitized affected-path context.

- [ ] **20. Log conflict events** (20 min)
  RF: RF-02, RF-04, RF-11
  Done when: Tests verify file-content and file-directory conflicts produce conflict events without changing existing user decisions.

- [ ] **21. Log retry and warning events** (20 min)
  RF: RF-02, RF-04
  Done when: Tests verify locked-file retry start, eventual completion, and warning conditions emit their corresponding events.

- [ ] **22. Log sanitized operation failures** (25 min)
  RF: RF-02, RF-06, RF-10, RF-11
  Done when: Tests verify operation failures retain sanitized diagnostic context while the original user-facing error behavior remains unchanged.

- [ ] **23. Add dual-destination filesystem integration coverage** (25 min)
  RF: RF-03, RF-05
  Done when: A temporary application-data test confirms one synchronization event produces equivalent structured records in captured console output and the active log file.

- [ ] **24. Add rotation and retention filesystem integration coverage** (25 min)
  RF: RF-07, RF-08, RF-09
  Done when: Temporary-file tests cross the size boundary repeatedly, preserve triggering records, retain five files, and switch to console after a forced rotation failure.

- [ ] **25. Add destination fallback integration coverage** (25 min)
  RF: RF-07, RF-12, RF-13, RF-14
  Done when: Integration tests verify root overlap, file-only fallback, console-only fallback, and termination after both destinations become unavailable.

- [ ] **26. Add protected-data end-to-end coverage** (25 min)
  RF: RF-06, RF-10, RF-11
  Done when: End-to-end tests confirm protected values and file contents are absent from every console and file record while safe context remains visible.

- [ ] **27. Run the complete project test suite** (10 min)
  RF: RF-01 through RF-14
  Done when: `go test ./...` passes with automated coverage for every log-system requirement and defined error behavior.
