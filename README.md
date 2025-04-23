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
// フロントエンドのプロジェクトに移動(個人のいつもの開き方でOK)
cd simple-calendar-frontend

// Node.jsのパッケージ管理システムをインストール
npm install
```

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

1. GoLang を実行する

```bash
// バックエンドのターミナルで実行
go run main.go
```
注:許可求めてくるけど許可でOK

2. 開発用のサーバーを起動し、サーバーを起動する

```bash
// フロントエンドのターミナルで実行
npm run dev
```

3. 開発用のサーバーにアクセス
   基本的には http://localhost:3000: にあります。npm run dev を実行した powershell にリンクが出るのでそっちを見てください。

バックエンドは基本的に http://localhost:8080/api/calendar にあります。フロントエンドはここからデータを取得しているのでデバックの際にどうぞ。
