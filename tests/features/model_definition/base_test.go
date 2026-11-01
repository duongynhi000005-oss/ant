package model_definition

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/skudra-go/ant"
	"github.com/skudra-go/ant/tests/features/fixture"
)

// Embedding ant.Base adds ID, CreatedAt, and UpdatedAt columns.
type Article struct {
	ant.Base
	Title string
}

type BlogDB struct {
	ant.Context
	Articles ant.Table[Article]
}

const articlesDDL = `CREATE TABLE articles (
	id BIGSERIAL PRIMARY KEY,
	created_at TIMESTAMPTZ NOT NULL,
	updated_at TIMESTAMPTZ NOT NULL,
	title TEXT NOT NULL)`

func TestBase(t *testing.T) {
	fixture.RunWith(t, articlesDDL, func(t *testing.T, db *BlogDB) {
		t.Run("fields are promoted", func(t *testing.T) {
			article := &Article{Title: "Hello"}
			require.NoError(t, db.Articles.Insert(t.Context(), article))
			require.NotZero(t, article.ID)
		})

		t.Run("timestamps are set for you", func(t *testing.T) {
			article := &Article{Title: "Dates"}
			before := time.Now().Add(-time.Second)

			require.NoError(t, db.Articles.Insert(t.Context(), article))
			require.True(t, article.CreatedAt.After(before))
			require.Equal(t, article.CreatedAt, article.UpdatedAt)
		})

		t.Run("a time you set is kept", func(t *testing.T) {
			at := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
			article := &Article{Base: ant.Base{CreatedAt: at}, Title: "Imported"}
			require.NoError(t, db.Articles.Insert(t.Context(), article))
			require.True(t, at.Equal(article.CreatedAt))
		})
	})
}
