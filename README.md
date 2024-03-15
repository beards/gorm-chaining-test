# GORM method chaining examination

## Method chaining

GORM designed a mechanism to allow chaining calls like
```golang
db.Where("org = ?", "txone").Order("id").First(&employee)
```
which will give us
```sql
SELECT * FROM employee WHERE org = 'txone' ORDER BY id LIMIT 1;
```

However, it sometimes may be surprising.
Considering code piece like below:
```golang
db, _ = gorm.Open(postgres.Open("host=localhost port=5432"), &gorm.Config{})
fmt.Printf("db: %T(%p)\n", db, db)

q1 := db.Where("id = ?", 1)
q2 := db.Where("id = ?", 1).Where("name = ?", "nn")
q3 := db.Where("name = ?", "aa")
printQuery("q1", q1)
printQuery("q2", q2)
printQuery("q3", q3)
```

The output will be 
```
db: *gorm.DB(0x140002fc2a0)
q1: *gorm.DB(0x140002fdf20): SELECT * FROM " WHERE id = 1
q2: *gorm.DB(0x14000360000): SELECT * FROM " WHERE id = 1 AND name = 'nn'
q3: *gorm.DB(0x14000360120): SELECT * FROM " WHERE name = 'aa'
```

This is fine. We have three `gorm.DB` instance with three different statement just like we expected.

Now, if we do this furthering
```golang
q4 := q1.Where("name = ?", "mm")
printQuery("q1", q1)
printQuery("q4", q4)
```

Guess what?
```
q1: *gorm.DB(0x140002fdf20): SELECT * FROM " WHERE id = 1 AND name = 'mm'
q4: *gorm.DB(0x140002fdf20): SELECT * FROM " WHERE id = 1 AND name = 'mm'
```

`q4` is same instance with `q1` and `q1` was modified!
WTF?

## New session methods

OK, the official document told that we can avoid this by using 
[new session methods](https://gorm.io/docs/method_chaining.html#New-Session-Methods).
```golang
q3_ := q3.Session(&gorm.Session{})
q5 := q3_.Where("id = ?", 2)
printQuery("q3 ", q3)
printQuery("q3_", q3_)
printQuery("q5 ", q5)
```

We'll get
```
q3 : *gorm.DB(0x14000360120): SELECT * FROM " WHERE name = 'aa'
q3_: *gorm.DB(0x14000360c90): SELECT * FROM " WHERE name = 'aa'
q5 : *gorm.DB(0x14000360cc0): SELECT * FROM " WHERE name = 'aa' AND id = 2
```

Now `q5` has appended statement and `q3` was not modified, good.

But, here comes the question.  
We can see that when we call `db.Where` every time, 
we get different instance (`q1`, `q2`, `q3` are all different);  
When we call `q1.Where`, we get same instance (`q4` is same with `q1`);  
And when we call `q3_.Where`, we get different instance again (`q5` is different with `q3_`).

All the calls are same `gorm.DB.Where`, why sometimes it returns a new instance and sometimes it doesn't?

## getInstance()

Let us look into the `Where` method:

```golang
func (db *DB) Where(query interface{}, args ...interface{}) (tx *DB) {
	tx = db.getInstance()
	if conds := tx.Statement.BuildCondition(query, args...); len(conds) > 0 {
		tx.Statement.AddClause(clause.Where{Exprs: conds})
	}
	return
}
```

We can see that the returned `tx` is created by the `getInstance` method.
Actually all the [chainable APIs](https://github.com/go-gorm/gorm/blob/master/chainable_api.go)
have same logic. So what is the `getInstance` method?

```golang
func (db *DB) getInstance() *DB {
	if db.clone > 0 {
		tx := &DB{Config: db.Config, Error: db.Error}

		if db.clone == 1 {
			// clone with new statement
			tx.Statement = &Statement{
				DB:        tx,
				ConnPool:  db.Statement.ConnPool,
				Context:   db.Statement.Context,
				Clauses:   map[string]clause.Clause{},
				Vars:      make([]interface{}, 0, 8),
				SkipHooks: db.Statement.SkipHooks,
			}
		} else {
			// with clone statement
			tx.Statement = db.Statement.clone()
			tx.Statement.DB = tx
		}

		return tx
	}

	return db
}
```

There is a magic flag called `clone` in `gorm.DB` structure that controls how method calls applied.
When the `clone` flag is 0, `getInstance` just returns `db` itself;  
When the `clone` flag is greater than 0, `getInstance` will return a new instance.

Then the question becomes, when/where did the `clone` flag get set?
Well, it's quite easy to guess: those "new session methods".

The official document listed three "new session methods": `Session`, `WithContext`, and `Debug`.
It's very easy to notice that all of these methods are just `Session` calls with different config.
So let's look into the `Session` method:

```golang
func (db *DB) Session(config *Session) *DB {
	var (
		txConfig = *db.Config
		tx       = &DB{
			Config:    &txConfig,
			Statement: db.Statement,
			Error:     db.Error,
			clone:     1,
		}
	)
	// ...
	if !config.NewDB {
		tx.clone = 2
	}
	// ...
	return tx
}
```

Things get cleared now.

```
  (q1)              (q4)(q1)
 ┌────┐              ┌────┐
 │ DB │ --(Where)--> │ DB │
 └────┘              └────┘
(clone=0)           (clone=0)


  (q3)                  (q3_)               (q5)
 ┌────┐                ┌────┐              ┌────┐
 │ DB │ --(Session)--> │ DB │ --(Where)--> │ DB │
 └────┘                └────┘              └────┘
(clone=0)             (clone=2)           (clone=0)
```

How about `db` (`gorm.Open`)? It's also a kind of "new session method".

```golang
func Open(dialector Dialector, opts ...Option) (db *DB, err error) {
    // ...
    db = &DB{Config: config, clone: 1}
    // ...
    return
}
```

## Scopes

GORM provides an interface called [`Scopes()`](https://gorm.io/docs/scopes.html) 
that allow you to re-use commonly used logic, it *may* help to make your code cleaner in some situation, 
but it's also easy to introduce bugs if your querying conditions are complex.
Writing the conditional functions must be very, very carefully.

There are some examples in the demo code.
Check `ageBetween_XXX` for more details.
