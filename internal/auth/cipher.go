package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"unicode/utf16"

	"golang.org/x/crypto/pbkdf2"
)

var derivationSalt = []byte{0x49, 0x76, 0x61, 0x6e, 0x20, 0x4d, 0x65, 0x64, 0x76, 0x65, 0x64, 0x65, 0x76}

type Cipher struct {
	key []byte
	iv  []byte
}

func NewCipher(secret string) Cipher {
	derived := pbkdf2.Key([]byte(secret), derivationSalt, 1000, 48, sha1.New)
	return Cipher{key: derived[:32], iv: derived[32:48]}
}

func (c Cipher) Encrypt(clearText string) string {
	plain := padBlock(utf16Bytes(clearText))
	block, _ := aes.NewCipher(c.key)
	encrypted := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, c.iv).CryptBlocks(encrypted, plain)
	return base64.StdEncoding.EncodeToString(encrypted)
}

func (c Cipher) Decrypt(cipherText string) (string, error) {
	encrypted, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(encrypted) == 0 || len(encrypted)%aes.BlockSize != 0 {
		return "", errors.New("cipher text is not a whole number of blocks")
	}
	block, _ := aes.NewCipher(c.key)
	plain := make([]byte, len(encrypted))
	cipher.NewCBCDecrypter(block, c.iv).CryptBlocks(plain, encrypted)
	unpadded, err := unpadBlock(plain)
	if err != nil {
		return "", err
	}
	return utf16String(unpadded), nil
}

func utf16Bytes(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, 0, len(units)*2)
	for _, unit := range units {
		out = append(out, byte(unit), byte(unit>>8))
	}
	return out
}

func utf16String(data []byte) string {
	units := make([]uint16, 0, len(data)/2)
	for index := 0; index+1 < len(data); index += 2 {
		units = append(units, uint16(data[index])|uint16(data[index+1])<<8)
	}
	return string(utf16.Decode(units))
}

func padBlock(data []byte) []byte {
	padding := aes.BlockSize - len(data)%aes.BlockSize
	return append(data, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func unpadBlock(data []byte) ([]byte, error) {
	padding := int(data[len(data)-1])
	if padding == 0 || padding > aes.BlockSize || padding > len(data) {
		return nil, errors.New("padding is invalid")
	}
	return data[:len(data)-padding], nil
}
