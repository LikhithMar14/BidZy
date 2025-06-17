package types

import (
	"context"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type User struct {
	ID        string    `json:"id"`
	UserName  string    `json:"user_name"`
	Email     string    `json:"email"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type UserStats struct {
	AuctionsCreated     int     `json:"auctions_created"`
	TotalBids           int     `json:"total_bids"`
	TotalAmountBid      float64 `json:"total_amount_bid"`
	ActiveAuctions      int     `json:"active_auctions"`
	ParticipatedAuctions int    `json:"participated_auctions"`
	WonAuctions         int     `json:"won_auctions"`
	AvgBidAmount        float64 `json:"avg_bid_amount"`
	HighestBidPlaced    float64 `json:"highest_bid_placed"`
}


type CreateUserRequest struct {
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUserResponse struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

type GetUserResponse struct {
	User User `json:"user"`
}

type LoginRequest struct {
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type UserClaims struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

type contextKey string

const UserContextKey contextKey = "user"


func GetUserClaimsFromContext(ctx context.Context) (*UserClaims, error) {
	claims, ok := ctx.Value(UserContextKey).(*UserClaims)
	if !ok || claims == nil {
		return nil, errors.New("user claims not found in context")
	}
	return claims, nil
}

// GetUserIDFromContext extracts just the user ID.
func GetUserIDFromContext(ctx context.Context) (string, error) {
	claims, err := GetUserClaimsFromContext(ctx)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
