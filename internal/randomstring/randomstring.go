package randomstring

import (
	"crypto/rand"
	"fmt"
)

const noCapsCharset = "abcdefghijklmnopqrstuvwxyz0123456789"
const withCapsCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func RandomString(length int, charset string) string {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

func RandomStringNoCaps(length int) string {
	return RandomString(length, noCapsCharset)
}

func RandomStringWithCaps(length int) string {
	return RandomString(length, withCapsCharset)
}
