package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// getEncryptionKey は環境変数から暗号化キーを取得します
func getEncryptionKey() string {
	key := os.Getenv("ENCRYPTION_KEY")
	if key == "" {
		// 開発環境用のデフォルトキー
		return "your-secret-encryption-key-32-chars-long!"
	}
	return key
}

// DecryptPassword は暗号化されたパスワードを復号化します（簡易版）
func DecryptPassword(encryptedPassword string) (string, error) {
	// Base64デコード
	decoded, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("Base64デコードエラー: %v", err)
	}

	// パスワードとキーを分離
	parts := strings.Split(string(decoded), ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("暗号化データの形式が不正です")
	}

	// キーの検証（オプション）
	expectedKey := getEncryptionKey()
	if parts[1] != expectedKey {
		return "", fmt.Errorf("暗号化キーが一致しません")
	}

	return parts[0], nil // パスワード部分を返す
}

// EncryptPassword はパスワードを暗号化します（テスト用）
func EncryptPassword(password string) (string, error) {
	// キーを16バイトに調整（AES-CBC用）
	key := []byte(getEncryptionKey())
	if len(key) < 16 {
		paddedKey := make([]byte, 16)
		copy(paddedKey, key)
		key = paddedKey
	} else if len(key) > 16 {
		key = key[:16]
	}

	// AES-CBCブロック暗号を作成
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("AES暗号作成エラー: %v", err)
	}

	// PKCS7パディングを追加
	paddedData := pkcs7Pad([]byte(password), aes.BlockSize)

	// 初期化ベクトル（IV）を生成
	iv := make([]byte, aes.BlockSize)
	if _, err := rand.Read(iv); err != nil {
		return "", fmt.Errorf("IV生成エラー: %v", err)
	}

	// CBCモードで暗号化
	mode := cipher.NewCBCEncrypter(block, iv)
	ciphertext := make([]byte, len(paddedData))
	mode.CryptBlocks(ciphertext, paddedData)

	// IVと暗号化データを結合
	combined := make([]byte, len(iv)+len(ciphertext))
	copy(combined, iv)
	copy(combined[len(iv):], ciphertext)

	// Base64エンコード
	return base64.StdEncoding.EncodeToString(combined), nil
}

// pkcs7Pad はPKCS7パディングを追加します
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
} 