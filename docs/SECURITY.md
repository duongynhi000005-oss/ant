# Security

To report a vulnerability, email the maintainers privately, and do not open a public issue.

This page records a review of ant on 2026-10-31, covering SQL construction, logging, and secrets. It found no open issue of high severity. It is a review by the project's own tooling and authors, not an outside audit.

## SQL injection

| Path | Protection |
| --- | --- |
| Values in queries | Always bound as parameters, in every statement ant writes |
| Column and table names | Come from struct metadata, never from input, and are quoted |
| `OrderBy`, `Where`, joins | Take field probes, so a request cannot name a column |
| Query comments and hints | Comment delimiters are removed, so they cannot close the comment. Covered by a test |
| JSON paths in `postgres.JSONText` and `JSONSet` | Bound as a parameter, and each part is quoted, so a key cannot add to the path. Covered by a test |
| Migration scripts | Identifiers are quoted, and text is escaped. A guarded script picks a dollar quote its content does not hold |

These take raw SQL by design, and must never be built from input. Pass values as arguments:

- `ant.Raw`, `ant.RawFile`, `ant.FromRaw`, `ant.FromFunc`
- `ant.SQL`, `ant.SQLCond`
- `b.Check`, `.Default`, `.Generated`, `.Type`, `.Where` on indexes: these are written by developers, into migrations
- `q.Hint`, `q.Tag`, and `ant.Func`'s name

`antvet` does not check these, so reviews should. A helper that takes a column or sort order from a request should map it through a fixed list.

## Logs and errors

- Values stay out of logs unless `ant.LogValues` is set. A test proves this.
- `.Sensitive()` keeps a column out even then, on inserts, updates, `Set`, and conditions. A test proves this. `ant.SQL` arguments are not covered, since ant cannot tell which column they belong to.
- Query errors hold the SQL with its placeholders, for `ant.FailedSQL`. `WithDetailedErrors` adds it to the message. Leave that off where errors reach users.
- **Known limit:** a database error's message can quote a value, as in `invalid input syntax for type integer: "abc"`, and `WithLogger` logs that message. Do not return `err.Error()` to users. The examples map errors to fixed messages.
- `Query.Debug` shows values on purpose, and is meant for a desk, not production.
- Migration failures show the database's detail, such as a duplicated value, to the operator, and store the message in the history table.

## Secrets

- `secure.Value` uses AES-256-GCM with a random nonce. It prints and marshals as `[encrypted]`.
- **Known limit:** a ciphertext is not bound to its row. See [encrypted fields](recipes/encryption.md#limits).
- ant does not log connection strings. **Known limit:** an error from `Open` comes from the driver, and pgx masks passwords on a best-effort basis, so a badly malformed connection string can still reach the message. Treat that error as sensitive.

## Tenancy

A global filter is a convenience, not a security boundary, because `ant.Raw` and `IgnoreFilters` bypass it. Filters that fail closed are described in the [tenancy recipe](recipes/tenancy.md).

## Migrations

- `reset` needs the database's name, typed or passed with `--confirm`, and refuses otherwise.
- A checksum stops an applied file changing unnoticed. It detects mistakes, not attackers: someone who can edit the history table can change it.
- Migration files are code. Treat access to them, and to the embedded binary, like access to the database.

## Dependencies

The root package uses the standard library only. Drivers, test helpers, and OpenTelemetry stay in sub-packages or modules of their own.

`govulncheck` on 2026-11-01 found nothing that ant's code reaches. One advisory stays open in a required module: GO-2026-5932 in `golang.org/x/crypto`, which has no fix yet, and which no code here calls.
