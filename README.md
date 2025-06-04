## 前提

動作には以上の二つのレポジトリのインストールが必要です

**フロントエンド**:https://github.com/HikariTakahashi/simple-calendar-frontend

**バックエンド**:https://github.com/HikariTakahashi/simple-calendar-backend ← いまここ

## 起動準備(フロントエンド)

1. レポジトリのクローン

```bash
git clone https://github.com/HikariTakahashi/simple-calendar-frontend.git
```

2. 必要なもののインストール

```bash
// フロントエンドのプロジェクトに移動(いつもの開き方でOK)
cd simple-calendar-frontend

// Node.jsのパッケージ管理システムをインストール
npm install
```

注:vue-drumroll-datetime-pickerの利用は廃止されました。特にアンインストール等は必要ないです。

## 起動準備(バックエンド)

1. レポジトリのクローン

```bash
git clone https://github.com/HikariTakahashi/simple-calendar-backend.git
```

2. バックエンドのプロジェクトに移動(個人のいつもの開き方で OK)

```bash
cd simple-calendar-backend
```

3. Firebaseへの接続準備
バックエンドサーバーはデータの永続化にFirebase (Firestore) を利用しています。そのため、以下の準備が必要です。

- GOOGLE_APPLICATION_CREDENTIALS_JSON 環境変数の設定: 
Firebaseのサービスアカウントキー(JSONファイル)はチームメンバーが管理しています。開発に必要なサービスアカウントキーの情報を受け取り、GOOGLE_APPLICATION_CREDENTIALS_JSON という名前の環境変数に設定してください。この環境変数を設定することで、Firebaseへの認証を行えるようになります。
- Firestoreへのアクセス権限: 
開発に使用するGoogleアカウントが、対象のFirestoreデータベースへの読み書き権限を持っていることが必要です。必要に応じて、Firebaseプロジェクトの管理者に招待を依頼してください。

これらの設定が正しく行われていない場合、バックエンドサーバーは起動時またはデータアクセス時にエラーを発生させます。

## 開発サーバーの起動

1. バックエンド起動（Go 言語）

```bash
// バックエンドのターミナルで実行
go run .
```

注: 以前は `go run main.go` でしたが、ファイル分割により `go run .`に変更されました。プロジェクト内のすべての .go ファイルがビルド対象になります。

2. フロントエンド起動（Next.js）

```bash
// フロントエンドのターミナルで実行
npm run dev
```

3. 開発用のサーバーにアクセス

基本的には http://localhost:3000 にあります。`npm run dev` を実行した powershell にリンクが出るのでそっちを見てください。

バックエンドは基本的に http://localhost:8080/api/calendar にあります。フロントエンドはここからデータを取得しているのでデバックの際にどうぞ。

## Thunder Client を使ったバックエンドのテスト

Thunder Client は VSCode の拡張機能で、フロントを立てずにバックエンド単体でリクエストのテストが可能です。

### 導入方法

1. VSCode の「拡張機能」から Thunder Client を検索してインストール

### テスト方法（GET リクエスト）

1. Thunder Client を開く

2. `GET`を選択し、URL に `http://localhost:8080/api/calendar?year=2024&month=5&move=`（テスト用クエリ）などを入力

3. 「Send」ボタンを押すと、右側にレスポンスが表示される

4. 表示されたレスポンスより、挙動やステータスコードを確認

### POST リクエストでのテスト（※未実装予定）

将来的にバックエンドで POST リクエストを受け取る場合、以下のようにテスト可能。

1. Thunder Client でメソッドを POST に設定

2. `Body` タブで `JSON` を選び、以下のように入力

```bash
  ｛
　"title": "会議",
  "date": "2024-05-10"
｝
```

3. `Send` を押して、レスポンスやエラーを確認

### ステータスコードの意味

- 200 OK：バックエンドが正常動作

- 400 Bad Request：クエリの入力ミスなど → フロントエンドの問題

- 500 Internal Server Error：サーバー内部のエラー → バックエンドの問題

- 404 Not Found：API エンドポイントが存在しない

- 403 Forbidden：管理者権限が必要

- 401 Unauthorized：認証が必要
