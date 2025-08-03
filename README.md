# ToKiWa-calendar

## 概要
このプロジェクトは、フロントエンドとバックエンドで構成されるカレンダーアプリケーションです。

### リポジトリ
動作には、以下の二つのリポジトリが必要です。

* **フロントエンド**: [https://github.com/HikariTakahashi/simple-calendar-frontend](https://github.com/HikariTakahashi/simple-calendar-frontend)
* **バックエンド**: [https://github.com/HikariTakahashi/simple-calendar-backend](https://github.com/HikariTakahashi/simple-calendar-backend) (このリポジトリ)

---

## 重要なドキュメント

この`README.md`は、開発環境をセットアップし、アプリケーションを起動するためのクイックスタートガイドです。

APIのエンドポイント仕様、各ファイルの責任範囲、詳細なテスト手順など、**プロジェクトの技術的な仕様はすべて以下のドキュメントに記載されています。** 開発を始める前に必ずご一読ください。

> **参照先**: `ToKiWa-calendar バックエンド仕様書` (チーム内で共有)

---

## 1. ローカル開発環境のセットアップ

### 1.1. バックエンド (このリポジトリ)
#### ① レポジトリのクローン
```bash
git clone [https://github.com/HikariTakahashi/simple-calendar-backend.git](https://github.com/HikariTakahashi/simple-calendar-backend.git)
cd simple-calendar-backend
```

#### ② 環境変数の設定
プロジェクトのルートに `.env` ファイルを作成し、Firebase管理者から共有された認証情報を設定します。

```env
# Firebaseのサービスアカウントキー(JSON)の"内容"をシングルクォートで囲って貼り付け
GOOGLE_APPLICATION_CREDENTIALS_JSON='{ ... }'

# FirebaseプロジェクトのWeb APIキー
FIREBASE_API_KEY="AIzaSy..."

# AES暗号化・復号化で使用するキー (現在は未使用)
ENCRYPTION_KEY="your-secret-encryption-key-32-chars-long!"
```
> **Note**: これらの設定が正しく行われていない場合、サーバーは起動時またはデータアクセス時にエラーを発生させます。

### 1.2. フロントエンド
#### ① レポジトリのクローン
```bash
git clone [https://github.com/HikariTakahashi/simple-calendar-frontend.git](https://github.com/HikariTakahashi/simple-calendar-frontend.git)
cd simple-calendar-frontend
```
#### ② 依存パッケージのインストール
```bash
npm install
```

---

## 2. 開発サーバーの起動

### 2.1. バックエンドサーバー (Go)
バックエンドのプロジェクトディレクトリで、以下のコマンドを実行します。

```bash
go run --tags=local .
```
* `--tags=local` フラグは、ローカル開発用の設定を有効にするために**必須**です。
* サーバーは `http://localhost:8080` で起動します。

### 2.2. フロントエンドサーバー (Nuxt.js)
フロントエンドのプロジェクトディレクトリで、以下のコマンドを実行します。

```bash
npm run dev
```
* 開発用のサーバーが `http://localhost:3000` で起動します。

---

## 3. 本番環境へのデプロイ (AWS Lambda)
本番環境であるAWS Lambdaへアプリケーションをデプロイする手順です。

### 3.1. 実行ファイルのビルド
Lambda上で動作する実行ファイル(`bootstrap`)を生成します。

* **PowerShell (Windows) の場合**
    ```powershell
    $env:GOOS="linux"; $env:GOARCH="arm64"; go build -tags lambda.norpc -o bootstrap .
    ```
* **Mac / Linux の場合**
    ```bash
    GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap .
    ```

### 3.2. Zip化
生成された`bootstrap`ファイルをZip形式で圧縮します。

* **PowerShell (Windows) の場合**
    ```powershell
    Compress-Archive -Path .\bootstrap -DestinationPath .\bootstrap.zip -Force
    ```
* **Mac / Linux の場合**
    ```bash
    zip bootstrap.zip bootstrap
    ```

### 3.3. AWS Lambdaへのアップロード
1.  AWS マネジメントコンソールで対象のLambda関数を開きます。
2.  「コードソース」セクションから「アップロード元」>「.zipファイル」を選択します。
3.  作成した`bootstrap.zip`をアップロードします。
4.  デプロイ完了後、「テスト」タブからテストイベントを実行し、成功することを確認します。

---

## 4. APIテストについて
VSCodeの拡張機能「Thunder Client」を使用することで、フロントエンドを介さずにバックエンドAPIの動作を直接テストできます。

各エンドポイントの具体的なテスト手順（リクエストボディ、ヘッダー、結果の判断など）については、**バックエンド仕様書**の「APIテストガイド」の章を詳しく参照してください。

### ステータスコードの基本的な切り分け
* `2xx` (成功): バックエンドは正常にリクエストを処理しています。
* `4xx` (クライアントエラー): リクエスト内容に問題があります (例: 必須項目不足、認証エラー)。多くの場合、フロントエンド側の問題です。
* `5xx` (サーバーエラー): サーバー内部で予期せぬエラーが発生しました。バックエンド側のコードやインフラの問題です。
