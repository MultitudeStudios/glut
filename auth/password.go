package auth

import (
	"errors"

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
