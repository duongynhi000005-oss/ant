# Upgrading

ant has not been released, so there is nothing to upgrade from yet. This page will list each release's breaking changes, and what to do about them, newest first.

## How to upgrade

1. Read the section for each release you cross.
2. Update the module: `go get github.com/skudra-go/ant@latest`.
3. Run `go build ./...`. A renamed or changed API fails here, which is the point of a typed one.
4. Run your tests against a real database.
5. Run `go run ./cmd/migrate check`. A release can change how a model maps to a schema, and this shows it. If it reports changes, read them, and generate a migration.
6. Run `go run ./cmd/migrate drift` against a staging database.

## Migrating between major versions

Migrations are plain SQL, so they never depend on ant's version. Regenerating a snapshot is safe at any time: `go run ./cmd/migrate snapshot` records the models as they are now.

## Leaving ant

See [leaving ant's tooling](migrations.md#leaving-ants-tooling).
