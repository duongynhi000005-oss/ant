// Package migrate keeps a database in step with its models.
//
// Structs are the source of truth. A command compares them with a stored
// snapshot and writes the difference as plain SQL, which you review and can edit.
// Another applies what is pending. Generating needs no database, and every file is
// ordinary SQL, so any tool can read them and you can stop using ant's runner at
// any time.
//
// Go cannot load structs from another module, so the CLI is a small program in
// yours:
//
//	// cmd/migrate/main.go
//	func main() {
//		migrate.Main[models.AppDB](postgres.Dialect)
//	}
//
// Run it with go run ./cmd/migrate, and see its help for the commands. To apply
// migrations when a program starts, embed them and call Up:
//
//	//go:embed all:versions all:code
//	var Files embed.FS
//
//	err := migrate.Up(ctx, db, migrate.Config{FS: migrations.Files})
//
// Everything has a default, and everything can be replaced: see Config for names,
// folders, and the history table, and Source for storing migrations somewhere
// other than files.
package migrate
