package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/golang-jwt/jwt/v5"
)

func (app *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":false,
				"message":"Unauthorized",
				"data":nil,
			})
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":false,
				"message":"Unauthorized",
				"data":nil,
			})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, &types.UserClaims{}, func(token *jwt.Token) (interface{}, error) {
			return []byte(app.Config.JwtSecret), nil
		})

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":false,
				"message":"Unauthorized",
				"data":nil,
			})
			return
		}

		if !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":false,
				"message":"Unauthorized",
				"data":nil,
			})
			return
		}

		claims, ok := token.Claims.(*types.UserClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success":false,
				"message":"Unauthorized",
				"data":nil,
			})
			return
		}

		ctx := context.WithValue(r.Context(), types.UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}