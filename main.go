// main.go (クリーンな状態)

//go:build local

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	// POST /api/time
	http.HandleFunc("/api/time", func(w http.ResponseWriter, r *http.Request) {
		// CORSプリフライトリクエストに対応
		if r.Method == http.MethodOptions {
			setCORS(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		setCORS(w) // 通常のリクエストにもCORSヘッダーを設定

		// 共通のロジックを呼び出す
		response, statusCode := processPostRequest(r.Context(), r)

		// 結果をHTTPレスポンスとして返す
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	})

	// GET /api/time/{spaceId}
	http.HandleFunc("/api/time/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			setCORS(w)
			w.WriteHeader(http.StatusOK)
			return
		}
		setCORS(w)

		parts := strings.Split(r.URL.Path, "/")
		if len(parts) < 4 || parts[3] == "" {
			http.Error(w, "Invalid URL: spaceId is missing", http.StatusBadRequest)
			return
		}
		spaceId := parts[3]

		// 共通のロジックを呼び出す
		response, statusCode := processGetRequest(r.Context(), spaceId)

		// 結果をHTTPレスポンスとして返す
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	})

	fmt.Println("Starting local server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}