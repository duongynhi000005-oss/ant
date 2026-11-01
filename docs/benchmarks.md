# Benchmarks

These compare ant with GORM v1.31, Bun v1.3, and plain `database/sql`, doing the same work on the same PostgreSQL 17 in Docker. Plain `database/sql` stands in for sqlc, whose generated code is a query string and a `Scan`. Run them yourself:

```sh
make db-up DATABASES=postgres
cd tools/compare && go test -run '^$' -bench . -benchmem -benchtime 2s
```

These results came from one run on one laptop in WSL, on 2026-10-31. Read them as a shape, not as a promise. Differences of a few percent are noise: a shorter run of the same code once put ant's single-row read at 504 µs and Bun's update at 5.8 ms, and neither survived a longer one. Allocations are stable, and times are not.

| Benchmark | `database/sql` | ant | GORM | Bun |
| --- | --- | --- | --- | --- |
| **Read 1000 rows** | 1.68 ms | 1.56 ms | 3.03 ms | 2.16 ms |
| Memory | 223 kB | 254 kB | 369 kB | 270 kB |
| Allocations | 7,771 | 6,833 | 14,808 | 8,790 |
| **Find one row** | 370 µs | 384 µs | 390 µs | 375 µs |
| Memory | 1.5 kB | 6.0 kB | 5.8 kB | 6.4 kB |
| Allocations | 36 | 93 | 89 | 41 |
| **Filter, sort, limit** | 541 µs | 537 µs | 544 µs | 528 µs |
| Memory | 3.5 kB | 9.9 kB | 9.7 kB | 9.2 kB |
| Allocations | 98 | 178 | 219 | 124 |
| **Insert one row** | 2.80 ms | 2.90 ms | 2.91 ms | 3.47 ms |
| Memory | 1.3 kB | 3.2 kB | 7.1 kB | 6.6 kB |
| Allocations | 22 | 76 | 76 | 39 |
| **Update one column** | 2.86 ms | 2.96 ms | 2.94 ms | 2.96 ms |
| Memory | 0.4 kB | 2.9 kB | 5.7 kB | 5.2 kB |
| Allocations | 8 | 47 | 58 | 14 |

**Rows are where a mapper's cost shows.** ant reads a row with a compiled list of field pointers, so reading 1000 rows costs about what hand-written `Scan` calls do, and half what GORM does.

**A single row is mostly a network round trip.** Every library spends nearly all its time waiting, so the four are within 5% of each other, and one fewer query matters far more. ant allocates more than Bun or plain SQL here. That shows up in a garbage collector's work under load, not in this table.

**Writes wait for the disk.** The insert and update figures are dominated by the commit.

## Building queries

`tests/benchmarks` measures what ant adds with no database involved:

| Operation | Time | Allocations |
| --- | --- | --- |
| Resolving a field probe | 20 ns | 0 |
| Building and rendering a query with two conditions, an order, and a limit | 7.3 µs | 78 |

Budgets in `tests/benchmarks/allocs_test.go` fail a test if these grow by accident.

Rendering takes a few microseconds, against a few hundred for the round trip, which is why ant does not cache rendered SQL: a cache key would cost about what it saves, and pgx already caches prepared statements per connection.
