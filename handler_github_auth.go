package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"firebase.google.com/go/v4/auth"
	"github.com/aws/aws-lambda-go/events"
)

// GitHubAuthRequest はGitHub OAuth2.0認証リクエストの構造体です
type GitHubAuthRequest struct {
	Code        string `json:"code"`
	RedirectURI string `json:"redirect_uri"`
}

// GitHubAuthResponse はGitHub OAuth2.0認証レスポンスの構造体です
type GitHubAuthResponse struct {
	Message     string `json:"message"`
	UID         string `json:"uid,omitempty"`
	Email       string `json:"email,omitempty"`
	CustomToken string `json:"customToken,omitempty"`
	Error       string `json:"error,omitempty"`
}

// GitHubTokenResponse はGitHub OAuth2.0トークンレスポンスの構造体です
type GitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

// GitHubUserInfo はGitHubユーザー情報の構造体です
type GitHubUserInfo struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Company   string `json:"company"`
	Blog      string `json:"blog"`
	Location  string `json:"location"`
	Bio       string `json:"bio"`
}

// getGitHubClientID は環境変数からGitHub Client IDを取得します
func getGitHubClientID() string {
	return os.Getenv("GITHUB_CLIENT_ID")
}

// getGitHubClientSecret は環境変数からGitHub Client Secretを取得します
func getGitHubClientSecret() string {
	return os.Getenv("GITHUB_CLIENT_SECRET")
}

// exchangeGitHubCodeForToken は認証コードをアクセストークンと交換します
func exchangeGitHubCodeForToken(code, redirectURI string) (*GitHubTokenResponse, error) {
	clientID := getGitHubClientID()
	clientSecret := getGitHubClientSecret()

	if clientID == "" || clientSecret == "" {
		return nil, fmt.Errorf("GitHub OAuth2.0設定が不完全です")
	}

	// トークンエンドポイントにリクエスト
	tokenURL := "https://github.com/login/oauth/access_token"
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return nil, fmt.Errorf("トークン交換リクエストエラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("トークン交換失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	// GitHubのレスポンスはapplication/x-www-form-urlencoded形式
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("レスポンス読み取りエラー: %v", err)
	}

	// URLエンコードされたレスポンスをパース
	values, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("レスポンス解析エラー: %v", err)
	}

	// エラーチェック
	if values.Get("error") != "" {
		return nil, fmt.Errorf("GitHub OAuthエラー: %s - %s", values.Get("error"), values.Get("error_description"))
	}

	accessToken := values.Get("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("アクセストークンが取得できませんでした")
	}

	return &GitHubTokenResponse{
		AccessToken: accessToken,
		TokenType:   values.Get("token_type"),
		Scope:       values.Get("scope"),
	}, nil
}

// getUserInfoFromGitHub はGitHubからユーザー情報を取得します
func getUserInfoFromGitHub(accessToken string) (*GitHubUserInfo, error) {
	userInfoURL := "https://api.github.com/user"
	
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報リクエスト作成エラー: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tokiwa-calendar")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ユーザー情報取得エラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ユーザー情報取得失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var userInfo GitHubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("ユーザー情報解析エラー: %v", err)
	}

	// メールアドレスが取得できない場合は、メールエンドポイントから取得を試行
	if userInfo.Email == "" {
		email, err := getUserEmailFromGitHub(accessToken)
		if err != nil {
			log.Printf("WARN: Failed to get user email: %v", err)
		} else {
			userInfo.Email = email
		}
	}

	return &userInfo, nil
}

// getUserEmailFromGitHub はGitHubからユーザーのメールアドレスを取得します
func getUserEmailFromGitHub(accessToken string) (string, error) {
	emailURL := "https://api.github.com/user/emails"
	
	req, err := http.NewRequest("GET", emailURL, nil)
	if err != nil {
		return "", fmt.Errorf("メール情報リクエスト作成エラー: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "tokiwa-calendar")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("メール情報取得エラー: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("メール情報取得失敗 (HTTP %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var emails []struct {
		Email   string `json:"email"`
		Primary bool   `json:"primary"`
		Verified bool  `json:"verified"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", fmt.Errorf("メール情報解析エラー: %v", err)
	}

	// プライマリで認証済みのメールアドレスを探す
	for _, email := range emails {
		if email.Primary && email.Verified {
			return email.Email, nil
		}
	}

	// プライマリのメールアドレスを探す
	for _, email := range emails {
		if email.Primary {
			return email.Email, nil
		}
	}

	// 最初のメールアドレスを返す
	if len(emails) > 0 {
		return emails[0].Email, nil
	}

	return "", fmt.Errorf("メールアドレスが見つかりませんでした")
}

// createOrGetFirebaseUserForGitHub はGitHubアカウントでFirebaseユーザーを作成または取得します
func createOrGetFirebaseUserForGitHub(ctx context.Context, githubUser *GitHubUserInfo) (string, error) {
	// メールアドレスが取得できない場合はエラー
	if githubUser.Email == "" {
		return "", fmt.Errorf("GitHubアカウントからメールアドレスを取得できませんでした")
	}

	// 既存のユーザーを確認
	userRecord, err := authClient.GetUserByEmail(ctx, githubUser.Email)
	if err != nil {
		// ユーザーが存在しない場合は新しいGitHubユーザーを作成
		uid := fmt.Sprintf("github_%d", githubUser.ID)
		params := (&auth.UserToCreate{}).
			Email(githubUser.Email).
			DisplayName(githubUser.Name).
			PhotoURL(githubUser.AvatarURL).
			EmailVerified(true).
			UID(uid)

		userRecord, err = authClient.CreateUser(ctx, params)
		if err != nil {
			return "", fmt.Errorf("Firebaseユーザー作成エラー: %v", err)
		}
		log.Printf("INFO: Created new Firebase user for GitHub account: %s", githubUser.Email)
	} else {
		// 既存のユーザーが見つかった場合
		log.Printf("INFO: Found existing Firebase user for email: %s", githubUser.Email)
		
		// 既存のユーザーがGitHubプロバイダーで作成されているかチェック
		providers := userRecord.ProviderUserInfo
		hasGitHubProvider := false
		
		for _, provider := range providers {
			if provider.ProviderID == "github.com" {
				hasGitHubProvider = true
				break
			}
		}
		
		if !hasGitHubProvider {
			// 既存のユーザーがメールアドレスログインで作成されている場合
			// この場合、既存のユーザーアカウントを使用する
			log.Printf("INFO: Using existing email user for GitHub login: %s", githubUser.Email)
			
			// 既存ユーザーの情報を更新（表示名やプロフィール画像など）
			updateParams := (&auth.UserToUpdate{}).
				DisplayName(githubUser.Name).
				PhotoURL(githubUser.AvatarURL).
				EmailVerified(true)
			
			_, err = authClient.UpdateUser(ctx, userRecord.UID, updateParams)
			if err != nil {
				log.Printf("WARN: Failed to update user profile: %v", err)
				// 更新に失敗してもログインは続行
			}
		}
	}

	return userRecord.UID, nil
}

func processGitHubAuthRequest(ctx context.Context, req interface{}) (map[string]interface{}, int) {
	var bodyBytes []byte
	var err error

	// リクエストソース（ローカルサーバー or Lambda）に応じてリクエストボディをバイトスライスとして取得
	switch r := req.(type) {
	case *http.Request:
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Printf("ERROR: Failed to read request body: %v\n", err)
			return map[string]interface{}{"error": "リクエストの処理に失敗しました"}, http.StatusInternalServerError
		}
		defer r.Body.Close()
	case events.APIGatewayV2HTTPRequest:
		bodyBytes = []byte(r.Body)
	default:
		log.Printf("ERROR: Unknown request type: %T\n", r)
		return map[string]interface{}{"error": "不明なリクエストタイプです"}, http.StatusInternalServerError
	}

	// JSONを構造体にデコード
	var authData GitHubAuthRequest
	if err := json.Unmarshal(bodyBytes, &authData); err != nil {
		log.Printf("WARN: Failed to parse GitHub auth JSON: %v. Body: %s", err, string(bodyBytes))
		return map[string]interface{}{"error": "リクエストされたJSONの形式が正しくありません。"}, http.StatusBadRequest
	}

	// バリデーション
	if authData.Code == "" {
		return map[string]interface{}{"error": "認証コードが提供されていません"}, http.StatusBadRequest
	}
	if authData.RedirectURI == "" {
		return map[string]interface{}{"error": "リダイレクトURIが提供されていません"}, http.StatusBadRequest
	}

	log.Printf("INFO: GitHub OAuth2.0 request received with code length: %d", len(authData.Code))

	// 認証コードをアクセストークンと交換
	tokenResponse, err := exchangeGitHubCodeForToken(authData.Code, authData.RedirectURI)
	if err != nil {
		log.Printf("ERROR: Failed to exchange code for token: %v\n", err)
		return map[string]interface{}{"error": "認証コードの交換に失敗しました"}, http.StatusBadRequest
	}

	// GitHubからユーザー情報を取得
	userInfo, err := getUserInfoFromGitHub(tokenResponse.AccessToken)
	if err != nil {
		log.Printf("ERROR: Failed to get user info from GitHub: %v\n", err)
		return map[string]interface{}{"error": "ユーザー情報の取得に失敗しました"}, http.StatusInternalServerError
	}

	// メールアドレスの検証
	if userInfo.Email == "" {
		return map[string]interface{}{"error": "メールアドレスが取得できませんでした"}, http.StatusBadRequest
	}

	log.Printf("INFO: GitHub user info retrieved for email: %s", userInfo.Email)

	// Firebaseユーザーを作成または取得
	uid, err := createOrGetFirebaseUserForGitHub(ctx, userInfo)
	if err != nil {
		log.Printf("ERROR: Failed to create/get Firebase user: %v\n", err)
		return map[string]interface{}{"error": "ユーザーアカウントの作成に失敗しました"}, http.StatusInternalServerError
	}

	// カスタムトークンを生成
	customToken, err := authClient.CustomToken(ctx, uid)
	if err != nil {
		log.Printf("ERROR: Failed to create custom token for UID %s: %v\n", uid, err)
		return map[string]interface{}{"error": "認証トークンの生成に失敗しました"}, http.StatusInternalServerError
	}

	log.Printf("INFO: GitHub OAuth2.0 authentication successful for UID: %s", uid)

	return map[string]interface{}{
		"message":     "GitHubアカウントでのログインが成功しました",
		"uid":         uid,
		"email":       userInfo.Email,
		"customToken": customToken,
	}, http.StatusOK
} 