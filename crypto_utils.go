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

// DecryptPassword はAES-CBCで暗号化されたパスワードを復号化します
func DecryptPassword(encryptedPassword string) (string, error) {
	key := []byte(getEncryptionKey())
	if len(key) > 16 {
		key = key[:16]
	} else if len(key) < 16 {
		return "", fmt.Errorf("暗号化キーは16バイト以上である必要があります")
	}

	// Base64デコード
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("Base64デコードエラー: %v", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("暗号文が短すぎます")
	}

	// IV（初期化ベクトル）と暗号データを分離
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("AES暗号作成エラー: %v", err)
	}

	// CBCモードで復号
	mode := cipher.NewCBCDecrypter(block, iv)
	// 復号後のデータは元の暗号文と同じサイズのスライスに書き込まれる
	mode.CryptBlocks(ciphertext, ciphertext)

	// PKCS7パディングを削除
	decrypted, err := pkcs7Unpad(ciphertext)
	if err != nil {
		return "", fmt.Errorf("パディング解除エラー: %v", err)
	}

	return string(decrypted), nil
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

// pkcs7Unpad はPKCS7パディングを削除します
func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, fmt.Errorf("pkcs7: unpadding error - data is empty")
	}
	unpadding := int(data[length-1])
	if unpadding > length || unpadding == 0 {
		return nil, fmt.Errorf("pkcs7: unpadding error - invalid padding size")
	}
	return data[:(length - unpadding)], nil
}

// decryptPassword はフロントエンドの簡易暗号化方式に対応した復号化関数です
func decryptPassword(encryptedPassword string) (string, error) {
	// Base64デコード
	decoded, err := base64.StdEncoding.DecodeString(encryptedPassword)
	if err != nil {
		return "", fmt.Errorf("Base64デコードエラー: %v", err)
	}

	// パスワードとキーを分離
	combined := string(decoded)
	parts := strings.Split(combined, ":")
	if len(parts) != 2 {
		return "", fmt.Errorf("暗号化データの形式が不正です")
	}

	return parts[0], nil // パスワード部分を返す
}