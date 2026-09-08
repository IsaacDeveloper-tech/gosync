# Log System Specification

## Purpose

Provide a persistent and immediately visible record of GoSync activity and failures so users can understand synchronization outcomes and diagnose problems without exposing protected information.

## Functional Requirements

RF-01. When GoSync starts or finishes a synchronization operation, the system shall record the event and its outcome. This is required to identify the boundaries and result of each operation.

RF-02. When GoSync synchronizes a change, detects a conflict, retries an operation, raises a warning, or encounters an error, the system shall record the event and its relevant context. This is required to explain what happened during synchronization and why user attention may be needed.

RF-03. While both log destinations are available, the system shall write every log entry to the console and to a persistent log file located in the operating system's standard per-user application data location for GoSync. This is required to provide immediate visibility, retain a diagnostic history, and keep logs independent from the command's execution directory.

RF-04. The system shall classify every log entry as `DEBUG`, `INFO`, `WARN`, or `ERROR` according to its severity. This is required to distinguish diagnostic details, normal activity, recoverable concerns, and failures.

RF-05. The system shall present every log entry as structured text containing a date and time, severity level, event, and human-readable message. This is required to make entries understandable and consistently searchable.

RF-06. When an operation fails, the system shall record a sanitized cause without replacing or suppressing the existing user-facing error behavior. The sanitized cause shall preserve the error type and useful context while replacing credentials, authentication secrets, cryptographic keys, and file contents. This is required to preserve both diagnostics and the established failure contract without exposing protected information.

RF-07. If the persistent log file cannot be created, opened, written, rotated, or retained while the console remains available, the system shall report the logging failure through the console, disable persistent logging for the remainder of the current execution, and allow synchronization to continue. This is required to preserve the primary operation without exceeding storage limits or repeatedly attempting an unavailable destination.

RF-08. When the active log file reaches 10 MB, the system shall rotate the log history and continue recording in an active log file. This is required to prevent unbounded storage growth.

RF-09. When log rotation would produce more than five log files in total, the system shall remove the oldest retained log file. This is required to limit the approximate maximum log storage to 50 MB while preserving the most recent history.

RF-10. The system shall never record credentials, authentication secrets, cryptographic keys, or file contents in any log entry, including when protected information appears within an error, path, or name. This is required to prevent diagnostic records from exposing protected data.

RF-11. When an event concerns a filesystem location, the system may record its path or name after replacing any credentials, authentication secrets, cryptographic keys, or file contents contained within it. This is required to identify affected items without disclosing protected information.

RF-12. If the standard persistent log location is equal to or contained within either selected synchronization root, the system shall warn the user through the console, disable persistent logging for the current execution, and allow synchronization to continue. This is required to prevent log writes from triggering additional synchronization activity.

RF-13. If console output becomes unavailable while persistent logging remains available, the system shall record the console failure in the persistent log and allow synchronization to continue. This is required to preserve diagnostics while at least one log destination remains operational.

RF-14. If both console output and persistent logging are unavailable, the system shall stop synchronization with a logging error. This is required to avoid continuing without any operational destination for activity and failure records.

## Out of Scope

- Remote log collection, forwarding, or centralized aggregation.
- A graphical or web-based log viewer.
- Metrics, distributed tracing, alert delivery, or audit certification.
- Recording file contents, credentials, authentication secrets, or cryptographic keys.
- Anonymizing or hiding filesystem paths and names that do not contain protected information.
- Log compression, encryption, or archival backup.
- User configuration of log destinations, format, rotation size, retention count, or severity filtering.

## Completion Criteria

- Synchronization starts, outcomes, synchronized changes, conflicts, retries, warnings, and errors produce structured entries with a date and time, severity, event, and message.
- Entries use the `DEBUG`, `INFO`, `WARN`, and `ERROR` severity classifications appropriately.
- Normal operation writes equivalent diagnostic information to the console and a persistent log file in the operating system's standard per-user application data location for GoSync.
- A persistent log location within either synchronization root is reported through the console and disabled for the current execution.
- Failure of any persistent logging activity is visible in the console, disables persistent logging for the current execution, and does not stop synchronization while the console remains available.
- Failure of console output is recorded in the persistent log and does not stop synchronization while persistent logging remains available.
- Synchronization stops with a logging error when neither console output nor persistent logging is available.
- Log files rotate at 10 MB and no more than five log files are retained in total.
- Logs contain no credentials, authentication secrets, cryptographic keys, or file contents, including within recorded errors, paths, and names.
- Automated tests verify every functional requirement, including logging failures, rotation boundaries, retention, and protected-data exclusions.
