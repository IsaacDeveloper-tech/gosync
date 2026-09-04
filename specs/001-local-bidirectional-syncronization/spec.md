# Local Bidirectional Synchronization Specification

## Purpose

Keep two local directories equivalent so that each contains the same structure, names, and file contents.

## Functional Requirements

RF-01. When synchronization completes successfully, the system shall retain a confirmed record of the synchronized items. This is required to distinguish later additions from deletions.

RF-02. When synchronization does not complete successfully, the system shall preserve the previous confirmed record. This is required to prevent incomplete work from being treated as a completed synchronization.

RF-03. When no confirmed record is available after an interrupted synchronization, the system shall compare both selected directories and determine whether their structures, names, and file contents are identical. This is required to establish whether recovery is needed.

RF-04. When the selected directories differ and no confirmed record is available, the system shall ask the user which directory shall synchronize over the other. This is required to avoid choosing a version without sufficient history.

RF-05. When a file or directory exists in only one selected directory and is absent from the confirmed record, the system shall create an equivalent item in the other directory. This is required to replicate additions.

RF-06. When an item present in the confirmed record is absent from one selected directory and present in the other, the system shall delete the remaining equivalent item. This is required to replicate deletions.

RF-07. When equivalent files have different contents, the system shall retain the version with the most recent modification time in both directories. This is required to resolve conflicts predictably.

RF-08. When equivalent files have different contents and the same modification time, the system shall stop and ask the user which version to retain. This is required because neither version is objectively more recent.

RF-09. When the same path is a file in one directory and a directory in the other, the system shall stop, list the contents of the directory to the user, and ask which item shall prevail. This is required to prevent unintentional loss of directory contents.

RF-10. When a selected root directory does not exist, the system shall create it and synchronize the contents of the other selected directory into it. This is required to support a new local synchronization location.

RF-11. When the selected directories are identical or one is contained within the other, the system shall reject synchronization and report an explanatory error. This is required to prevent self-synchronization and recursive copying.

RF-12. When the system detects a symbolic link or special filesystem entry in either selected directory before synchronization begins, the system shall stop and report an explanatory error without modifying either directory. This is required to avoid unsafe or undefined handling of unsupported entries.

RF-13. When a filesystem error other than a locked file prevents synchronization, the system shall stop, report the cause to the user, and preserve the previous confirmed record. This is required to prevent an incomplete synchronization from being confirmed.

RF-14. When a file is locked by another program, the system shall notify the user, continue retrying that file automatically until it becomes available, and notify the user when it has synchronized successfully. This is required to complete synchronization without requiring the user to restart it.

RF-15. When determining whether selected directories are synchronized, the system shall compare their structure, names, and file contents, and shall not require matching permissions or modification times. This is required to define synchronization independently of non-content metadata.

## Out of Scope

- FTP, SFTP, cloud, or other remote synchronization.
- Compression and backup archive creation.
- Graphical user interfaces and web services.
- Replicating symbolic links or special files.
- Version history and restoration of prior file versions.

## Completion Criteria

- Automated tests cover every functional requirement and its defined error behavior.
- A successful synchronization leaves both selected directories with the same structure, names, and file contents.
- The confirmed record is updated only after a successful synchronization.
- Additions and deletions are distinguished using the confirmed record and replicated correctly.
- Tied file conflicts and file-directory conflicts require a user decision.
- Missing root directories are created, overlapping roots are rejected, and unsupported entries cause no changes.
- Filesystem errors preserve the confirmed record; locked files are retried and reported when synchronized.
