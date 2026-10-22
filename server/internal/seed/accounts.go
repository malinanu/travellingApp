package seed

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/spotto/server/internal/auth"
	"github.com/spotto/server/internal/domain"
	"github.com/spotto/server/internal/store/postgres"
)

// DemoEmailPattern matches every account the demo seeder and importsrilanka
// create. Their passwords are published in the README, or guessable from the
// public dataset, so these accounts must never be loginable in production.
const DemoEmailPattern = "%@spotto.dev"

// EnsureAdmin makes the ADMIN_EMAIL/ADMIN_PASSWORD account exist as an
// admin with exactly that password. It creates the account if missing,
// otherwise it corrects the role and password. Changing the password bumps
// token_version, which signs out sessions issued under the old one.
func EnsureAdmin(ctx context.Context, s *postgres.Store, email, password string) error {
	u, err := s.UserByEmail(ctx, email)
	if errors.Is(err, postgres.ErrNotFound) {
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return err
		}
		return s.CreateUser(ctx, domain.User{
			ID: "usr_admin_" + hex.EncodeToString(b), Email: email, PasswordHash: hash,
			Role: domain.RoleAdmin, Name: "Spotto Admin", CreatedAt: time.Now(),
		})
	}
	if err != nil {
		return fmt.Errorf("look up admin: %w", err)
	}
	if u.Role != domain.RoleAdmin {
		if err := s.SetUserRole(ctx, u.ID, domain.RoleAdmin); err != nil {
			return fmt.Errorf("set admin role: %w", err)
		}
	}
	if !auth.CheckPassword(u.PasswordHash, password) {
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		if err := s.UpdateUserPassword(ctx, u.ID, hash); err != nil {
			return fmt.Errorf("set admin password: %w", err)
		}
	}
	return nil
}

// LockDemoAccounts locks every demo and import account except keepEmail
// (the env-defined admin, which may share the domain). Production calls it
// at boot.
func LockDemoAccounts(ctx context.Context, s *postgres.Store, keepEmail string) (int64, error) {
	return s.LockUsersByEmail(ctx, DemoEmailPattern, keepEmail)
}
