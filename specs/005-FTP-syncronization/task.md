# FTP/SFTP Destination Implementation Tasks

Tasks are ordered by dependency, are each estimated below 30 minutes, and remain unchecked until implemented and verified. `RF` identifiers refer to this specification; `Hecho cuando:` gives the observable completion check. Use the existing `src/` and `test/` layout, test changed behavior, and keep local-to-local behavior intact.

## Command And Endpoint Foundation

- [ ] **1. Represent local and remote watch destinations separately** (15 min)
  RF: RF-01, RF-03, RF-04
  Hecho cuando: Tests distinguish a local path from FTP and SFTP URLs without normalizing a URL as a local filesystem path.

- [ ] **2. Classify both watch arguments before configuration access** (20 min)
  RF: RF-01, RF-02, RF-04
  Hecho cuando: Parsing tests reject a remote first argument and invalid argument counts without calling the configuration store.

- [ ] **3. Validate remote scheme, host, and below-root directory** (20 min)
  RF: RF-01, RF-04
  Hecho cuando: Tests accept explicit FTP/SFTP subdirectories and reject missing hosts, missing paths, `/`, and unsupported schemes.

- [ ] **4. Reject unsafe or ambiguous remote path encodings** (25 min)
  RF: RF-04
  Hecho cuando: Tests reject relative paths, dot segments, encoded separators, escapes, and equivalent ambiguous spellings before any connection.

- [ ] **5. Reject URL credentials, queries, and fragments** (15 min)
  RF: RF-04, RF-05, RF-30
  Hecho cuando: Tests reject userinfo, passwords, queries, and fragments without logging the rejected secret or prompting for access.

- [ ] **6. Preserve local-to-local argument routing** (20 min)
  RF: RF-01, RF-03
  Hecho cuando: Existing valid local pairs still reach their original path validator and no remote authentication hook is invoked.

- [ ] **7. Reject remote bidirectional mode after configuration loads** (20 min)
  RF: RF-01, RF-02
  Hecho cuando: A configured bidirectional watch with a remote destination errors before credential prompts, connections, or mutations.

- [ ] **8. Route supported remote modes from one startup snapshot** (20 min)
  RF: RF-01, RF-03, RF-12, RF-29
  Hecho cuando: Policy tests select remote mirror or BACKUP only for their configured modes and retain the startup interval and retention.

- [ ] **9. Apply local source, configuration, and log root guards** (20 min)
  RF: RF-09, RF-12, RF-31
  Hecho cuando: A remote watch rejects an unsafe local source or configuration overlap and retains the existing log-overlap behavior.

## Authentication And Remote Contract

- [ ] **10. Define execution-scoped username/password values** (15 min)
  RF: RF-05, RF-30
  Hecho cuando: Domain tests represent credentials without including either value in destination identity or persistent-state models.

- [ ] **11. Prompt once for remote username and password** (20 min)
  RF: RF-05, RF-06
  Hecho cuando: An isolated CLI test obtains both values once for one watch execution and never includes them in the URL.

- [ ] **12. Handle cancelled, absent, and failed credential I/O** (20 min)
  RF: RF-06, RF-37
  Hecho cuando: EOF, cancellation, input failure, and prompt-output failure each stop before connecting or changing remote data.

- [ ] **13. Sanitize credential-bearing reports and state** (20 min)
  RF: RF-05, RF-06, RF-30
  Hecho cuando: Endpoint displays, credential outcomes, and early logging tests expose no username/password even in server-echoed errors.

- [ ] **14. Resolve established SSH trust by host and port** (25 min)
  RF: RF-08
  Hecho cuando: Temporary SSH trust fixtures distinguish a matching identity, an unknown host, a changed key, and another port.

- [ ] **15. Refuse untrusted SFTP before sending credentials** (20 min)
  RF: RF-06, RF-08
  Hecho cuando: A fake connection records no password transmission or remote mutation for unknown, changed, or unavailable trust.

- [ ] **16. Define an injectable remote destination operations contract** (25 min)
  RF: RF-10, RF-13, RF-18, RF-26, RF-28
  Hecho cuando: Contract tests exercise typed inspection, read/write, create, delete, publish, and close outcomes through a fake.

- [ ] **17. Enforce the selected remote path boundary** (20 min)
  RF: RF-04, RF-10, RF-11
  Hecho cuando: Attempts to operate on the server root, redirected ancestors, or paths outside the selected directory fail before mutation.

- [ ] **18. Represent regular, directory, link, special, and unknown entries** (15 min)
  RF: RF-11, RF-19, RF-28
  Hecho cuando: Inspection results distinguish all five types and never classify unknown entries as empty directories or regular files.

- [ ] **19. Model required server capabilities and refusal outcomes** (20 min)
  RF: RF-25, RF-28, RF-32
  Hecho cuando: Contract tests distinguish supported capabilities from missing typed listings, safe names, verification, and ownership.

- [ ] **20. Preserve close and cleanup failures separately** (20 min)
  RF: RF-26, RF-27, RF-37
  Hecho cuando: Tests return both a primary operation error and a distinct connection-close or cleanup error.

- [ ] **21. Provide a reusable fault-injectable remote contract fixture** (20 min)
  RF: RF-26, RF-28, RF-37
  Hecho cuando: Tests can fail each remote operation independently without accessing the network or user data.

## FTP Adapter

- [ ] **22. Build an isolated FTP control-channel test fixture** (25 min)
  RF: RF-07, RF-26
  Hecho cuando: A loopback fixture accepts one session and produces configurable success or failure replies without external servers.

- [ ] **23. Add FTP data-channel and disconnect fixtures** (25 min)
  RF: RF-07, RF-26, RF-35
  Hecho cuando: Tests can transfer bytes and disconnect before or after a final reply deterministically.

- [ ] **24. Connect and authenticate using ordinary FTP** (25 min)
  RF: RF-05 through RF-07
  Hecho cuando: A loopback test accepts username/password and rejects failed login without any destination write.

- [ ] **25. Require binary transfers and safe data-channel handling** (25 min)
  RF: RF-07, RF-23, RF-28
  Hecho cuando: A binary payload round-trips unchanged and data-channel failures remain visible to the caller.

- [ ] **26. Obtain complete typed FTP directory listings** (25 min)
  RF: RF-11, RF-28
  Hecho cuando: Fixture tests enumerate files, directories, and links by type; unsupported/incomplete listings fail instead of guessing.

- [ ] **27. Reject unsafe FTP entry types and redirects** (20 min)
  RF: RF-10, RF-11, RF-28
  Hecho cuando: FTP fixture tests refuse redirected ancestors or unknown managed entries before destructive operations.

- [ ] **28. Read FTP file contents for digest verification** (25 min)
  RF: RF-13, RF-16, RF-23, RF-28
  Hecho cuando: Binary, zero-length, and unreadable file fixtures yield correct bytes or explanatory read failures.

- [ ] **29. Upload FTP file contents without treating partial writes as success** (25 min)
  RF: RF-13, RF-18, RF-26
  Hecho cuando: An interrupted upload remains unconfirmed and both data and final control replies are checked.

- [ ] **30. Create FTP destination directories safely** (20 min)
  RF: RF-10, RF-13
  Hecho cuando: A missing selected subdirectory can be created, while existing files and redirected parents remain unchanged.

- [ ] **31. Delete FTP files and directories within the selected root** (25 min)
  RF: RF-13, RF-15, RF-19
  Hecho cuando: Tests delete only selected entries and propagate failures without touching a sibling or parent directory.

- [ ] **32. Publish FTP candidates and propagate session-close errors** (25 min)
  RF: RF-23, RF-26, RF-35
  Hecho cuando: A verified candidate receives a final unique name and rename or close failures are not treated as confirmation.

## SFTP Adapter

- [ ] **33. Select and pin justified Go SSH/SFTP dependencies** (20 min)
  RF: RF-08, RF-28
  Hecho cuando: Dependency versions are fixed and their need for SSH/SFTP absent from the standard library is recorded.

- [ ] **34. Build an isolated SSH host-key test fixture** (25 min)
  RF: RF-08
  Hecho cuando: A loopback SSH fixture presents controlled matching, unknown, and changed host identities.

- [ ] **35. Add a disposable SFTP filesystem fixture** (25 min)
  RF: RF-10, RF-11, RF-28
  Hecho cuando: The SSH fixture exposes isolated files, directories, and injected SFTP failures without external servers.

- [ ] **36. Verify SFTP trust before authentication** (25 min)
  RF: RF-06, RF-08
  Hecho cuando: Tests observe zero password attempts and writes when host-and-port trust is absent or mismatched.

- [ ] **37. Authenticate SFTP with an interactive password** (20 min)
  RF: RF-05, RF-06, RF-08
  Hecho cuando: Correct credentials open a session; rejection stops before inspecting or changing destination entries.

- [ ] **38. Inspect SFTP entries without following links** (25 min)
  RF: RF-10, RF-11, RF-19
  Hecho cuando: Fixture tests distinguish directories, files, links, and unknown types without traversing redirected ancestors.

- [ ] **39. Read SFTP file bytes with complete error reporting** (20 min)
  RF: RF-16, RF-21, RF-23, RF-28
  Hecho cuando: A binary file yields the expected digest and a partial read returns an error rather than matching contents.

- [ ] **40. Upload SFTP files with partial-write detection** (25 min)
  RF: RF-13, RF-18, RF-26
  Hecho cuando: A complete payload round-trips, while a short write or failed close never marks it complete.

- [ ] **41. Create SFTP directories safely** (20 min)
  RF: RF-10, RF-13
  Hecho cuando: Only missing directories under the selected path are created; files and redirected ancestors cause refusal.

- [ ] **42. Delete SFTP files and directories safely** (20 min)
  RF: RF-13, RF-15, RF-19
  Hecho cuando: Selected entries are removed and an injected refusal leaves unrelated entries untouched.

- [ ] **43. Publish SFTP candidates and report close failures** (25 min)
  RF: RF-23, RF-26, RF-35
  Hecho cuando: Publication can be verified by read-back, and rename, file-close, and session-close errors propagate.

## Inventory, Shared State, And Ownership

- [ ] **44. Build a source inventory before remote mutation** (20 min)
  RF: RF-09, RF-12, RF-34
  Hecho cuando: Missing, unreadable, linked, or special source entries fail before the remote fake receives a write.

- [ ] **45. Build complete remote inventories from file contents** (25 min)
  RF: RF-13, RF-16, RF-21, RF-28
  Hecho cuando: Directory, empty-file, and file-content changes alter the inventory; timestamps alone do not.

- [ ] **46. Preflight remote links, unknown entries, and ancestors** (20 min)
  RF: RF-10, RF-11, RF-28
  Hecho cuando: A mirror rejects unsupported entries before any write, and an unsupported BACKUP path or managed entry blocks its cycle.

- [ ] **47. Detect unrepresentable and colliding remote names** (25 min)
  RF: RF-04, RF-28
  Hecho cuando: Tests reject names altered, merged by case, or escaped by either protocol before claiming successful equivalence.

- [ ] **48. Define the shared managed backup history schema** (20 min)
  RF: RF-18 through RF-21, RF-24, RF-33
  Hecho cuando: Domain tests model actual destination, canonical source, ordered confirmed records, and pending removal separately from foreign entries.

- [ ] **49. Validate shared history encoding strictly** (25 min)
  RF: RF-19 through RF-21, RF-33
  Hecho cuando: Round-trip tests accept one complete state and reject malformed, missing, duplicate, wrong-order, or unknown fields.

- [ ] **50. Load missing, valid, and unsafe history outcomes** (20 min)
  RF: RF-19, RF-21, RF-33
  Hecho cuando: Tests distinguish empty unbound history, valid history, absent state with managed entries, and unreadable state without mutation.

- [ ] **51. Commit shared history without exposing partial records** (25 min)
  RF: RF-21, RF-24, RF-35, RF-37
  Hecho cuando: Interrupted-write tests expose either the previous durable record or the complete new record, never a partial confirmation.

- [ ] **52. Encode a remote archive's source/destination identity** (20 min)
  RF: RF-18 through RF-22, RF-35
  Hecho cuando: Archive identity tests bind a candidate to one canonical source and actual destination without changing the local ZIP contract.

- [ ] **53. Classify managed, foreign, and conflicting entries** (25 min)
  RF: RF-19, RF-22, RF-32, RF-33
  Hecho cuando: Tests preserve unrelated files and links, recognize owned candidates, and reject name collisions without deleting anything.

- [ ] **54. Verify every previously confirmed remote archive** (25 min)
  RF: RF-21, RF-33
  Hecho cuando: Missing, altered, corrupt, unreadable, or wrong-source versions block cleanup, publication, and retention.

- [ ] **55. Enforce one canonical source binding** (20 min)
  RF: RF-20, RF-33
  Hecho cuando: Source path aliases remain accepted but a different canonical source is rejected before any destination mutation.

- [ ] **56. Resolve actual remote directory identity across URL aliases** (25 min)
  RF: RF-20, RF-25, RF-32
  Hecho cuando: Two fixture URLs to the same directory yield one ownership identity; unprovable equivalence fails closed.

- [ ] **57. Define exclusive remote ownership outcomes** (20 min)
  RF: RF-25, RF-26, RF-32
  Hecho cuando: Contract tests distinguish current owner, competing owner, safe recovery, and uncertain/unsupported ownership.

- [ ] **58. Acquire an exclusive shared destination claim** (25 min)
  RF: RF-25, RF-32
  Hecho cuando: A server fixture grants only one owner and reports unsupported exclusivity before the first backup mutation.

- [ ] **59. Reject competing owners across clients and aliases** (25 min)
  RF: RF-20, RF-25, RF-32
  Hecho cuando: A second client or alias cannot create, verify, or remove an entry while the first holds ownership.

- [ ] **60. Handle stale or uncertain remote ownership conservatively** (25 min)
  RF: RF-22, RF-25, RF-33, RF-35
  Hecho cuando: Proven-safe recovery succeeds, whereas an ambiguous abandoned owner blocks publication and deletion.

- [ ] **61. Release remote ownership and preserve release errors** (20 min)
  RF: RF-25, RF-26, RF-37
  Hecho cuando: Normal shutdown releases ownership; a failed release is reported even when the cycle already failed.

## Remote Unidirectional Mirror

- [ ] **62. Detect confirmed and unconfirmed backup footprints** (25 min)
  RF: RF-19, RF-32, RF-33
  Hecho cuando: A read-only scan identifies managed history or uncertain classification and forbids mirror mutation.

- [ ] **63. Block a mirror while BACKUP owns the destination** (20 min)
  RF: RF-25, RF-32
  Hecho cuando: Active ownership reached through either URL causes mirror rejection without a copy or deletion.

- [ ] **64. Plan source-only additions for the remote mirror** (20 min)
  RF: RF-12, RF-13
  Hecho cuando: A local-only nested file or empty directory yields only remote creation actions.

- [ ] **65. Plan source-authoritative file and type replacements** (20 min)
  RF: RF-12 through RF-14
  Hecho cuando: Changed contents and file/directory conflicts choose the local entry without prompting.

- [ ] **66. Plan deletion of remote-only entries** (20 min)
  RF: RF-12, RF-13, RF-15
  Hecho cuando: An ordered plan removes only remote extras beneath the chosen directory, including for an empty source.

- [ ] **67. Create a missing remote mirror destination** (20 min)
  RF: RF-09, RF-10, RF-13
  Hecho cuando: A valid source populates a missing directory; an invalid source leaves the missing destination absent.

- [ ] **68. Execute remote file and directory creations** (25 min)
  RF: RF-12 through RF-14
  Hecho cuando: Files, zero-length files, and empty directories appear remotely with unchanged local bytes.

- [ ] **69. Execute remote replacements and removals safely** (25 min)
  RF: RF-11 through RF-15, RF-26
  Hecho cuando: Type conflicts and nested deletions converge to the source; injected remote failures stop further actions.

- [ ] **70. Mirror an empty local directory exactly** (20 min)
  RF: RF-13, RF-15, RF-32
  Hecho cuando: A successful empty-source cycle leaves an empty selected remote directory or rejects protected BACKUP history.

- [ ] **71. Verify final remote structure and file contents** (25 min)
  RF: RF-13, RF-16, RF-28, RF-36
  Hecho cuando: Missing, extra, altered, or concurrently changed remote entries prevent a successful mirror result.

- [ ] **72. Detect local changes between inventory and confirmation** (20 min)
  RF: RF-16, RF-34
  Hecho cuando: Logical changes invalidate the cycle while metadata-only changes do not.

- [ ] **73. Commit confirmed mirror state only after verification** (25 min)
  RF: RF-16, RF-17, RF-27, RF-37
  Hecho cuando: A successful mirror commits its record, while verification or persistence failure leaves the previous record intact.

- [ ] **74. Reassess a partial mirror at the next execution** (20 min)
  RF: RF-12, RF-17, RF-26, RF-34
  Hecho cuando: After interrupted writes, a later cycle re-inventories both sides and never copies remote-only changes to the source.

- [ ] **75. Coordinate one complete mirror cycle with fake transport** (25 min)
  RF: RF-09 through RF-17, RF-27, RF-32, RF-34
  Hecho cuando: A fake-network cycle performs preflight, exact copy/delete, stable verification, and final confirmation in order.

## Remote BACKUP Publication And Retention

- [ ] **76. Expose full and empty local ZIPs to the remote workflow** (20 min)
  RF: RF-12, RF-18, RF-23
  Hecho cuando: A remote-workflow unit test receives valid nonempty and empty ZIPs while local BACKUP tests remain unchanged.

- [ ] **77. Allocate a unique remote archive name and order** (20 min)
  RF: RF-18, RF-20, RF-24
  Hecho cuando: Two versions with the same clock still receive distinct names and consecutive confirmation orders.

- [ ] **78. Track a newly created remote backup directory** (20 min)
  RF: RF-09, RF-10, RF-27
  Hecho cuando: A failed first cycle removes its new empty directory or reports both primary and rollback failures.

- [ ] **79. Reject invalid previously confirmed backup history** (25 min)
  RF: RF-20, RF-21, RF-33
  Hecho cuando: Wrong source, missing archive, invalid order, or corrupt history fails before candidate recovery or publication.

- [ ] **80. Permit unrelated entries without changing them** (20 min)
  RF: RF-11, RF-19, RF-36
  Hecho cuando: Classification and recovery tests leave foreign files, directories, links, and special entries untouched.

- [ ] **81. Reject conflicts between foreign and managed entries** (20 min)
  RF: RF-19, RF-22, RF-33
  Hecho cuando: A foreign entry occupying a required managed name prevents any destination change.

- [ ] **82. Restore pending retention compliance before publishing** (25 min)
  RF: RF-21, RF-24, RF-27
  Hecho cuando: Gate tests prune an over-limit history first and refuse candidate creation when compliance cannot be restored.

- [ ] **83. Identify interrupted, unconfirmed managed candidates** (20 min)
  RF: RF-19, RF-22, RF-35
  Hecho cuando: Owned candidates are recognized without classifying a similar user ZIP as GoSync-managed.

- [ ] **84. Remove only validated unconfirmed candidates** (20 min)
  RF: RF-19, RF-22, RF-26
  Hecho cuando: Recovery removes owned incomplete work and an injected deletion failure prevents new publication.

- [ ] **85. Upload a distinguishable remote ZIP candidate** (25 min)
  RF: RF-18, RF-22, RF-23, RF-35
  Hecho cuando: A partial transfer remains unconfirmed and cannot collide with an existing or foreign destination entry.

- [ ] **86. Read back and verify the transferred ZIP** (25 min)
  RF: RF-18, RF-21, RF-23, RF-28
  Hecho cuando: Remote ZIP bytes, identity, hierarchy, and file contents match the source or publication is rejected.

- [ ] **87. Recheck source stability before backup confirmation** (20 min)
  RF: RF-09, RF-12, RF-23
  Hecho cuando: Changed source bytes, names, or types leave the ZIP unconfirmed; metadata-only changes remain acceptable.

- [ ] **88. Publish a verified candidate with a unique final name** (25 min)
  RF: RF-18, RF-23, RF-35
  Hecho cuando: A final ZIP appears only after read-back and source checks; uncertain publication is not declared confirmed.

- [ ] **89. Commit one durable, source-bound confirmation** (25 min)
  RF: RF-20, RF-23, RF-24, RF-33, RF-35
  Hecho cuando: Every client sees the same confirmed ZIP, source, digest, and order after a successful commit.

- [ ] **90. Reconcile a lost confirmation acknowledgment** (25 min)
  RF: RF-22, RF-26, RF-35
  Hecho cuando: Durable proof preserves an actual confirmation; no proof leaves a readable ZIP unconfirmed for owned recovery.

- [ ] **91. Handle interrupted publication without adoption** (20 min)
  RF: RF-22, RF-27, RF-35
  Hecho cuando: A later execution removes an identifiable published-but-unconfirmed ZIP instead of confirming it by appearance.

- [ ] **92. Select expired versions by confirmation order** (20 min)
  RF: RF-19, RF-24
  Hecho cuando: Retention counts 1 and 100 ignore clocks and never select unrelated or newest confirmed entries.

- [ ] **93. Record pending retention removals durably** (25 min)
  RF: RF-21, RF-24, RF-27
  Hecho cuando: An interruption between removal stages remains distinguishable from external corruption on the next execution.

- [ ] **94. Delete only expired confirmed managed ZIPs** (20 min)
  RF: RF-19, RF-24, RF-36
  Hecho cuando: Retention removes only selected managed versions and leaves all foreign and newer files intact.

- [ ] **95. Preserve confirmation on failed retention deletion** (20 min)
  RF: RF-24, RF-26, RF-27
  Hecho cuando: A delete failure keeps the new version confirmed, stops the cycle, and blocks later publication until compliant.

- [ ] **96. Coordinate one complete backup cycle with fake transport** (25 min)
  RF: RF-18 through RF-27, RF-33, RF-35
  Hecho cuando: A fake-network cycle validates, recovers, transfers, verifies, confirms, then retains in that order.

## Watch, Logging, And Failure Integration

- [ ] **97. Route remote cycles through the existing watch command** (25 min)
  RF: RF-01 through RF-03, RF-29
  Hecho cuando: Remote modes select their coordinator while both local paths still invoke the original behavior.

- [ ] **98. Hold backup ownership throughout a watch execution** (20 min)
  RF: RF-20, RF-25, RF-29
  Hecho cuando: An active remote BACKUP watch rejects a second owner between cycles, not just during transfers.

- [ ] **99. Run immediately and wait after each nonterminal cycle** (20 min)
  RF: RF-23, RF-29
  Hecho cuando: Fake waits show an immediate first cycle, configured post-cycle intervals, and no overlap.

- [ ] **100. Retry a changed local source only at the next interval** (20 min)
  RF: RF-16, RF-23, RF-29, RF-34
  Hecho cuando: A source-only change leaves state unchanged, reports the cycle, and starts exactly one later retry.

- [ ] **101. Stop remote watch on network or remote operation failure** (20 min)
  RF: RF-06, RF-26, RF-27, RF-29
  Hecho cuando: A failed FTP/SFTP operation reports an error and no later cycle runs automatically.

- [ ] **102. Report backup and mirror lifecycle events** (20 min)
  RF: RF-26, RF-30, RF-31
  Hecho cuando: Logs identify start, mutation, verification, confirmation, recovery, retention, success, and failure without contents.

- [ ] **103. Sanitize remote errors for logs and the console** (20 min)
  RF: RF-05, RF-06, RF-30
  Hecho cuando: Server errors containing the password, username, or URL secrets remain diagnosable but disclose none of them.

- [ ] **104. Preserve fallback, cleanup, and release error outcomes** (25 min)
  RF: RF-26, RF-27, RF-30, RF-31, RF-37
  Hecho cuando: Tests cover console/persistent-log fallback, fatal dual failure, and separate primary, cleanup, close, and release errors.

## End-to-End And Regression Verification

- [ ] **105. Exercise FTP mirror end-to-end on a disposable server** (25 min)
  RF: RF-07, RF-09 through RF-17, RF-28
  Hecho cuando: A real FTP fixture mirrors additions, replacements, deletions, and an empty source with final byte equality.

- [ ] **106. Exercise SFTP mirror end-to-end on a disposable server** (25 min)
  RF: RF-08, RF-09 through RF-17, RF-28
  Hecho cuando: A trusted-password SFTP fixture reaches the same exact mirror without changing source bytes.

- [ ] **107. Exercise FTP BACKUP end-to-end on a disposable server** (25 min)
  RF: RF-18 through RF-24, RF-28
  Hecho cuando: A real FTP fixture retains verified ZIPs while leaving unrelated destination entries unchanged.

- [ ] **108. Exercise SFTP BACKUP end-to-end on a disposable server** (25 min)
  RF: RF-18 through RF-24, RF-28
  Hecho cuando: A real SFTP fixture publishes and prunes verified versions without modifying foreign files or the source.

- [ ] **109. Verify real-process remote ownership** (25 min)
  RF: RF-20, RF-25, RF-33
  Hecho cuando: A second process cannot create, verify, or remove files while another process owns the same destination.

- [ ] **110. Verify URL-alias remote ownership** (25 min)
  RF: RF-20, RF-25, RF-32
  Hecho cuando: Two URL spellings of one directory observe the same owner and a mirror cannot bypass that owner.

- [ ] **111. Cover interrupted acknowledgments** (25 min)
  RF: RF-22, RF-23, RF-35
  Hecho cuando: Disconnects before and after confirmation preserve only durably proved versions as confirmed.

- [ ] **112. Reject concurrent remote content changes** (20 min)
  RF: RF-16, RF-21, RF-36
  Hecho cuando: Injected external edits never produce a confirmed mirror or version with unverified contents.

- [ ] **113. Run remote I/O and state failure tests** (25 min)
  RF: RF-06, RF-26 through RF-28, RF-37
  Hecho cuando: Listing, transfer, state, retention, close, and cleanup failures preserve prior confirmation and report every cause.

- [ ] **114. Verify log fallback and protected-data exclusions** (20 min)
  RF: RF-05, RF-30, RF-31
  Hecho cuando: Console/persistent failures follow existing fallback rules and neither logs nor user errors reveal secrets.

- [ ] **115. Run unchanged local-mode regression tests** (15 min)
  RF: RF-03, RF-31
  Hecho cuando: Local bidirectional, unidirectional, BACKUP, configuration, and logging tests still pass without contract changes.

- [ ] **116. Audit RF coverage and run the complete Go suite** (20 min)
  RF: RF-01 through RF-37
  Hecho cuando: Every RF has a passing test or explicit negative case and both `go test` and `go test ./...` pass.
