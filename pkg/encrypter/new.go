package encrypter

// Encrypter provides encryption and decryption functionality using AES-GCM.
type Encrypter interface {
	// Encrypt encrypts a plaintext string and returns a base64-encoded ciphertext.
	Encrypt(plaintext string) (string, error)
	// Decrypt decrypts a base64-encoded ciphertext string and returns the plaintext.
	Decrypt(ciphertext string) (string, error)
	// EncryptBytesToString encrypts a byte slice and returns a base64-encoded ciphertext.
	EncryptBytesToString(data []byte) (string, error)
	// DecryptStringToBytes decrypts a base64-encoded ciphertext string and returns plaintext bytes.
	DecryptStringToBytes(ciphertext string) ([]byte, error)

	// HashPassword hashes a password using bcrypt with the default cost.
	HashPassword(password string) (string, error)

	// CheckPasswordHash compares a password with its bcrypt hash.
	CheckPasswordHash(password, hash string) bool
}

type implEncrypter struct {
	key string
}

// New creates a new Encrypter instance with the provided key.
// The key must be 16, 24, or 32 bytes long for AES-128, AES-192, or AES-256 respectively.
func New(key string) Encrypter {
	return &implEncrypter{
		key: key,
	}
}
