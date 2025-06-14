package main

// firestoreCollectionName は使用するFirestoreのコレクション名を保持する
// この変数の値は、ビルドタグによってconfig_local.goかconfig_prod.goで設定される
var firestoreCollectionName string