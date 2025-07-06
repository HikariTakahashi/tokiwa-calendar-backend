package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// PasswordStrengthResult はパスワード強度チェックの結果を表します
type PasswordStrengthResult struct {
	IsValid bool
	Errors  []string
}

// checkPasswordStrength はパスワードの強度をチェックします
func checkPasswordStrength(password string) PasswordStrengthResult {
	var errors []string

	// 最小長チェック
	if len(password) < 8 {
		errors = append(errors, "パスワードは8文字以上で入力してください")
	}

	// 大文字チェック
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		errors = append(errors, "大文字を含めてください")
	}

	// 小文字チェック
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		errors = append(errors, "小文字を含めてください")
	}

	// 数字チェック
	if !regexp.MustCompile(`\d`).MatchString(password) {
		errors = append(errors, "数字を含めてください")
	}

	// アルファベットと数字以外の文字を禁止
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(password) {
		errors = append(errors, "パスワードは英数字のみ使用可能です")
	}

	return PasswordStrengthResult{
		IsValid: len(errors) == 0,
		Errors:  errors,
	}
}

// generateSalt はランダムなソルトを生成します
func generateSalt(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashPasswordWithSalt はパスワードをソルト付きでハッシュ化します
func hashPasswordWithSalt(password, salt string) string {
	// パスワードとソルトを結合
	combined := password + salt
	
	// SHA-256でハッシュ化
	hash := sha256.Sum256([]byte(combined))
	
	// 16進数文字列に変換
	return hex.EncodeToString(hash[:])
}

// HashPasswordForStorage はパスワードをストレージ用にハッシュ化します
func HashPasswordForStorage(password string) (string, string, error) {
	// ソルトを生成
	salt, err := generateSalt(32)
	if err != nil {
		return "", "", fmt.Errorf("ソルト生成エラー: %v", err)
	}

	// パスワードをハッシュ化
	hashedPassword := hashPasswordWithSalt(password, salt)

	return hashedPassword, salt, nil
}

// VerifyPassword はパスワードを検証します
func VerifyPassword(password, hashedPassword, salt string) bool {
	// 入力されたパスワードを同じソルトでハッシュ化
	inputHash := hashPasswordWithSalt(password, salt)
	
	// ハッシュ値を比較
	return strings.EqualFold(inputHash, hashedPassword)
} 