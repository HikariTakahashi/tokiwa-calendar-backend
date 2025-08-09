package main

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailConfig はメール送信設定を保持する構造体です
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// VerificationToken は認証トークンの情報を保持する構造体です
type VerificationToken struct {
	Token     string    `json:"token"`
	Email     string    `json:"email"`
	UID       string    `json:"uid"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// generateVerificationToken は認証トークンを生成します
func generateVerificationToken(email, uid string) (*VerificationToken, error) {
	// 32バイトのランダムなトークンを生成
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("トークン生成に失敗しました: %v", err)
	}
	
	token := hex.EncodeToString(tokenBytes)
	now := time.Now()
	
	verificationToken := &VerificationToken{
		Token:     token,
		Email:     email,
		UID:       uid,
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // 24時間後に期限切れ
	}
	
	return verificationToken, nil
}

// validateEmailConfig はメール設定の妥当性を検証します
func validateEmailConfig() error {
	config := getEmailConfig()
	
	var errors []string
	
	// 必須項目のチェック
	if config.SMTPUsername == "" {
		errors = append(errors, "SMTP_USERNAMEが設定されていません")
	}
	if config.SMTPPassword == "" {
		errors = append(errors, "SMTP_PASSWORDが設定されていません")
	}
	
	// メールアドレスの形式チェック
	if config.SMTPUsername != "" && !strings.Contains(config.SMTPUsername, "@") {
		errors = append(errors, "SMTP_USERNAMEが有効なメールアドレスではありません")
	}
	if config.FromEmail != "" && !strings.Contains(config.FromEmail, "@") {
		errors = append(errors, "FROM_EMAILが有効なメールアドレスではありません")
	}
	
	// パスワードの長さチェック（Gmailアプリパスワードは16文字）
	if config.SMTPPassword != "" && len(config.SMTPPassword) < 8 {
		errors = append(errors, "SMTP_PASSWORDが短すぎます（最低8文字必要）")
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("メール設定エラー: %s", strings.Join(errors, ", "))
	}
	
	return nil
}

// getEmailConfig は環境変数からメール設定を取得します
func getEmailConfig() EmailConfig {
	// SMTPユーザー名をデフォルトの送信者メールアドレスとして使用
	defaultFromEmail := os.Getenv("SMTP_USERNAME")
	if defaultFromEmail == "" {
		defaultFromEmail = "noreply@example.com" // フォールバック用
	}
	
	return EmailConfig{
		SMTPHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvOrDefault("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		FromEmail:    getEnvOrDefault("FROM_EMAIL", defaultFromEmail),
		FromName:     getEnvOrDefault("FROM_NAME", "Tokiwa Calendar"),
	}
}

// getEnvOrDefault は環境変数を取得し、存在しない場合はデフォルト値を返します
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// sendVerificationEmail は認証メールを送信します
func sendVerificationEmail(toEmail, verificationToken string) error {
	config := getEmailConfig()
	
	// 設定の検証
	if config.SMTPUsername == "" || config.SMTPPassword == "" {
		return fmt.Errorf("SMTP認証情報が設定されていません")
	}

	// 設定情報をログに出力（デバッグ用）
	log.Printf("INFO: Email config - SMTP: %s:%s, From: %s <%s>", 
		config.SMTPHost, config.SMTPPort, config.FromName, config.FromEmail)
	log.Printf("INFO: SMTP Username: %s", config.SMTPUsername)
	log.Printf("INFO: SMTP Password length: %d", len(config.SMTPPassword))

	// メール本文の作成
	subject := "Tokiwa Calendar - メールアドレスの確認"
	body := createVerificationEmailBodyFromTemplate(verificationToken)
	
	// メールヘッダーの作成
	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", config.FromName, config.FromEmail)
	headers["To"] = toEmail
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// メールヘッダーを文字列に変換
	var headerLines []string
	for key, value := range headers {
		headerLines = append(headerLines, fmt.Sprintf("%s: %s", key, value))
	}
	headerLines = append(headerLines, "") // 空行でヘッダーとボディを区切る

	// メール全体の作成
	message := strings.Join(headerLines, "\r\n") + body

	// SMTP認証
	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	// メール送信
	addr := fmt.Sprintf("%s:%s", config.SMTPHost, config.SMTPPort)
	log.Printf("INFO: Attempting to send email via %s", addr)
	
	// STARTTLS接続を使用
	log.Printf("INFO: Using STARTTLS connection")
	err := sendMail(addr, auth, config.FromEmail, []string{toEmail}, []byte(message))
	
	if err != nil {
		log.Printf("ERROR: Failed to send verification email to %s: %v", toEmail, err)
		log.Printf("ERROR: SMTP config - Host: %s, Port: %s, Username: %s", 
			config.SMTPHost, config.SMTPPort, config.SMTPUsername)
		return fmt.Errorf("メール送信に失敗しました: %v", err)
	}

	log.Printf("INFO: Verification email sent successfully to %s", toEmail)
	return nil
}



// sendMail はSTARTTLS接続を使用してメールを送信します
func sendMail(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// 通常のTCP接続を確立
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("TCP接続に失敗しました: %v", err)
	}
	defer conn.Close()
	
	// SMTPクライアントを作成
	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("SMTPクライアントの作成に失敗しました: %v", err)
	}
	defer client.Close()
	
	// STARTTLSを開始（さくらメールボックス用に柔軟な設定）
	if err = client.StartTLS(&tls.Config{
		ServerName:         strings.Split(addr, ":")[0],
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS10,
		MaxVersion:         tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}); err != nil {
		return fmt.Errorf("STARTTLSの開始に失敗しました: %v", err)
	}
	
	// 認証
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP認証に失敗しました: %v", err)
	}
	
	// 送信者を設定
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("送信者の設定に失敗しました: %v", err)
	}
	
	// 受信者を設定
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return fmt.Errorf("受信者の設定に失敗しました: %v", err)
		}
	}
	
	// メール本文を送信
	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("データ送信の準備に失敗しました: %v", err)
	}
	
	_, err = writer.Write(msg)
	if err != nil {
		return fmt.Errorf("メール本文の送信に失敗しました: %v", err)
	}
	
	err = writer.Close()
	if err != nil {
		return fmt.Errorf("データ送信の終了に失敗しました: %v", err)
	}
	
	return nil
}



 