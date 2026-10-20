package seed_test

import (
	"context"
	"testing"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/seed"
	"github.com/spotto/server/internal/store/postgres"
)

func connect(t *testing.T) *postgres.Store {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	store, err := postgres.Connect(ctx, "postgres://postgres:admin123@localhost:5432/spotto?sslmode=disable")
	if err != nil {
		t.Skipf("skipping: no local postgres available: %v", err)
	}
	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestEnsureAdmin(t *testing.T) {
	s := connect(t)
	ctx := context.Background()
	const email = "boss@ensureadmin.test"
	t.Cleanup(func() { s.Pool.Exec(ctx, `DELETE FROM users WHERE email = $1`, email) })

	// First boot creates the account.
	if err := seed.EnsureAdmin(ctx, s, email, "first-long-password"); err != nil {
		t.Fatal(err)
	}
	u, err := s.UserByEmail(ctx, email)
	if err != nil {
		t.Fatal(err)
	}
	if u.Role != domain.RoleAdmin || !auth.CheckPassword(u.PasswordHash, "first-long-password") {
		t.Fatalf("created admin wrong: role=%s", u.Role)
	}

	// A later boot with a rotated password and a demoted role corrects both
	// and signs out old tokens.
	if err := s.SetUserRole(ctx, u.ID, domain.RoleTraveller); err != nil {
		t.Fatal(err)
	}
	if err := seed.EnsureAdmin(ctx, s, email, "second-long-password"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.UserByEmail(ctx, email)
	if got.Role != domain.RoleAdmin {
		t.Errorf("role not restored: %s", got.Role)
	}
	if !auth.CheckPassword(got.PasswordHash, "second-long-password") || auth.CheckPassword(got.PasswordHash, "first-long-password") {
		t.Error("password not rotated")
	}
	if got.TokenVersion <= u.TokenVersion {
		t.Error("token_version not bumped")
	}
}

func TestLockUsersByEmailKeepsAdmin(t *testing.T) {
	s := connect(t)
	ctx := context.Background()
	t.Cleanup(func() { s.Pool.Exec(ctx, `DELETE FROM users WHERE email LIKE '%@lockdemo.test'`) })

	hash, _ := auth.HashPassword("spotto123")
	for _, id := range []string{"demo", "boss"} {
		if err := s.CreateUser(ctx, domain.User{ID: "usr_lockdemo_" + id, Email: id + "@lockdemo.test", PasswordHash: hash, Role: domain.RoleTraveller, CreatedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}

	n, err := s.LockUsersByEmail(ctx, "%@lockdemo.test", "BOSS@lockdemo.test")
	if err != nil || n != 1 {
		t.Fatalf("want 1 locked, got %d (%v)", n, err)
	}
	demo, _ := s.UserByEmail(ctx, "demo@lockdemo.test")
	boss, _ := s.UserByEmail(ctx, "boss@lockdemo.test")
	if auth.CheckPassword(demo.PasswordHash, "spotto123") {
		t.Error("demo account still accepts its public password")
	}
	if !auth.CheckPassword(boss.PasswordHash, "spotto123") {
		t.Error("kept admin was locked")
	}
	// A second boot finds nothing new to lock.
	if n, _ := s.LockUsersByEmail(ctx, "%@lockdemo.test", "boss@lockdemo.test"); n != 0 {
		t.Errorf("relock: want 0, got %d", n)
	}
}
