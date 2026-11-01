# Encrypted fields

`secure.Value[T]` stores a value encrypted with AES-256-GCM, so a database dump or a stolen backup reveals nothing.

```go
secure.SetKeys("2026", map[string][]byte{"2026": key})    // 32 bytes, at startup

type Patient struct {
    ID  int64
    SSN secure.Value[string]
}

patient := &Patient{SSN: secure.New("123-45-6789")}
db.Patients.Insert(ctx, patient)

loaded, err := db.Patients.Find(ctx, patient.ID)
fmt.Println(loaded.SSN.V)
```

The value cannot leak by accident: `fmt` and JSON both print `[encrypted]`. Read `.V` to use it. Add `.Sensitive()` too, so that `ant.LogValues` keeps it out of logs.

## Rotating keys

Add a new key and make it active. Rows still read with the old one, because each value names its key, and every write uses the new one.

```go
secure.SetKeys("2027", map[string][]byte{"2027": newKey, "2026": oldKey})
```

To finish, load and save the rows that still use the old key, then drop it. `ant.ChunkByKey` suits this, and this SQL finds what is left:

```sql
SELECT count(*) FROM patients WHERE ssn NOT LIKE '2027:%';
```

## Searching

A value encrypts differently every time, so `ant.Eq` cannot find it, and an index is useless. To look rows up, store a keyed hash beside it:

```go
type Patient struct {
    ID      int64
    SSN     secure.Value[string]
    SSNHash []byte `ant:"unique"`
}

func blindIndex(value string) []byte {
    mac := hmac.New(sha256.New, hashKey)
    mac.Write([]byte(value))
    return mac.Sum(nil)
}

db.Patients.Where(ant.Eq(&P.SSNHash, blindIndex(ssn))).First(ctx)
```

The hash needs its own key. It reveals when two rows hold the same value, so use it only where that is acceptable.

## Limits

- A value is not tied to its row, so someone who can write to the table can copy a ciphertext from one row to another.
- Keys live in the process. Keep them in a secret store, and never in the repository.
- GCM with random nonces is safe for about four billion values per key. Rotate before then.
- It protects data at rest. It does not protect against an application that is itself compromised.

The [ledger example](https://github.com/skudra-go/ant-examples/tree/main/examples/ledger) stores IBANs this way.
