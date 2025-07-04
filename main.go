//go:build local

package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// corsMiddleware は、CORSヘッダーを設定し、OPTIONSリクエストを処理するミドルウェアです。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// handlePostRequest はPOSTリクエストを処理するハンドラです。
// この関数は、ミドルウェアによってラップされて呼び出されます。
func handlePostRequest(w http.ResponseWriter, r *http.Request) {
	response, statusCode := processPostRequest(r.Context(), r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// apiRouter は、HTTPメソッドに基づいてリクエストを適切なハンドラに振り分けるルーターです。
func apiRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// GETリクエストの処理: /api/time/{spaceId}
		spaceId := strings.TrimPrefix(r.URL.Path, "/api/time/")
		if spaceId == "" {
			http.Error(w, "spaceId is missing in the URL path", http.StatusBadRequest)
			return
		}
		response, statusCode := processGetRequest(r.Context(), spaceId)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)

	case http.MethodPost:
		// POSTリクエストの処理: /api/time
		optionalAuthMiddleware(http.HandlerFunc(handlePostRequest)).ServeHTTP(w, r)

	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func main() {
	apiHandler := http.HandlerFunc(apiRouter)

	// CORSミドルウェアでapiHandlerをラップし、/api/time/ パスに登録
	http.Handle("/api/time/", corsMiddleware(apiHandler))

	log.Println("Starting local server on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}