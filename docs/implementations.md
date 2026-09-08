# Implementations

Added the GoSync project constitution with seven enforceable engineering principles.
The principles define stack, application boundaries, error handling, quality, tests, and modification limits.

Added the local bidirectional synchronization specification with EARS functional requirements.
The specification defines conflict, deletion, missing-directory, and unsupported-entry behavior and its boundaries.

Revised the local synchronization specification to resolve ambiguity in deletions, conflicts, and filesystem failures.
It now defines confirmed state, recovery, retry handling, path validation, and automated test completion criteria.

Added the active specification's implementation plan in English, including module responsibilities and data models.
The plan maps each module and test scenario to the local synchronization functional requirements.

Added the active specification's ordered implementation task list in English.
Each task is estimated below 30 minutes and maps to functional requirements with a verifiable completion condition.

Added the `gosync watch <directory-a> <directory-b>` command contract to the active specification.
The plan and tasks now cover argument validation, immediate synchronization, and fixed five-second rechecks.

Completed task 1 by defining synchronization entry, comparison, action, and result domain types.
Added tests that verify the contracts represent file, directory, copy, deletion, success, and failure outcomes.

Completed task 2 by adding root path normalization and validation for equal or nested synchronization roots.
Added tests for normalized paths, identical roots, nested roots, sibling roots, and empty root arguments.

Completed task 3 by adding a read-only preflight scan for both synchronization roots.
Added tests for regular entries, missing roots, symbolic links, and unsupported filesystem entry modes.

Completed task 4 by creating missing synchronization roots while preserving existing directory contents.
Added tests for one or two missing roots and errors when a selected root is an existing file.

Completed task 5 by adding persistent confirmed-state storage keyed by the selected root pair.
Added tests for state loading, root-pair isolation, missing state, and preservation after incomplete synchronization.

Completed task 6 by adding recursive directory inventories with file content digests and modification times.
Added tests for inventory entries and comparisons that ignore metadata while detecting content and structure differences.

Completed task 7 by classifying one-sided entries as additions or deletions using confirmed state.
Added tests for both directions, unchanged entries, absent entries, and deterministic action order.

Completed task 8 by comparing both inventories when confirmed state is unavailable.
Added tests for matching directories, authoritative-side selection, existing state, and decision errors.

Completed task 9 by resolving differing file contents using modification time or an explicit user choice on ties.
Added tests for both newer-file directions, equal-time conflicts, matching content, and decision errors.

Completed task 10 by listing directory contents and requiring an explicit side selection for file-directory conflicts.
Added tests for both conflict directions, sorted listings, missing choices, and decision errors.

Completed task 11 by generating complete synchronization plans from additions, deletions, and conflicts.
Added tests confirming ordered actions, required user decisions, directory contents, and rejection of partial plans.
