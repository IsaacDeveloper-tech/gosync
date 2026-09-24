# FTP/SFTP Destination Synchronization Specification

## Purpose

Extend the existing `watch` command so a local directory can be synchronized to a directory on an FTP or SFTP server in `unidirectional` or `backup` mode. Preserve the established local behavior and source-authoritative guarantees while making remote access failures, deletion boundaries, and backup ownership explicit.

## Command Interface

```text
gosync watch <local-directory> <ftp://server/remote-directory>
gosync watch <local-directory> <sftp://server/remote-directory>
```

The second argument identifies the remote destination directory; the configured synchronization mode determines whether that directory is an exact mirror or contains retained backup versions. The existing `gosync watch <local-directory-a> <local-directory-b>` command remains available.

## Scope and Precedence

For remote destinations only, this specification supersedes the FTP/SFTP exclusions in the local synchronization, configuration, and compressed backup specifications. It also supersedes the compressed backup specification's dedicated-directory and foreign-entry restrictions (RF-07, RF-15, their corresponding Out of Scope exclusions, and completion criteria) only to allow unrelated entries to coexist with one source-bound managed backup history. All other local-mode contracts and backup protections remain in force unless expressly superseded below. This distinction is required to make the historical specifications and the new remote acceptance criteria compatible without changing local behavior.

## Functional Requirements

RF-01. When `gosync watch` receives a local first argument and an `ftp://` or `sftp://` destination as its second argument, the system shall support that destination only while `unidirectional` or `backup` mode is active. This is required to add remote destinations without implying remote-to-local or two-way synchronization.

RF-02. If a remote URL is supplied as the source, the system shall reject the command before loading or creating configuration. If `bidirectional` mode is active with a remote destination, it shall reject the command before requesting credentials, connecting, or changing either location. In either case it shall report an explanatory error. This is required to keep the direction and supported modes unambiguous without starting configuration for an invalid source.

RF-03. While both `watch` arguments are local directories, the system shall retain the existing behavior of all three modes and their current configuration contract. This is required to avoid changing established local synchronization and backup operations.

RF-04. If the remote destination URL lacks a server or an explicit, absolute directory beneath the server-visible root, identifies that root itself, has a scheme other than `ftp` or `sftp`, contains embedded identity or credentials, contains a query or fragment, or has dot segments, encoded separators, or other path forms that escape or ambiguously identify the selected directory, the system shall reject it before loading or creating configuration, prompting for credentials, connecting, or modifying data. The selected remote directory shall be the boundary of every remote change. This is required to prevent ambiguous arguments from authorizing changes outside the intended destination.

RF-05. When a remote `watch` execution starts with a valid destination and supported mode, the system shall ask interactively for a username and password for that execution for both FTP and SFTP, without requiring them in the URL or saving them in GoSync configuration or confirmed state. This is required to define one usable authentication contract without retaining reusable secrets.

RF-06. If asking for or reading either credential fails, input is cancelled or unavailable, or authentication fails, the system shall report the cause and stop before changing the remote destination or confirming a cycle. This is required to prevent unattended or incorrectly authenticated synchronization and to make interactive I/O failures visible.

RF-07. While connecting to an `ftp://` destination, the system shall accept ordinary FTP, including its unencrypted connection behavior. This is required to support the requested FTP destinations without silently substituting a different protocol.

RF-08. Before sending credentials or changing data on an `sftp://` destination, the system shall verify the server identity against the user's previously established SSH trust for that server and port. If trust is absent, the identity has changed, or verification is unavailable, it shall report the failure and stop without trusting the server automatically. This is required to avoid disclosing credentials or data to an untrusted server.

RF-09. If the local source is missing, is not a readable directory, contains an unsupported entry, or cannot be completely inspected, the system shall report the cause and make no remote changes. This is required to prevent a bad or incomplete source view from authorizing remote deletion or a misleading backup.

RF-10. If the specified remote directory does not exist, the system shall create it before the first successful cycle; if it exists but is not an accessible directory, or the selected path would traverse an unsupported or redirected entry, the system shall report the error rather than replace or follow that entry. If the first backup cycle fails after creating a new, still-empty remote destination, it shall remove that directory or report the cleanup failure. This is required to support a new destination without changing a different location or leaving an unused backup directory after failure.

RF-11. Before changing a remote destination in `unidirectional` mode, the system shall reject the cycle without modifying either location if the destination contains a symbolic link, special entry, or entry whose type cannot be established safely. In `backup` mode, it shall leave unrelated symbolic links and special entries untouched, but shall reject an unsupported root, path ancestor, or managed entry before changing the destination. This is required to prevent unsafe traversal or deletion while allowing unrelated backup-directory contents to coexist.

RF-12. While a remote destination is selected, the system shall keep the first, local directory authoritative and shall never apply remote additions, modifications, or deletions to it. This is required to preserve the direction selected by `unidirectional` and `backup` modes.

RF-13. While `unidirectional` mode is active, the system shall propagate local additions and content changes to the remote directory and delete remote items absent from the local source, so that a successful cycle leaves the remote structure, names, entry types, and file contents equivalent to the local source. This is required to retain exact-mirror semantics, including for a pre-existing remote directory.

RF-14. When local and remote items conflict in `unidirectional` mode, including a file-versus-directory conflict, the system shall apply the local version without asking which side wins. This is required to apply source authority consistently.

RF-15. When the local source is an empty directory in `unidirectional` mode, the system shall leave the selected remote directory empty after a successful cycle. This is required to make an empty source an exact mirror rather than an implicit request to preserve old remote data.

RF-16. When a remote `unidirectional` cycle finishes, the system shall confirm synchronized state only after verifying the remote structure, names, entry types, and file contents against a local source whose logical contents remained stable throughout the cycle. If verification or source-stability checking fails, it shall preserve the previous confirmed state and report the failure. This is required to avoid treating a partial transfer or mixed source states as a completed mirror.

RF-17. When a later `unidirectional` execution starts after an interrupted or failed remote cycle, the system shall reassess the current local and remote contents using the local source as authority rather than assuming the failed cycle completed. This is required to repair partial remote changes without propagating them back to the source.

RF-18. While `backup` mode is active with a remote destination, the system shall produce complete, independently usable ZIP versions of the local source and apply the configured count-based retention policy to confirmed versions. This is required to preserve the existing backup purpose and retention limit on FTP and SFTP.

RF-19. When a remote backup destination contains files or directories not managed by GoSync, the system shall allow them to remain, shall never change or remove them, and shall apply recovery and retention only to identifiable GoSync-managed backup entries. If an unrelated entry conflicts with a required managed entry or managed entries cannot be distinguished safely, it shall fail the cycle without changing the destination. This requirement supersedes the rejection of all foreign destination entries in RF-15 of the compressed backup specification. This is required to support a shared directory without taking ownership of unrelated data.

RF-20. When the first remote backup version is confirmed, the system shall bind the managed backup history in that actual remote directory to one local source's absolute canonical path. On later executions, it shall reject a different source before publishing or deleting managed entries, including when the source has identical contents or the remote directory is reached through a different URL. This is required to prevent one source from inheriting or pruning another source's versions.

RF-21. Before changing existing managed backup entries, the system shall verify the ownership and integrity of all previously confirmed managed versions and shall distinguish them from incomplete managed work. If a confirmed version is missing, corrupt, unreadable, or attributed to a different source, it shall report the problem and leave the destination unchanged. This is required to prevent new versions or retention from hiding damage to established history.

RF-22. When an interrupted remote backup has left identifiable unconfirmed managed work, the system shall report and remove only that work before publishing another version, without promoting an unconfirmed ZIP based only on its apparent completeness. If it cannot be removed, it shall publish nothing new and report the failure. This is required to recover safely without deleting unrelated destination contents or accepting an uncertain backup as confirmed.

RF-23. When a remote ZIP has been transferred, the system shall verify that the remote version is readable and represents the complete, stable local source before confirming it. If remote verification fails, it shall leave the version unconfirmed, preserve all previously confirmed versions, report the failure, and stop `watch`. If only the local source changes and remote cleanup succeeds, it shall leave the version unconfirmed, report the inconsistent source, and try a new cycle after the configured interval. This is required to distinguish a successful upload from a usable backup while preserving the established source-change behavior.

RF-24. When a new remote backup version is confirmed, the system shall retain only the configured number of newest confirmed versions belonging to the bound source, in their confirmation order. If pruning fails, it shall keep the newly confirmed version and existing valid versions, report the error, and prevent further publication until the managed history meets the configured limit. This is required to bound managed backup growth without sacrificing confirmed history or unrelated files.

RF-25. While one backup execution owns an actual remote managed backup destination, the system shall reject another execution attempting to manage it, including from a different computer or through a different URL; the rejected execution shall not create, verify, or remove anything there. If reliable exclusive ownership of that destination cannot be established, it shall reject the backup before changing the destination. This is required to prevent competing publication and retention decisions even when the directory also contains unrelated files.

RF-26. If connection, authentication, remote inspection, transfer, remote verification, deletion, remote cleanup, connection closure, or remote ownership release fails during a remote cycle or on termination of a remote `watch` execution, the system shall report the cause and stop that execution without automatically retrying; a new execution may reassess and recover incomplete work. This requirement supersedes next-interval retry for remote-side blocked entries in RF-22 of the compressed backup specification. This is required to make remote failures visible rather than repeatedly acting on an uncertain destination.

RF-27. If a remote cycle fails after making some changes, the system shall not mark an incomplete mirror or backup as successful, shall preserve the previous mirror's confirmed state and the previously confirmed backup versions except for expired versions whose retention deletion already succeeded after a new version was confirmed, and shall report any additional cleanup failure. This is required to prevent partial transfers or failed deletions from becoming authoritative history while respecting completed retention.

RF-28. If the server cannot expose every item and its type within the selected managed scope, represent the source's relative names and distinct paths without collisions or substitutions, transfer file contents without alteration, or allow complete content-and-structure verification, the system shall report the unsupported behavior and fail rather than claim success. If incompatibility can be established from the local source before remote changes, it shall reject the cycle before changing the destination. This is required to keep synchronization guarantees meaningful across differing FTP and SFTP servers.

RF-29. When a remote `watch` execution starts successfully, the system shall run a cycle immediately and, after each cycle that does not terminate the execution, wait the configured interval before the next cycle without overlap; it shall keep the mode, interval, and backup retention selected at startup for that execution. This is required to preserve the existing predictable `watch` schedule and configuration behavior.

RF-30. When a remote cycle starts, succeeds, fails, changes destination content, confirms a backup, or recovers incomplete work, the system shall record the event under the log system specification without recording credentials, secrets, or file contents, including in destination URLs and error text. If a remote error includes protected data, the user-facing report shall also omit that data while retaining the failure cause. This is required to provide usable diagnostics without exposing access information.

RF-31. While synchronizing remotely, the system shall retain the established local-source protections, configuration and log location protections, error reporting, and logging fallback rules wherever they apply to the local source. This is required to add a remote destination without weakening existing safety or operational visibility.

RF-32. If a remote directory selected for `unidirectional` mode contains confirmed or identifiable unconfirmed GoSync-managed backup entries, or is currently owned by a backup execution, the system shall reject the mirror before modifying the destination, even when those entries belong to the same source or are reached through a different URL. If the system cannot establish whether protected managed backup entries or an active backup owner are present, it shall reject the mirror before modifying the destination. This is required to prevent a mode change or concurrent mirror from silently destroying backup history.

RF-33. If a remote backup directory already contains managed versions but the source association, confirmation order, or integrity of that history cannot be established, the system shall reject the cycle before changing any destination entries; it shall not bind the existing history to a new source. This is required to avoid misidentifying confirmed backups after a move, interruption, or execution from another computer.

RF-34. If a local source entry's name, type, hierarchy, or file contents change during a remote `unidirectional` cycle before state confirmation, the system shall report an inconsistent source, leave the previous confirmed state unchanged, and re-evaluate on the next configured interval unless a remote failure has already stopped `watch`. Changes only to metadata excluded by the local synchronization contract shall not invalidate the cycle. This is required to avoid confirming a mirror assembled from different logical source states.

RF-35. If a connection interruption leaves the outcome of a remote backup transfer or confirmation uncertain, the system shall treat the version as unconfirmed unless complete, durable confirmation can be established. On a later execution, it shall remove identifiable unconfirmed managed work before publishing again and shall not infer confirmation merely from a readable ZIP. This is required to prevent lost acknowledgments from turning an unconfirmed transfer into trusted history.

RF-36. If another actor changes managed backup entries or changes mirror contents while a cycle is in progress, the system shall not confirm a result whose ownership, integrity, or final contents cannot be verified. Unrelated changes in a shared backup directory shall not be removed or treated as managed work unless they conflict with the managed history. This is required to avoid confirming an inconsistent destination while respecting unrelated user data.

RF-37. If storing or reading confirmed-state information, recording remote backup ownership, or closing a local resource required for a remote cycle fails, the system shall report the cause, stop the affected execution, and shall not treat incomplete work as confirmed; if more than one failure occurs, it shall report each relevant cause. This is required to handle local I/O failures with the same integrity guarantees as remote failures.

## Out of Scope

- FTP or SFTP as a source, bidirectional remote synchronization, and remote-to-local restoration.
- Remote destinations in modes other than `unidirectional` and `backup`.
- FTPS, cloud storage, HTTP-based protocols, and automatic upgrading of ordinary FTP to an encrypted protocol.
- Automatically trusting an unknown or changed SFTP server identity or managing trusted server identities within `watch`.
- FTP anonymous access, SFTP key-based authentication, or any login other than an interactively supplied username and password.
- Supplying credentials in the destination URL, saving them in GoSync configuration, or reusing them across executions.
- Treating the server-visible root `/` as a destination or following a remote path outside the explicitly selected directory.
- Multiple local sources sharing one managed backup history, even though unrelated files may coexist in its directory.
- Deleting or applying backup retention to files not managed by GoSync.
- Incremental backups, remote ZIP restoration, new backup retention rules, or changing the local backup archive contract.
- Changing local-to-local behavior, configuration schemas, or the interactive `configure` command.

## Completion Criteria

- `watch` accepts FTP and SFTP destination URLs only for configured `unidirectional` and `backup` modes; existing local executions retain their behavior.
- Malformed, ambiguous, credential-bearing, or unsupported remote URLs fail before any connection or data change, and remote sources or bidirectional remote destinations are rejected.
- Each remote execution requests credentials without saving or logging them; authentication failures stop before changes, and SFTP rejects unknown, changed, or unverifiable server identities before sending credentials.
- A missing remote destination can be created, while missing or unreadable sources, unsupported entries, inaccessible remote paths, and insufficient server capabilities produce explanatory failures.
- Remote paths identify one absolute directory below the server-visible root; root, escaping, or ambiguous paths cannot authorize deletion, and aliases of the same directory cannot bypass backup ownership.
- An unidirectional run cannot delete managed backups in the selected remote directory, including after a mode change or through a different URL.
- Unsupported remote entries block all changes in unidirectional mode; unrelated links and special entries in a shared BACKUP directory remain untouched.
- A successful unidirectional cycle leaves the selected remote directory an exact content-and-structure mirror, including source-directed conflict resolution, deletion of remote-only entries, and empty sources; the local source is never modified from remote changes.
- A remote mirror is confirmed only after source stability and final verification; a changing source leaves confirmed state unchanged and triggers a later scheduled cycle, while a failed remote transfer stops `watch`.
- Remote BACKUP produces verified, complete ZIP versions for one bound local source and retains the configured number of newest confirmed managed versions.
- Unrelated entries in a remote backup directory remain untouched; managed-entry collisions, damaged or unidentifiable confirmed history, competing backup executions across machines and URL aliases, and unrecoverable incomplete managed work block publication or deletion.
- An uncertain upload or lost confirmation acknowledgment cannot promote an unconfirmed ZIP; confirmed history remains distinguishable on later executions.
- Transfer, verification, cleanup, and retention failures are reported without falsely confirming incomplete work or deleting unrelated entries; a confirmed version remains confirmed if subsequent retention fails.
- Remote cycles follow the existing immediate, completion-based configured schedule; local source changes can be retried at the next interval, while remote-operation failures stop the execution without automatic retry.
- Remote lifecycle, authentication, ownership, and failure reports comply with existing logging fallback and secret-exclusion requirements, including when server error text contains protected data.
- Automated tests verify every functional requirement and its defined error behavior for both protocols and modes, including malformed and aliased URLs, trust and credential failures, prompt-output failures, missing destinations and cleanup, source changes, unsupported and foreign entries, mode changes, interrupted confirmation, ownership across executions, retention failures, server naming or listing limitations, close and state-persistence failures, logging fallback, scheduling, and local-mode regressions.
