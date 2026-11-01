// Package compare measures ant against GORM, Bun, and plain database/sql on one
// PostgreSQL, doing the same work. It is a module of its own, so that ant does not
// depend on what it is compared with. Plain database/sql stands in for sqlc, whose
// generated code is exactly that: a query string and a Scan.
//
//	make db-up DATABASES=postgres
//	cd tools/compare && go test -run '^$' -bench . -benchmem -benchtime 2s
package compare

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/skudra-go/ant"
	antpg "github.com/skudra-go/ant/dialect/postgres"
)

// User is the row every library reads.
type User struct {
	ID        int64 `gorm:"primaryKey" bun:",pk,autoincrement"`
	CreatedAt time.Time
	UpdatedAt time.Time
	Name      string
	Email     string
	Age       int
	Bio       *string

	bun.BaseModel `bun:"table:users" gorm:"-" ant:"-"`
}

// AppDB is ant's context.
type AppDB struct {
	ant.Context
	Users ant.Table[User]
}

var U = ant.Fields[User]()

const ddl = `
	DROP SCHEMA IF EXISTS ant_compare CASCADE;
	CREATE SCHEMA ant_compare;
	CREATE TABLE ant_compare.users (id BIGSERIAL PRIMARY KEY, created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL,
		name TEXT NOT NULL, email TEXT NOT NULL, age INT NOT NULL, bio TEXT);
	INSERT INTO ant_compare.users (created_at, updated_at, name, email, age)
	SELECT now(), now(), 'user ' || n, 'u' || n || '@example.com', n % 80 FROM generate_series(1, 1000) n`

type libraries struct {
	plain *sql.DB
	ant   *AppDB
	gorm  *gorm.DB
	bun   *bun.DB
}

func setup(b *testing.B) *libraries {
	b.Helper()

	dsn := os.Getenv("ANT_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@127.0.0.1:55432/ant_test?sslmode=disable"
	}

	dsn += "&search_path=ant_compare"
	ctx := context.Background()

	plain, err := sql.Open("pgx", dsn)
	if err != nil {
		b.Skip(err)
	}

	if err := plain.PingContext(ctx); err != nil {
		b.Skip("no database: ", err)
	}

	if _, err := plain.ExecContext(ctx, ddl); err != nil {
		b.Fatal(err)
	}

	db, err := ant.Open[AppDB](ctx, antpg.Dialect, dsn, ant.WithTracking(false))
	if err != nil {
		b.Fatal(err)
	}

	orm, err := gorm.Open(
		postgres.New(postgres.Config{Conn: plain}),
		&gorm.Config{Logger: logger.Discard, SkipDefaultTransaction: true},
	)
	if err != nil {
		b.Fatal(err)
	}

	b.Cleanup(func() { db.Close(); plain.Close() })

	return &libraries{plain, db, orm, bun.NewDB(plain, pgdialect.New())}
}

func BenchmarkRead1000(b *testing.B) {
	l := setup(b)
	ctx := context.Background()

	b.Run("sql", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			rows, err := l.plain.QueryContext(
				ctx,
				"SELECT id, created_at, updated_at, name, email, age, bio FROM users",
			)
			if err != nil {
				b.Fatal(err)
			}

			out := make([]*User, 0, 1000)

			for rows.Next() {
				u := new(User)
				if err := rows.Scan(
					&u.ID,
					&u.CreatedAt,
					&u.UpdatedAt,
					&u.Name,
					&u.Email,
					&u.Age,
					&u.Bio,
				); err != nil {
					b.Fatal(err)
				}

				out = append(out, u)
			}

			rows.Close()

			if len(out) != 1000 {
				b.Fatal(len(out))
			}
		}
	})

	b.Run("ant", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			out, err := l.ant.Users.All(ctx)
			if err != nil || len(out) != 1000 {
				b.Fatal(len(out), err)
			}
		}
	})

	b.Run("gorm", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var out []*User
			if err := l.gorm.WithContext(ctx).Find(&out).Error; err != nil || len(out) != 1000 {
				b.Fatal(len(out), err)
			}
		}
	})

	b.Run("bun", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var out []*User
			if err := l.bun.NewSelect().Model(&out).Scan(ctx); err != nil || len(out) != 1000 {
				b.Fatal(len(out), err)
			}
		}
	})
}

func BenchmarkFindOne(b *testing.B) {
	l := setup(b)
	ctx := context.Background()

	b.Run("sql", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			u := new(User)

			err := l.plain.QueryRowContext(ctx, "SELECT id, created_at, updated_at, name, email, age, bio FROM users WHERE id = $1", 500).
				Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt, &u.Name, &u.Email, &u.Age, &u.Bio)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ant", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if _, err := l.ant.Users.Find(ctx, int64(500)); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var u User
			if err := l.gorm.WithContext(ctx).First(&u, 500).Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("bun", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var u User
			if err := l.bun.NewSelect().Model(&u).Where("id = ?", 500).Scan(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkFilteredQuery(b *testing.B) {
	l := setup(b)
	ctx := context.Background()

	b.Run("sql", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			rows, err := l.plain.QueryContext(
				ctx,
				"SELECT id, created_at, updated_at, name, email, age, bio FROM users WHERE age > $1 AND name LIKE $2 ORDER BY name LIMIT 10",
				30,
				"user 1%",
			)
			if err != nil {
				b.Fatal(err)
			}

			for rows.Next() {
				u := new(User)
				if err := rows.Scan(
					&u.ID,
					&u.CreatedAt,
					&u.UpdatedAt,
					&u.Name,
					&u.Email,
					&u.Age,
					&u.Bio,
				); err != nil {
					b.Fatal(err)
				}
			}

			rows.Close()
		}
	})

	b.Run("ant", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			_, err := l.ant.Users.Where(ant.Gt(&U.Age, 30), ant.Like(&U.Name, "user 1%")).
				OrderBy(&U.Name).
				Limit(10).
				All(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var out []*User
			if err := l.gorm.WithContext(ctx).
				Where("age > ? AND name LIKE ?", 30, "user 1%").
				Order("name").
				Limit(10).
				Find(&out).
				Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("bun", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			var out []*User
			if err := l.bun.NewSelect().
				Model(&out).
				Where("age > ?", 30).
				Where("name LIKE ?", "user 1%").
				Order("name").
				Limit(10).
				Scan(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkInsert(b *testing.B) {
	l := setup(b)
	ctx := context.Background()
	n := 0
	fresh := func() *User {
		n++

		return &User{Name: "new", Email: fmt.Sprint("new", n, "@example.com"), Age: 30}
	}

	b.Run("sql", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			u := fresh()
			now := time.Now()

			err := l.plain.QueryRowContext(
				ctx,
				"INSERT INTO users (created_at, updated_at, name, email, age, bio) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id",
				now,
				now,
				u.Name,
				u.Email,
				u.Age,
				u.Bio,
			).Scan(&u.ID)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ant", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if err := l.ant.Users.Insert(ctx, fresh()); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if err := l.gorm.WithContext(ctx).Create(fresh()).Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("bun", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			if _, err := l.bun.NewInsert().Model(fresh()).Exec(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkUpdateOneColumn(b *testing.B) {
	l := setup(b)
	ctx := context.Background()
	age := 0

	b.Run("sql", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			age++
			if _, err := l.plain.ExecContext(
				ctx,
				"UPDATE users SET age = $1, updated_at = $2 WHERE id = $3",
				age%90,
				time.Now(),
				500,
			); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ant", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			age++
			if _, err := l.ant.Users.Where(ant.Eq(&U.ID, 500)).
				Update(ctx, ant.Set(&U.Age, age%90)); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("gorm", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			age++
			if err := l.gorm.WithContext(ctx).
				Model(&User{}).
				Where("id = ?", 500).
				Update("age", age%90).
				Error; err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("bun", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			age++
			if _, err := l.bun.NewUpdate().
				Model((*User)(nil)).
				Set("age = ?", age%90).
				Set("updated_at = ?", time.Now()).
				Where("id = ?", 500).
				Exec(ctx); err != nil {
				b.Fatal(err)
			}
		}
	})
}
