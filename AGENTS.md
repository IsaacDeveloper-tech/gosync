# GoSync

## Project

A command-line application for synchronizing one directory with another in real time. The target folder can be local or synchronized via FTP or SFTP, and includes an optional compression feature for use as a backup.

## Commands

- Build: `go build`
- Run: `go run .`
- Test: `go test`

## Conventions

- Methods, functions, and variables must be descriptive (indicating what they store and their purpose) and written in English.

- Methods and functions must be testable, and assertive programming must be used (checking for potential errors first).

## Rules

- Read `docs/constitution.md` and the active spec in `specs/` before touching the code.

- Do not modify files within `specs/` unless explicitly requested.

- Before completing a task, we write a 2- or 3-line summary of what has been done in the `docs/implementations.md` file.

## Upon completing any task

- Run the `go test` command and confirm that everything passes correctly.