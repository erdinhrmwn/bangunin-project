// Package hash wraps bcrypt for password hashing.
package hash

import "golang.org/x/crypto/bcrypt"

// Cost 12 per FR-2.1 (bcrypt.DefaultCost is only 10).
const cost = 12

// Hash bcrypt-hashes a plaintext password.
func Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(b), err
}

// Compare reports whether password matches the bcrypt hash.
func Compare(hashed, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password)) == nil
}
