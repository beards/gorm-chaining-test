package main

import (
	"flag"
	"fmt"

	"gorm.io/gorm"
)

var t struct{}

func printQuery(name string, query *gorm.DB) {
	//sql := DryRun(query).Find(&t).Statement.SQL.String()
	sql := DB().ToSQL(func(_ *gorm.DB) *gorm.DB {
		return DryRun(query).Find(&t)
	})
	fmt.Printf("%s: %T(%p): %s\n", name, query, query, sql)
}

func chainingTest() {
	db := DB()
	fmt.Printf("db: %T(%p)\n", db, db)

	q1 := db.Where("id = ?", 1)
	q2 := db.Where("id = ?", 1).Where("name = ?", "nn")
	q3 := db.Where("name = ?", "aa")

	printQuery("q1", q1)
	printQuery("q2", q2)
	printQuery("q3", q3)

	fmt.Println("")

	q4 := q1.Where("name = ?", "mm")
	printQuery("q1", q1)
	printQuery("q4", q4)

	fmt.Println("")

	q3_ := q3.Session(&gorm.Session{})
	q5 := q3_.Where("id = ?", 2)
	printQuery("q3 ", q3)
	printQuery("q3_", q3_)
	printQuery("q5 ", q5)
}

func activated(tx *gorm.DB) *gorm.DB {
	return tx.Where("activated = ?", true)
}

func locationIs(s string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("location = ?", s)
	}
}

func ageBetween_wrong1(a1, a2 int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("age >= ?", a1).Or("age <= ?", a2)
	}
}

func ageBetween_wrong2(a1, a2 int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		db = db.Where("age >= ?", a1).Or("age <= ?", a2)
		return db.Where(db)
	}
}

func ageBetween_wrong3(a1, a2 int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		inner := db.Session(&gorm.Session{}).Where("age >= ?", a1).Or("age <= ?", a2)
		return db.Where(inner)
	}
}

func ageBetween_fix1(a1, a2 int) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		clean := db.Session(&gorm.Session{NewDB: true})
		inner := clean.Where("age >= ?", a1).Or("age <= ?", a2)
		return db.Where(inner)
	}
}

type betweenFunc func(int, int) func(*gorm.DB) *gorm.DB

func scopeTest(name string, f betweenFunc) {
	q := DB().Scopes(
		activated,
		locationIs("test"),
		f(20, 30),
	)

	printQuery(name, q)
}

func ageBetween_fix2(base *gorm.DB, a1, a2 int) func(*gorm.DB) *gorm.DB {
	return func(scoped *gorm.DB) *gorm.DB {
		clean := base.Session(&gorm.Session{NewDB: true})
		inner := clean.Where("age >= ?", a1).Or("age <= ?", a2)
		return scoped.Where(base.Where(inner))
	}
}

func advancedTest() {
	base := activated(DB()).Session(&gorm.Session{})
	fq1 := base.Scopes(
		locationIs("test"),
		ageBetween_fix1(20, 30),
	)
	printQuery("fq1", fq1)

	fq2 := base.Scopes(
		locationIs("test"),
		ageBetween_fix2(base, 20, 30),
	)
	printQuery("fq2", fq2)
}

func main() {
	flag.Parse()

	switch cmd := flag.Arg(0); cmd {
	case "c":
		chainingTest()

	case "s1":
		scopeTest("s1", ageBetween_wrong1)

	case "s2":
		scopeTest("s2", ageBetween_wrong2)

	case "s3":
		scopeTest("s3", ageBetween_wrong3)

	case "f1":
		scopeTest("f1", ageBetween_fix1)

	case "a":
		advancedTest()

	default:
		fmt.Println("unknown command:", cmd)
	}
}
