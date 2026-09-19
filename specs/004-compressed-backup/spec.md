# Compressed Backup Mode Specification

## Purpose

Provide a source-authoritative backup mode that periodically stores complete compressed versions of one directory in another directory, while bounding retained storage through configurable version retention.

## Functional Requirements

RF-01. When the system requests the synchronization mode, it shall present `BACKUP` as an available mode in addition to `bidirectional` and `unidirectional`. This is required to make compressed backup behavior explicitly selectable rather than treating it as ordinary synchronization. This requirement supersedes RF-05, RF-06, and the statements that `BACKUP` is unavailable or out of scope in the configuration system specification.

RF-02. When the user selects `BACKUP`, the system shall request a whole-number retention count from 1 through 100 inclusive and shall offer 3 as the default. This is required to let the user bound the number of complete backup versions while preventing impractically large or unrepresentable values.

RF-03. If the user provides an empty retention answer, the system shall select 3; if the user provides a fractional, non-numeric, out-of-range, or otherwise unrepresentable value, the system shall explain the invalid value and request it again. This is required to prevent an absent or unusable retention policy from becoming active.

RF-04. When a valid `BACKUP` configuration is ready for confirmation, the system shall include the mode and retention count in the configuration summary and shall save neither value unless the user explicitly confirms them. This is required to prevent unintended backup and deletion policies from becoming active.

RF-05. When configuration is saved after this capability is available, the system shall persist schema version 2. A `backup` configuration shall contain exactly `schemaVersion`, `synchronizationIntervalSeconds`, `synchronizationMode`, and `backupRetentionCount`; a `bidirectional` or `unidirectional` configuration shall contain exactly the first three properties and shall not contain `backupRetentionCount`. This is required to keep strict validation while storing backup-only policy only when it applies. This requirement supersedes RF-18 and RF-19 of the configuration system specification where they restrict the schema version, exact properties, and accepted modes.

RF-06. When the system loads a valid schema version 1 configuration, it shall continue to accept its existing `bidirectional` or `unidirectional` behavior. When it loads schema version 2, it shall strictly require the property set defined by RF-05 for the selected mode and shall reject missing, additional, duplicate, incorrectly capitalized, incorrectly typed, out-of-range, or unsupported values. This is required to preserve existing user configurations without weakening the strict configuration contract.

RF-07. While `BACKUP` mode is active, the system shall treat the first directory supplied to `gosync watch <directory-a> <directory-b>` as the source and the second as the dedicated backup destination. This is required to establish one unambiguous source-to-destination direction.

RF-08. If the source does not exist, is not a readable directory, or cannot be completely inspected before destination changes begin, the system shall report the cause, fail the cycle, and leave the destination and all previously confirmed backups unchanged. This is required to prevent an invalid source from causing destination changes or producing an uncertain backup.

RF-09. If the backup destination does not exist, the system shall create it as a directory for the first backup cycle. If that cycle does not complete with a confirmed version, the system shall remove the newly created empty destination; if removal fails, it shall report both the cycle failure and the cleanup failure. This is required to restore the observable pre-cycle state whenever possible while propagating every cleanup error.

RF-10. If the source and backup destination are equal, either contains the other, or GoSync's configuration location is equal to or contained within either root, the system shall reject the backup operation before changing either root. This is required to prevent recursive archives, self-inclusion, and configuration data from becoming backup content.

RF-11. If GoSync's persistent log location is equal to or contained within either backup root, the system shall warn through the console, disable persistent logging for the current execution, and allow backup to continue while console logging remains available. This is required to retain the log system specification's root-overlap behavior without including active logs in a backup.

RF-12. If an existing backup destination is not a directly accessed regular directory, including when it is a file, symbolic link, or special filesystem entry, the system shall report the unsupported destination type and leave it unchanged. This is required to prevent redirection and undefined ownership of the backup location.

RF-13. While one `BACKUP` process owns a destination, the system shall reject any other `BACKUP` process that attempts to use that destination and shall allow the rejected process to create, verify, or remove nothing there. This is required to prevent concurrent publication and retention from corrupting a backup set.

RF-14. When the first backup version is confirmed in a destination, the system shall bind that destination exclusively to the source's absolute canonical path. On later executions, it shall reject a different source path before creating or deleting anything, even if the other source has identical contents. This is required to prevent unrelated sources from sharing one retention history accidentally.

RF-15. Before changing an existing destination, the system shall distinguish confirmed versions belonging to its bound source, its own unconfirmed artifacts, and all other entries. If any foreign or unidentifiable entry is present, the system shall report it, reject the cycle, and leave the destination unchanged. This is required to ensure that retention never deletes user-managed or unrelated data.

RF-16. If destination inspection finds only valid confirmed versions and GoSync-owned unconfirmed artifacts, the system shall report and remove every unconfirmed artifact before starting a new backup. If any such removal fails, it shall report the failure and publish no new version. This is required to recover safely from interrupted work without treating incomplete output as user data or confirmed history.

RF-17. Before removing an unconfirmed artifact or publishing a new version, the system shall validate the ownership and integrity of every previously confirmed version. If a confirmed version is missing, altered, unreadable, corrupt, or no longer attributable to the bound source, it shall report the affected version, reject the cycle, and leave every destination entry unchanged. This is required to prevent retention or new publication from concealing damage to established backup history.

RF-18. When a `BACKUP` cycle runs for a non-empty source, the system shall produce one complete ZIP version containing the source hierarchy relative to the source root, including all regular files, empty directories, and zero-length files. When the source is empty, it shall produce one valid empty ZIP version. This is required to make each retained version independently represent the complete logical state of the source.

RF-19. While creating a backup version, the system shall store regular-file payloads using ZIP compression. The backup contract shall preserve relative names, entry types, hierarchy, and file contents but shall not require preservation of timestamps, permissions, ownership, access-control lists, extended attributes, or other metadata. This is required to reduce storage where content is compressible while keeping verification independent of platform-specific metadata.

RF-20. While `BACKUP` mode is active, the system shall never modify the source or propagate additions, changes, or deletions from the backup destination to the source. This is required to keep backup behavior strictly unidirectional and the source authoritative.

RF-21. If the source contains a symbolic link, socket, device, or any entry other than a regular file or directory at initial inspection or before publication, the system shall reject the entire cycle, publish no version, and identify the unsupported entry. This is required to avoid silently incomplete archives, traversal outside the source, and platform-dependent backup meaning.

RF-22. If a source entry, destination entry, or destination ownership operation is blocked by another program, the system shall report the blocked item, fail the current cycle without immediate retry, preserve all confirmed versions, and try again only at the next configured interval. This is required to prevent one blocked item from holding a cycle indefinitely while retaining predictable scheduled recovery.

RF-23. During a backup cycle, the system shall require the source's relative names, entry types, hierarchy, and file contents to remain unchanged from initial inspection through archive verification. If any of those change, the system shall discard the unconfirmed version, apply no retention deletion, report an inconsistent source, and wait until the next configured interval before trying again. Changes only to metadata excluded by RF-19 shall not invalidate the cycle. This is required to prevent a confirmed ZIP from combining different logical source states.

RF-24. If any inspection, open, read, write, close, verification, publication, confirmation, removal, ownership, or cleanup operation fails, the system shall report the specific cause, publish no incomplete version, and preserve every previously confirmed version except for an expired version whose successful deletion had already completed. Any unconfirmed artifact left because cleanup also failed shall remain classified as unconfirmed and shall be handled by RF-16. This is required to propagate every filesystem error without representing partial work as a successful backup.

RF-25. When ZIP creation finishes, the system shall confirm that the archive is readable and that its relative names, entry types, hierarchy, and file contents exactly match the stable logical source state defined by RF-23 before publishing it. This is required to distinguish successfully written bytes from a complete and usable backup.

RF-26. The system shall classify a version as confirmed only after archive verification, source-stability verification, source ownership association, and confirmation-order recording all complete successfully. If execution is interrupted before all confirmation conditions complete, the version shall remain unconfirmed and shall be removed under RF-16 during the next cycle. This is required to make confirmed status unambiguous across process interruption.

RF-27. When a backup version is confirmed, the system shall publish it with a unique `.zip` name containing a human-readable UTC creation date and time plus a collision-resistant identifier. This is required to make versions recognizable to users without allowing clock repetition to overwrite another version.

RF-28. The system shall assign each confirmed version an unambiguous confirmation order and shall use that order, rather than filename, filesystem timestamps, or wall-clock ordering, to determine which versions are newest for retention. This is required to preserve deterministic retention when clocks repeat, change, or move backwards.

RF-29. When a new backup version is confirmed, the system shall retain the configured number of newest confirmed versions by confirmation order and remove older confirmed versions. It shall never apply retention deletion before the new version is confirmed. This is required to bound normal backup storage without sacrificing known-valid history for a backup that might fail.

RF-30. If removal of an expired version fails after a new version is confirmed, the system shall keep the new version confirmed, report a retention failure for the cycle, and preserve all remaining valid versions. Before any later cycle publishes another version, it shall first restore compliance with the configured retention count; if cleanup still fails, it shall publish nothing new and shall never delete the newest confirmed version. This is required to prevent cleanup failures from causing backup loss or unbounded publication growth.

RF-31. When `gosync watch` starts with a valid `BACKUP` configuration, it shall run one backup cycle immediately, wait the configured interval after that cycle finishes, and never overlap backup cycles within that execution. This is required to retain the established completion-based scheduling contract.

RF-32. While a `BACKUP` watch execution is running, the system shall use the interval and retention count loaded at startup; a later execution shall load the current complete configuration. This is required to keep each execution internally consistent while making confirmed configuration changes effective predictably.

RF-33. When a backup cycle starts, succeeds, fails, verifies a version, confirms a version, removes an expired version, recovers an unconfirmed artifact, or rejects concurrent ownership, the system shall record the event and outcome according to the log system specification without recording file contents. This is required to make backup history and failures diagnosable without exposing protected data.

RF-34. While creating and retaining backup versions, the system shall continue to propagate every filesystem, configuration, logging, and scheduling error according to the existing specifications except where RF-01, RF-05, RF-06, RF-11, and RF-22 explicitly supersede earlier behavior. This is required to add backup behavior without weakening established safety and error-reporting guarantees.

## Out of Scope

- Restoring, extracting, browsing, mounting, or validating a backup through a dedicated GoSync command.
- Recovering an interrupted but unconfirmed ZIP as a confirmed version.
- Incremental, differential, deduplicated, or continuously appended archives.
- Password protection, encryption, signing, or key management for ZIP files.
- Guaranteed compression ratios or guaranteed storage savings for content that is already compressed or otherwise incompressible.
- Retention by age, total bytes, available space, calendar schedule, or any rule other than the configured version count.
- Selecting individual files, exclusion patterns, ignore files, or multiple source roots.
- Following symbolic links or archiving sockets, devices, and other special filesystem entries.
- Preserving timestamps, permissions, ownership, access-control lists, extended attributes, or other filesystem metadata.
- Sharing one backup destination between different canonical source paths, user-managed files, or unrelated backup sets.
- Waiting for another backup process to release a destination or allowing concurrent processes to share it.
- Immediate or indefinite retry of blocked files within the same backup cycle.
- FTP, SFTP, cloud, removable-media-specific behavior, or any non-local backup destination.
- Changing the behavior of existing `bidirectional` and `unidirectional` modes except for accepting the versioned configuration contracts defined here.

## Completion Criteria

- `BACKUP` supersedes its former unavailable status, defaults retention to 3, accepts only 1 through 100, displays the complete policy, and saves it only after confirmation.
- Schema version 2 strictly uses a mode-dependent property set, and valid schema version 1 configurations continue to run their existing modes.
- The first `watch` directory is always the immutable authoritative source, and each destination is exclusively bound to the first source's absolute canonical path.
- Missing, unreadable, overlapping, unsupported, foreign-owned, concurrently owned, or configuration-containing roots stop the cycle before backup publication or retention.
- A persistent log inside either root is disabled with a console warning rather than blocking backup, subject to the log system's remaining-destination rules.
- A failed first cycle removes the destination it created, while every cleanup failure is separately reported and no partial artifact is confirmed.
- Existing destinations distinguish confirmed versions, owned unconfirmed artifacts, and foreign entries; only owned unconfirmed artifacts are automatically removed.
- Every previously confirmed version is validated before destination mutation, and any missing, altered, unreadable, corrupt, or wrongly owned version blocks the cycle without deletion.
- Every successful cycle publishes exactly one independently usable ZIP, including a valid empty ZIP for an empty source, containing only the required relative hierarchy and file contents.
- Source structure and contents remain stable through verification; any logical change or blocked entry fails the cycle without immediate retry or retention deletion.
- A ZIP is confirmed only after archive, source-stability, ownership, and confirmation-order checks all succeed; interruption before that point leaves an unconfirmed artifact for later cleanup.
- Confirmed ZIP names contain UTC date and time plus a collision-resistant identifier, while retention ordering uses confirmation order independently from clocks and filesystem timestamps.
- The configured number of newest confirmed versions is retained only after a new version is confirmed.
- A retention deletion failure keeps the new version confirmed, reports a cycle error, and blocks later publication until the destination satisfies the configured limit.
- Backup cycles run immediately and then after the configured completion-based interval without overlap, using the startup configuration snapshot.
- Backup lifecycle, verification, ownership, recovery, concurrency, retention, and failure events follow the existing logging and protected-data rules.
- Restoration and all other capabilities listed under Out of Scope are not required for completion.
- Automated tests verify every functional requirement, including both configuration schemas, retention boundaries, empty sources, incompressible and zero-length files, root and process ownership, foreign entries, unsupported destination types, source changes, blocked entries, interrupted and failed writes, failed cleanup, prior-version corruption, archive verification, deterministic retention, scheduling, and source immutability.
