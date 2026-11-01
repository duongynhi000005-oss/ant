package example

import (
	"context"

	"github.com/skudra-go/ant"
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

var (
	U     = ant.Fields[User]() // want U:"probe of example.User"
	P     = ant.Fields[Post]() // want P:"probe of example.Post"
	Alias = U                  // want Alias:"probe of example.User"
)

func good(ctx context.Context, users *ant.Table[User], posts *ant.Table[Post]) {
	users.Where(ant.Eq(&U.Age, 18), ant.Gt(&Alias.Age, 1))
	users.Where(ant.EqCol(&U.ID, &P.UserID)) // two columns may refer to an outer query
	posts.Where(ant.Eq(&U.Name, "x")).Join() // a condition can come before its join
	local := ant.Fields[User]()
	users.Where(ant.Gt(&local.Age, 1))
	users.Find(ctx, int64(1))
	users.Find(ctx, 1)
	posts.Find(ctx, "abc")
}

func bad(ctx context.Context, users *ant.Table[User], posts *ant.Table[Post]) {
	x := 5
	ant.Gt(&x, 3) // want `ant.Gt needs a field of a probe`
	user := &User{}
	ant.Eq(&user.Age, 18) // want `not of an ordinary variable`

	posts.Where(ant.Gt(&U.Age, 18)) // want `condition on User in a query of Post`
	users.Find(ctx, "1")            // want `Find key is string, and User has a key of type int64`
	posts.Find(ctx, 1)              // want `Find key is int, and Post has a key of type string`
}
