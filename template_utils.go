package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"path/filepath"
)

// EmailTemplateData はメールテンプレートに渡すデータの構造体です
type EmailTemplateData struct {
	FrontendURL string
	Token       string
}

// loadEmailTemplate はHTMLテンプレートを読み込みます
func loadEmailTemplate(templateName string) (*template.Template, error) {
	// テンプレートファイルのパスを取得
	templatePath := filepath.Join("templates", templateName)
	
	// テンプレートファイルを読み込み
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("テンプレートファイルの読み込みに失敗しました: %v", err)
	}
	
	return tmpl, nil
}

// renderEmailTemplate はテンプレートをレンダリングします
func renderEmailTemplate(templateName string, data EmailTemplateData) (string, error) {
	// テンプレートを読み込み
	tmpl, err := loadEmailTemplate(templateName)
	if err != nil {
		return "", err
	}
	
	// テンプレートをレンダリング
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("テンプレートのレンダリングに失敗しました: %v", err)
	}
	
	return buf.String(), nil
}

// createVerificationEmailBodyFromTemplate は認証メールの本文を作成します
func createVerificationEmailBodyFromTemplate(token string) string {
	// テンプレートデータを準備
	data := EmailTemplateData{
		FrontendURL: getEnvOrDefault("FRONTEND_URL", "http://localhost:3000"),
		Token:       token,
	}
	
	// テンプレートをレンダリング
	body, err := renderEmailTemplate("verification_email.html", data)
	if err != nil {
		// テンプレート処理に失敗した場合のフォールバック
		log.Printf("WARN: Failed to render email template: %v, using fallback", err)
		return createFallbackEmailBody(token)
	}
	
	return body
}

// createFallbackEmailBody はテンプレート処理に失敗した場合のフォールバック用メール本文です
func createFallbackEmailBody(token string) string {
	frontendURL := getEnvOrDefault("FRONTEND_URL", "http://localhost:3000")
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>メールアドレスの確認</title>
</head>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h2 style="color: #2c3e50;">Tokiwa Calendar</h2>
        <h3>メールアドレスの確認</h3>
        
        <p>Tokiwa Calendarにご登録いただき、ありがとうございます。</p>
        
        <p>以下のリンクをクリックして、メールアドレスの確認を完了してください：</p>
        
        <div style="text-align: center; margin: 30px 0;">
            <a href="%s/verify-email?token=%s" 
               style="background-color: #3498db; color: white; padding: 12px 24px; 
                      text-decoration: none; border-radius: 5px; display: inline-block;">
                メールアドレスを確認する
            </a>
        </div>
        
        <p style="font-size: 14px; color: #7f8c8d;">
            このリンクは24時間後に無効になります。<br>
            このメールに心当たりがない場合は、無視していただいて構いません。
        </p>
        
        <hr style="border: none; border-top: 1px solid #ecf0f1; margin: 30px 0;">
        <p style="font-size: 12px; color: #95a5a6;">
            Tokiwa Calendar Team
        </p>
    </div>
</body>
</html>`, frontendURL, token)
} 