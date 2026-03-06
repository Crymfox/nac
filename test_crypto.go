package main
import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	
	"fmt"
)
func main() {
	passphrase := "mykey"
	encrypted := "U2FsdGVkX19xLH5zjgQ+E2BaIfBAHIj8fSf3XPKkuPIeh7iOxnP33Poaaj0CN1gG"
	raw, _ := base64.StdEncoding.DecodeString(encrypted)
	salt := raw[8:16]
	fmt.Printf("Salt: %x\n", salt)

	// Derive key and IV
	data := make([]byte, len(passphrase)+len(salt))
	copy(data, passphrase)
	copy(data[len(passphrase):], salt)
	md5Hash := md5.Sum(data)
	d := md5Hash[:]
	d0 := d

	data = make([]byte, len(d0)+len(passphrase)+len(salt))
	copy(data, d0)
	copy(data[len(d0):], passphrase)
	copy(data[len(d0)+len(passphrase):], salt)
	md5Hash = md5.Sum(data)
	d = append(d, md5Hash[:]...)
	d1 := d[16:32]

	data = make([]byte, len(d1)+len(passphrase)+len(salt))
	copy(data, d1)
	copy(data[len(d1):], passphrase)
	copy(data[len(d1)+len(passphrase):], salt)
	md5Hash = md5.Sum(data)
	d = append(d, md5Hash[:]...)

	key := d[:32]
	iv := d[32:48]

	fmt.Printf("Key: %x\n", key)
	fmt.Printf("IV:  %x\n", iv)

	block, _ := aes.NewCipher(key)
	ciphertext := raw[16:]
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	fmt.Printf("Decrypted (hex): %x\n", ciphertext)
	fmt.Printf("Decrypted (string): %s\n", ciphertext)
}
