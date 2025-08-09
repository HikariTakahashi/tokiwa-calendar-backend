package main

import (
	"errors"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// UserSession はユーザーセッション情報を表します
type UserSession struct {
	UID       string    `json:"uid"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

// generateSessionToken はUIDとEmailからセッショントークンを生成します
func generateSessionToken(uid, email string) (string, error) {
	session := UserSession{
		UID:       uid,
		Email:     email,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24時間有効
	}

	// JWTクレームを作成
	claims := jwt.MapClaims{
		"uid":   session.UID,
		"email": session.Email,
		"exp":   session.ExpiresAt.Unix(),
		"iat":   time.Now().Unix(),
	}

	// JWTトークンを生成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 署名キーを取得（環境変数から）
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		// 開発環境用のデフォルトキー（本番では必ず環境変数を設定）
		secretKey = "your-default-secret-key-for-development-only"
	}

	return token.SignedString([]byte(secretKey))
}

// validateSessionToken はセッショントークンを検証してUserSessionを返します
func validateSessionToken(tokenString string) (*UserSession, error) {
	// デバッグ情報
	log.Printf("DEBUG: validateSessionToken called with token length: %d", len(tokenString))
	
	// 署名キーを取得
	secretKey := os.Getenv("JWT_SECRET")
	if secretKey == "" {
		secretKey = "your-default-secret-key-for-development-only"
		log.Printf("DEBUG: Using default JWT secret key")
	} else {
		log.Printf("DEBUG: Using environment JWT secret key")
	}

	// JWTトークンをパース・検証
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 署名メソッドの確認
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("ERROR: Unexpected signing method: %v", token.Method)
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		log.Printf("ERROR: JWT parse error: %v", err)
		return nil, err
	}

	// トークンの有効性確認
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	// クレームの抽出
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	// UIDとEmailの抽出
	uid, ok := claims["uid"].(string)
	if !ok {
		return nil, errors.New("uid not found in token")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return nil, errors.New("email not found in token")
	}

	// 有効期限の確認
	exp, ok := claims["exp"].(float64)
	if !ok {
		return nil, errors.New("expiration not found in token")
	}

	if time.Now().Unix() > int64(exp) {
		return nil, errors.New("token expired")
	}

	return &UserSession{
		UID:       uid,
		Email:     email,
		ExpiresAt: time.Unix(int64(exp), 0),
	}, nil
}

