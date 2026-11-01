# Compatibility

## Go

ant needs Go 1.27 or later, because its API uses generic methods. Each release supports the two newest Go versions, as Go itself does.

## Databases

| Database | Status |
| --- | --- |
| PostgreSQL 13 to 18 | Supported: the whole suite passes on each, as of 2026-11-01 |
| MySQL 8.4, MariaDB 11.8, SQLite (modernc 1.61), SQL Server 2022 | Queries and writes: the conformance suite passes. No migrations |

`MERGE` needs PostgreSQL 15. Everything else works from 13. Version 12 and older are out of support upstream, and untested here. The comments in each dialect name the oldest versions expected to work, from their documentation: only the versions above have been run. See [databases](databases.md) for what each one lacks.

ant runs on the `pgx` driver through `database/sql`. Another PostgreSQL driver may work, but `pgxext` needs pgx.

## Versions

ant has not reached 1.0, so anything can change between releases. Each release that changes behaviour says so in the [upgrade guide](upgrading.md).

From 1.0, ant follows semantic versioning:

- **Patch** releases fix bugs.
- **Minor** releases add to the API, and change nothing that exists.
- **Major** releases can break things, and the upgrade guide says how to move.

What counts as the API:

- Every exported name in the [public packages](api.txt), which a test keeps a record of.
- The behaviour the documentation and `tests/features` describe.
- The migration file format, and the history table.

What does not:

- Anything under `internal`.
- The exact SQL ant writes, which can change as long as it means the same.
- The text of error messages. Match errors with `errors.Is`.
- The `dialect` and `qb` packages, until a dialect outside this module exists to settle them.

## Files that outlive a release

Migrations and snapshots stay readable. A snapshot records its format, and a newer format is refused with a clear error, not misread. The history table only gains columns.
