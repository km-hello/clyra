package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"golang.org/x/crypto/argon2"
	"strings"
)

const passwordPrefix = "$argon2id$v=19$m=65536,t=3,p=4$"

// Bound Argon2 memory use to two concurrent 64 MiB derivations.
var passwordSlots = make(chan struct{}, 2)

func derivePassword(password string, salt []byte) []byte {
	passwordSlots <- struct{}{}
	defer func() { <-passwordSlots }()
	return argon2.IDKey([]byte(password), salt, 3, 64*1024, 4, 32)
}
func Hash(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := derivePassword(password, salt)
	return passwordPrefix + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func Verify(hash, password string) bool {
	if !strings.HasPrefix(hash, passwordPrefix) {
		return false
	}
	parts := strings.Split(strings.TrimPrefix(hash, passwordPrefix), "$")
	if len(parts) != 2 {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[0])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil || len(expected) != 32 {
		return false
	}
	return subtle.ConstantTimeCompare(derivePassword(password, salt), expected) == 1
}
