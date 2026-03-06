package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

const (
	saltLen    = 8
	magicStart = "Salted__"
)

// Encrypt encrypts a string using AES-256-CBC with MD5 key derivation,
// matching OpenSSL's enc and n8n's encryption format.
func Encrypt(plaintext, passphrase string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Generate 8-byte random salt
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Derive 32-byte key (AES-256) and 16-byte IV using EVP_BytesToKey (MD5)
	key, iv := evpBytesToKey(passphrase, salt)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// Apply PKCS#7 padding
	paddedData := pkcs7Pad([]byte(plaintext), aes.BlockSize)

	// Encrypt
	ciphertext := make([]byte, len(paddedData))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedData)

	// Assemble final payload: "Salted__" + salt + ciphertext
	payload := make([]byte, 0, len(magicStart)+saltLen+len(ciphertext))
	payload = append(payload, []byte(magicStart)...)
	payload = append(payload, salt...)
	payload = append(payload, ciphertext...)

	// Return Base64 encoded string
	return base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt decrypts an OpenSSL AES-256-CBC Base64 string,
// matching n8n's encryption format.
func Decrypt(encryptedBase64, passphrase string) (string, error) {
	if encryptedBase64 == "" {
		return "", nil
	}

	// Decode Base64
	payload, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return "", err
	}

	// Check minimum length (magic + salt + at least 1 block)
	minLen := len(magicStart) + saltLen + aes.BlockSize
	if len(payload) < minLen {
		return "", errors.New("encrypted payload too short")
	}

	// Verify "Salted__" magic string
	if string(payload[:len(magicStart)]) != magicStart {
		return "", errors.New("missing 'Salted__' magic string")
	}

	// Extract salt
	salt := payload[len(magicStart) : len(magicStart)+saltLen]
	ciphertext := payload[len(magicStart)+saltLen:]

	// Check ciphertext is multiple of block size
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", errors.New("ciphertext length is not a multiple of the block size")
	}

	// Derive key and IV
	key, iv := evpBytesToKey(passphrase, salt)

	// Decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Remove PKCS#7 padding
	unpadded, err := pkcs7Unpad(ciphertext)
	if err != nil {
		return "", err
	}

	return string(unpadded), nil
}

// evpBytesToKey implements OpenSSL's EVP_BytesToKey algorithm.
// n8n uses MD5, 1 iteration, to derive a 32-byte key and 16-byte IV.
func evpBytesToKey(passphrase string, salt []byte) (key []byte, iv []byte) {
	var d []byte
	var d_i []byte

	for len(d) < 48 {
		data := make([]byte, len(d_i)+len(passphrase)+len(salt))
		copy(data, d_i)
		copy(data[len(d_i):], passphrase)
		copy(data[len(d_i)+len(passphrase):], salt)

		hash := md5.Sum(data)
		d_i = hash[:]
		d = append(d, d_i...)
	}

	// Return 32-byte key and 16-byte IV
	return d[:32], d[32:48]
}

// pkcs7Pad applies PKCS#7 padding.
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpad removes PKCS#7 padding.
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("empty data")
	}

	paddingLen := int(data[length-1])
	if paddingLen == 0 || paddingLen > length {
		return nil, errors.New("invalid padding length")
	}

	for i := 0; i < paddingLen; i++ {
		if data[length-1-i] != byte(paddingLen) {
			return nil, errors.New("invalid padding bytes")
		}
	}

	return data[:length-paddingLen], nil
}
