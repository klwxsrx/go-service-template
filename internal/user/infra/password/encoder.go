package password

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"

	"github.com/klwxsrx/go-service-template/internal/user/app/encoding"
)

// encoder is a fake example implementation
// Use bcrypt in a real application
type encoder struct{}

func NewEncoder() encoding.PasswordEncoder {
	return encoder{}
}

func (e encoder) HashPassword(password string) (string, error) {
	hash := md5.Sum([]byte(password)) //nolint:gosec
	return hex.EncodeToString(hash[:]), nil
}

func (e encoder) CompareHash(passwordHash, password string) bool {
	hash, err := e.HashPassword(password)
	return err == nil && hash == passwordHash
}
