# Configuration System Specification

## Purpose

Allow each user to define GoSync's synchronization interval and synchronization mode once, retain those choices in a global JSON file, and apply them consistently to later command executions.

## Command Interface

```text
gosync configure
```

The `configure` command accepts no additional arguments, collects configuration values through interactive questions, and replaces the user's previous configuration only after a complete valid configuration is explicitly confirmed.

## Functional Requirements

RF-01. When the user invokes `gosync configure` without additional arguments, the system shall interactively request the synchronization interval and synchronization mode. This is required to make configuration available without requiring direct JSON editing or command options.

RF-02. If the user invokes `gosync configure` with any additional argument, the system shall display the valid command usage, stop configuration, and leave the existing configuration unchanged. This is required to keep the command contract unambiguous.

RF-03. When the system requests the synchronization interval, it shall accept a whole number from 1 through 86,400 seconds inclusive. This is required to provide a useful and verifiable range without accepting values that cannot serve as a practical schedule.

RF-04. If the user enters an interval outside the accepted range, a fractional value, or a non-numeric value, the system shall explain that the value is invalid and request it again. This is required to prevent an unusable synchronization schedule without discarding the configuration session.

RF-05. When the system requests the synchronization mode, it shall present `bidirectional` and `unidirectional` as available modes and shall display `BACKUP` as a planned mode that is not yet available. This is required to distinguish executable behavior from a future capability.

RF-06. If the user enters an unknown synchronization mode or selects unavailable `BACKUP`, the system shall explain why the selection is invalid and request the mode again. This is required to prevent an unsupported mode from becoming active without cancelling the configuration session.

RF-07. When the user has provided a valid interval and mode, the system shall display a summary and require explicit confirmation before saving. This is required to let the user detect unintended choices before replacing the active configuration.

RF-08. If the user rejects the summary, provides no further input, or interrupts configuration before confirming the save, the system shall cancel configuration and leave any previous configuration unchanged. This is required to prevent partial or unintended settings from becoming active.

RF-09. While `bidirectional` mode is active, the system shall retain the behavior defined by the local bidirectional synchronization specification except for its fixed five-second recheck interval, which RF-29 of this specification replaces. This is required to preserve the existing two-way synchronization contract while making its schedule configurable.

RF-10. When the user selects `unidirectional`, the system shall treat the first directory supplied to `gosync watch <directory-a> <directory-b>` as the source and the second directory as the destination. This is required to assign an unambiguous direction to the existing command arguments.

RF-11. If the source directory does not exist in `unidirectional` mode, the system shall stop and report an explanatory error without modifying the destination. This is required to prevent an absent source from being interpreted as authority to erase destination contents.

RF-12. If the destination directory does not exist in `unidirectional` mode, the system shall create it and replicate the source into it. This is required to support a new destination without reversing the configured direction.

RF-13. While `unidirectional` mode is active, the system shall make the destination an exact replica of the source by propagating source additions and changes and removing items that exist only in the destination. This is required to provide predictable source-to-destination mirroring.

RF-14. While `unidirectional` mode is active, the system shall not propagate destination additions, changes, or deletions back to the source. This is required to keep the configured source authoritative.

RF-15. While `unidirectional` mode is active, the system shall retain the local synchronization specification's protections for equal or nested roots, unsupported filesystem entries, filesystem errors, locked files, final verification, and confirmed-state updates after success. This is required to preserve established filesystem safety and recovery guarantees independently from synchronization direction.

RF-16. When source and destination differ in `unidirectional` mode, including content conflicts and file-directory conflicts, the system shall apply the source version without requesting a conflict decision. This is required to make source authority consistent for every destination difference.

RF-17. When a `unidirectional` synchronization completes successfully, the system shall confirm the resulting shared state, and when it fails, the system shall preserve the previous confirmed state. This is required to keep later executions and mode changes from treating incomplete work as confirmed synchronization.

RF-18. When configuration is saved, the system shall store one global `config.json` in GoSync's standard per-user application-data location as a JSON object containing exactly `schemaVersion`, `synchronizationIntervalSeconds`, and `synchronizationMode`. `schemaVersion` shall be `1`; the interval shall satisfy RF-03; and the mode shall be exactly `bidirectional` or `unidirectional`. This is required to provide a stable, explicit, and versioned configuration contract.

RF-19. When GoSync loads `config.json`, it shall reject malformed JSON, duplicate properties, unknown properties, missing properties, different property capitalization, unsupported schema versions, invalid types, out-of-range intervals, and unsupported modes. This is required to prevent ambiguous interpretation and unnoticed configuration mistakes.

RF-20. When the user confirms a valid configuration, the system shall replace the complete previous configuration so that a successful save leaves the new complete JSON and an interruption during replacement leaves either the previous complete JSON or the new complete JSON, never a partial document. This is required to preserve a usable configuration across write failures and process interruption.

RF-21. If locating application data, reading interactive input, reading configuration storage, serializing JSON, creating storage, writing, replacing, or closing configuration storage fails, the system shall report the cause, stop the requested configuration-dependent operation, and preserve the last complete valid configuration. This is required to handle every configuration and I/O error without silently applying uncertain values.

RF-22. When `gosync watch` receives valid command arguments and no configuration file exists, the system shall start the interactive configuration flow before performing any synchronization. This is required to ensure first-time synchronization behavior reflects an explicit user choice.

RF-23. When `gosync watch` receives invalid command arguments, the system shall display its usage error and stop before loading or creating configuration. This is required to avoid requesting settings for a command that cannot execute.

RF-24. When the configuration flow started by `gosync watch` completes and saves successfully, the system shall continue with synchronization using the newly confirmed values. If that flow is cancelled or fails, the system shall not synchronize. This is required to support first-time setup without running under incomplete or unconfirmed behavior.

RF-25. If an existing `config.json` is invalid, `gosync watch` shall stop with an explanatory configuration error, while `gosync configure` shall explain the invalidity and permit the user to replace it only after completing and confirming a valid configuration. This is required to prevent synchronization with invalid settings while preserving an interactive recovery path.

RF-26. If `config.json` cannot be located or read for a reason other than not existing, the system shall stop the requested command and report the cause. This is required to distinguish an I/O failure from a first-time configuration state.

RF-27. When an existing `config.json` is inspected, the system shall accept it only as a regular file and shall reject symbolic links, directories, and special filesystem entries with an explanatory error. This is required to prevent configuration from being redirected or read from an unsupported source.

RF-28. While one `gosync configure` process is active, the system shall reject another concurrent `gosync configure` process with an explanatory error and without changing `config.json`. This is required to prevent competing configuration sessions from overwriting or corrupting global settings.

RF-29. When `gosync watch` loads a valid configuration, the system shall synchronize immediately, wait the configured interval after that synchronization finishes, and only then start the next synchronization. It shall never overlap synchronization cycles. This requirement supersedes only the fixed five-second interval in RF-17 and the corresponding interval exclusion of the local bidirectional synchronization specification. This is required to apply the selected frequency predictably even when synchronization takes longer than the interval.

RF-30. While `gosync watch` is running, the system shall continue using the configuration loaded at startup even if `config.json` changes; a new `gosync watch` execution shall load the current complete configuration. This is required to keep each execution internally consistent while applying later changes predictably.

RF-31. If the standard configuration location is equal to or contained within either selected synchronization root, `gosync watch` shall reject synchronization and report an explanatory error. This is required to prevent configuration from being copied, deleted, or treated as synchronized user content.

RF-32. When configuration starts, succeeds, is cancelled, or fails, the system shall record the event according to the log system specification without recording interactive responses or JSON contents. This is required to preserve operational diagnostics without exposing configuration data.

## Out of Scope

- Implementing BACKUP behavior, archive creation, compression, retention, or restoration.
- Saving `BACKUP` as an active synchronization mode before backup mode is implemented.
- Per-directory-pair configurations or named configuration profiles.
- Non-interactive configuration through command arguments, flags, environment variables, or standard input formats other than the defined questions.
- Reloading configuration while `gosync watch` is running.
- Applying default synchronization settings when the configuration file is absent.
- Configuring synchronization roots; roots remain arguments of `gosync watch`.
- Configuring log destinations, log format, log rotation, log retention, or severity filtering.
- Configuring FTP, SFTP, cloud, or other remote synchronization.
- Graphical interfaces, web services, or remote configuration storage.

## Completion Criteria

- `gosync configure` accepts no additional arguments, repeatedly requests invalid values, presents unavailable BACKUP accurately, and saves only after displaying a valid summary and receiving explicit confirmation.
- The configured interval is a whole number from 1 through 86,400 seconds.
- Bidirectional mode retains the existing two-way synchronization contract except for the recheck interval explicitly superseded by this specification.
- Unidirectional mode requires an existing source, creates a missing destination, makes the destination an exact source replica, preserves filesystem safety behavior, and never modifies the source from destination differences.
- `config.json` uses the exact versioned schema, location, property names, types, ranges, and mode values defined by RF-18 and rejects every invalid structure listed in RF-19.
- A successful configuration replacement leaves the new complete JSON; a failed or interrupted replacement leaves either the previous complete JSON or the new complete JSON, never a partial document.
- Cancellation, absent input, invalid storage types, and all defined configuration or I/O failures produce explanatory behavior and preserve the last complete valid configuration.
- Only one configuration process can be active, and competing processes cannot change the JSON.
- A missing configuration starts the interactive flow only after valid `watch` arguments and synchronization proceeds only after that configuration is confirmed and saved.
- Invalid existing configuration blocks `watch` but can be replaced through a successfully completed `configure` flow.
- `watch` runs immediately, waits the configured interval after each completed cycle, never overlaps cycles, retains startup settings for that execution, and loads current settings on its next execution.
- A configuration location equal to or contained within either synchronization root blocks synchronization.
- Configuration lifecycle events are logged without interactive responses or JSON contents.
- Automated tests verify every functional requirement and its validation, cancellation, atomic replacement, concurrency, persistence, I/O failure, scheduling, and synchronization-mode behavior.
