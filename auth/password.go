package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"golang.org/x/crypto/bcrypt"
)

type PasswordCompareFunc func(hash, plain string) (bool, error)

func hashPassword(password string) (string, error) {
	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	passwordHash := string(passwordBytes)
	return passwordHash, nil
}

func comparePasswords(passwordHash string, password string) (bool, error) {
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func validatePassword(hash, plain string, compareFunc PasswordCompareFunc) error {
	match, err := compareFunc(hash, plain)
	if err != nil {
		return err
	}
	if !match {
		return ErrInvalidPassword
	}
	return nil
}

func updateUserPassword(ctx context.Context, tx pgx.Tx, userID, password string) error {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return fmt.Errorf("auth.updateUserPassword: %w", err)
	}

	sql, args := psql.Update(
		um.Table("auth.users"),
		um.Set("password_hash").ToArg(passwordHash),
		um.Where(psql.Quote("id").EQ(psql.Arg(userID))),
	).MustBuild()

	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("auth.updateUserPassword: %w", err)
	}
	return nil
}
