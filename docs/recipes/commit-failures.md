# When a commit fails

Most failures are simple: the transaction rolled back, and nothing happened. One is not. If the connection drops while a `COMMIT` is on its way, the client cannot know whether the database received it. The transaction may have committed, and your code sees an error.

Retrying is then unsafe, because the work may already be done. `ant.Attempts` and `ant.WithRetry` do not retry this case: they only retry deadlocks and serialization failures, which PostgreSQL guarantees rolled back.

## Make the work safe to repeat

Give each operation a key the client chooses, and store it in the same transaction, with a unique constraint. A retry then finds the key, and returns the first result.

```go
existing, err := tx.Entries.Where(ant.Eq(&E.Key, &key)).First(ctx)
if err == nil {
    return existing, nil // it committed after all
}
```

The [ledger example](https://github.com/skudra-go/ant-examples/tree/main/examples/ledger) does this for transfers, and the [audit example](https://github.com/skudra-go/ant-examples/tree/main/examples/audit) does it with event IDs.

## Or check afterwards

Where a client key does not fit, reconnect and look for the result, using something the transaction wrote.

## After SaveChanges fails

A failed `SaveChanges` leaves pending changes in the tracker, and clears generated keys, so it can be called again. If the failure was a lost commit, check first, or the rows go in twice.
