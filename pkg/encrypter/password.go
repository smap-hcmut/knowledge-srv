package encrypter

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes a password using bcrypt with the default cost.
// Returns the hashed password as a string.
func (e implEncrypter) HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// CheckPasswordHash compares a password with its bcrypt hash.
// Returns true if the password matches the hash, false otherwise.
func (e implEncrypter) CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
