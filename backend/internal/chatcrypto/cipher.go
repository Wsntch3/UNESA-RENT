package chatcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const nonceSize = 12

var (
	ErrInvalidKey     = errors.New("kunci chat tidak valid")
	ErrInvalidVersion = errors.New("versi kunci chat tidak ditemukan")
	ErrInvalidCipher  = errors.New("ciphertext chat tidak valid")
)

type Cipher struct {
	keys   map[int][]byte
	active int
}

func New(keysJSON string, active int) (*Cipher, error) {
	var raw map[string]string
	if err := json.Unmarshal([]byte(keysJSON), &raw); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidKey, err)
	}
	keys := make(map[int][]byte, len(raw))
	for versionText, encoded := range raw {
		var version int
		if _, err := fmt.Sscanf(versionText, "%d", &version); err != nil || version < 1 {
			return nil, fmt.Errorf("%w: versi %q tidak valid", ErrInvalidKey, versionText)
		}
		key, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || len(key) != 32 {
			return nil, fmt.Errorf("%w: versi %d harus base64 32 byte", ErrInvalidKey, version)
		}
		keys[version] = key
	}
	if _, ok := keys[active]; !ok {
		return nil, fmt.Errorf("%w: versi aktif %d", ErrInvalidVersion, active)
	}
	return &Cipher{keys: keys, active: active}, nil
}

func (c *Cipher) Encrypt(plaintext string) (ciphertext, nonce []byte, version int, err error) {
	version = c.active
	block, err := aes.NewCipher(c.keys[version])
	if err != nil {
		return nil, nil, 0, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, 0, err
	}
	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, 0, err
	}
	return gcm.Seal(nil, nonce, []byte(plaintext), nil), nonce, version, nil
}

func (c *Cipher) Decrypt(ciphertext, nonce []byte, version int) (string, error) {
	key, ok := c.keys[version]
	if !ok {
		return "", ErrInvalidVersion
	}
	if len(nonce) != nonceSize {
		return "", ErrInvalidCipher
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrInvalidCipher
	}
	return string(plain), nil
}
