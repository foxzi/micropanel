package database

import (
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
)

// Migrations 008 and 009 go up and down, and 009 marks sites that already
// have a certificate as issued.
func TestMigrationsGoUpAndDown(t *testing.T) {
	db, err := New(filepath.Join(t.TempDir(), "m.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir, _ := filepath.Abs("../../migrations")
	driver, err := sqlite3.WithInstance(db.DB, &sqlite3.Config{})
	if err != nil {
		t.Fatal(err)
	}
	m, err := migrate.NewWithDatabaseInstance("file://"+dir, "sqlite3", driver)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(8); err != nil {
		t.Fatalf("up to 8: %v", err)
	}
	res, err := db.Exec(`INSERT INTO users (email, password_hash, role, is_active) VALUES ('a@b.c','x','user',1)`)
	if err != nil {
		t.Fatal(err)
	}
	owner, _ := res.LastInsertId()
	if _, err := db.Exec(`INSERT INTO sites (name, owner_id, ssl_enabled) VALUES ('with-cert.test', ?, 1), ('without.test', ?, 0)`, owner, owner); err != nil {
		t.Fatal(err)
	}
	if err := m.Migrate(9); err != nil {
		t.Fatalf("up to 9: %v", err)
	}
	var withCert, without string
	db.QueryRow(`SELECT ssl_status FROM sites WHERE name = 'with-cert.test'`).Scan(&withCert)
	db.QueryRow(`SELECT ssl_status FROM sites WHERE name = 'without.test'`).Scan(&without)
	if withCert != "issued" || without != "none" {
		t.Fatalf("backfill: with cert %q, without %q", withCert, without)
	}
	if err := m.Migrate(7); err != nil {
		t.Fatalf("down to 7: %v", err)
	}
	if err := m.Up(); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
