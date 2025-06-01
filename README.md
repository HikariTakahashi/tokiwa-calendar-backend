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
