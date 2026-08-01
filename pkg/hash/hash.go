// Package hash wraps bcrypt for password hashing.
package hash

import "golang.org/x/crypto/bcrypt"

// Hash bcrypt-hashes a plaintext password.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}

// Compare reports whether password matches the bcrypt hash.
func Compare(hashed, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}
