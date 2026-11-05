package example

import (
	"context"

	"github.com/skudra-go/ant"

	"models"
)

type User struct {
	ID   int64
	Age  int
	Name string
}

type Post struct {
	ID     string
	UserID int64
}

// Member is User under another name, as a test package might alias a fixture.
type Member = User

// Doc's key is a slug filled in Go.
type Doc struct { // want Doc:"key Slug"
	ID   int64
	Slug string
}

func (Doc) Configure(b *ant.ModelBuilder[Doc]) {
	ant.GenerateKey(b, &D.Slug, func() string { return "x" })
}

// Pair has a composite key, which Find cannot take.
type Pair struct { // want Pair:"key "
	ID int64
	A  int64
	B  int64
}

func (*Pair) Configure(b *ant.ModelBuilder[Pair]) {
	b.CompositeKey(ant.F(&Pa.A), ant.F(&Pa.B))
}

var (
	U     = ant.Fields[User]()   // want U:"probe of example.User"
	P     = ant.Fields[Post]()   // want P:"probe of example.Post"
	Alias = U                    // want Alias:"probe of example.User"
	M     = ant.Fields[Member]() // want M:"probe of example.User"
	D     = ant.Fields[Doc]()    // want D:"probe of example.Doc"
	Pa    = ant.Fields[Pair]()   // want Pa:"probe of example.Pair"
)

func good(
	ctx context.Context,
	users *ant.Table[User],
	members *ant.Table[Member],
	posts *ant.Table[Post],
	countries *ant.Table[models.Country],
	docs *ant.Table[Doc],
	pairs *ant.Table[Pair],
) {
	users.Where(ant.Eq(&U.Age, 18), ant.Gt(&Alias.Age, 1)).All(ctx)
	users.Where(ant.EqCol(&U.ID, &P.UserID))          // two columns may refer to an outer query
	posts.Where(ant.Eq(&U.Name, "x")).Join().All(ctx) // a condition can come before its join
	local := ant.Fields[User]()
	users.Where(ant.Gt(&local.Age, 1)).All(ctx)
	users.Find(ctx, int64(1))
	users.Find(ctx, 1)
	posts.Find(ctx, "abc")

	// An alias names the same model as what it stands for.
	members.Where(ant.Eq(&U.Name, "x")).All(ctx)
	users.Where(ant.Eq(&M.Name, "x")).Count(ctx)
	members.Find(ctx, int64(1))

	// A subquery may name its outer query's model, inline or built apart.
	users.Where(ant.Exists(posts.Where(ant.Gt(&U.Age, 18)))).All(ctx)
	inner := posts.Where(ant.Gt(&U.Age, 18))
	users.Where(ant.Exists(inner)).All(ctx)
	_ = posts.Where(ant.Gt(&U.Age, 18)).OrderBy(&P.ID)

	// Keys that Configure sets, in this package and another.
	countries.Find(ctx, "LV")
	docs.Find(ctx, "a-slug")
	pairs.Find(ctx, "anything")
}

func bad(
	ctx context.Context,
	users *ant.Table[User],
	members *ant.Table[Member],
	posts *ant.Table[Post],
	countries *ant.Table[models.Country],
	docs *ant.Table[Doc],
) {
	x := 5
	ant.Gt(&x, 3) // want `ant.Gt needs a field of a probe`
	user := &User{}
	ant.Eq(&user.Age, 18) // want `not of an ordinary variable`

	posts.Where(ant.Gt(&U.Age, 18)).All(ctx)                                    // want `condition on User in a query of Post`
	posts.Where(ant.Gt(&U.Age, 18)).OrderBy(&P.ID).Count(ctx)                   // want `condition on User in a query of Post`
	users.Where(ant.Exists(posts.Where(ant.Eq(&models.Co.Code, "x")))).All(ctx) // want `condition on Country in a query of Post`
	users.Find(ctx, "1")                                                        // want `Find key is string, and User has a key of type int64`
	posts.Find(ctx, 1)                                                          // want `Find key is int, and Post has a key of type string`
	members.Find(ctx, "1")                                                      // want `Find key is string, and User has a key of type int64`
	countries.Find(ctx, 1)                                                      // want `Find key is int, and Country has a key of type string`
	docs.Find(ctx, 7)                                                           // want `Find key is int, and Doc has a key of type string`
}

// Code that is wrong on purpose, such as a test of the error, says so.
func onPurpose(ctx context.Context, users *ant.Table[User]) {
	x := 5
	ant.Gt(&x, 3) //antvet:ignore the error is what this tests

	//antvet:ignore so is this one
	users.Find(ctx, "1")
}
