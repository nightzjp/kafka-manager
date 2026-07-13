package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encryptedPrefix = "enc:v1:"

func Encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key); if err != nil { return "", fmt.Errorf("create cipher: %w", err) }
	gcm, err := cipher.NewGCM(block); if err != nil { return "", fmt.Errorf("create GCM: %w", err) }
	nonce := make([]byte, gcm.NonceSize()); if _, err := io.ReadFull(rand.Reader, nonce); err != nil { return "", err }
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func Decrypt(key []byte, value string) (string, error) {
	if !strings.HasPrefix(value, encryptedPrefix) { return value, nil }
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, encryptedPrefix)); if err != nil { return "", err }
	block, err := aes.NewCipher(key); if err != nil { return "", err }
	gcm, err := cipher.NewGCM(block); if err != nil { return "", err }
	if len(data) < gcm.NonceSize() { return "", fmt.Errorf("encrypted value is truncated") }
	plain, err := gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil); if err != nil { return "", fmt.Errorf("decrypt value: %w", err) }
	return string(plain), nil
}

func Runtime(persisted Config,key []byte)(Config,error){runtime:=persisted;runtime.Clusters=append([]ClusterConfig(nil),persisted.Clusters...);for i:=range runtime.Clusters{password,err:=Decrypt(key,runtime.Clusters[i].Security.Password);if err!=nil{return Config{},fmt.Errorf("decrypt password for cluster %s: %w",runtime.Clusters[i].ID,err)};runtime.Clusters[i].Security.Password=password};return runtime,nil}
