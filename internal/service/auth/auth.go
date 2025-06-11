package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/LikhithMar14/BidZy/internal/store"
	"github.com/LikhithMar14/BidZy/pkg/types"
	"github.com/LikhithMar14/BidZy/pkg/utils"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

type AuthService struct {
	store             store.AuthRepository
	jwtSecret         string
	googleOauthClient *oauth2.Config
}

func NewAuthService(store store.AuthRepository, jwtSecret string, googleOauthClient *oauth2.Config) *AuthService {
	return &AuthService{
		store:             store,
		jwtSecret:         jwtSecret,
		googleOauthClient: googleOauthClient,
	}
}

// Traditional registration
func (s *AuthService) Register(ctx context.Context, req *types.CreateUserRequest) (*types.CreateUserResponse, error) {
	exists, err := s.store.GetUserByEmailAndUserName(ctx, req.Email, req.UserName)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, errors.New("user already exists with this email or username")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(ctx, req, hashedPassword)
	if err != nil {
		return nil, err
	}

	return &types.CreateUserResponse{
		User:  *user,
		Token: s.generateToken(user.UserName, user.Email, "user", user.ID),
	}, nil
}

// Traditional login
func (s *AuthService) Login(ctx context.Context, req *types.LoginRequest) (*types.LoginResponse, error) {
	user, err := s.store.GetUserByEmailAndUserName(ctx, req.UserName, req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || !utils.CheckPasswordHash(req.Password, user.Password) {
		return nil, errors.New("invalid credentials")
	}
	token := s.generateToken(user.UserName, user.Email, "user", user.ID)
	return &types.LoginResponse{Token: token}, nil
}

// JWT generation
func (s *AuthService) generateToken(userName, email, role, userID string) string {
	if role == "" {
		role = "user"
	}
	claims := &types.UserClaims{
		UserID:   userID,
		Email:    email,
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "BidZy",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now()),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   userID,
			Audience:  jwt.ClaimStrings{"BidZy"},
			ID:        userID,
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		log.Println("Error signing token:", err)
		return ""
	}
	return tokenString
}

// Google OAuth: Get login URL
func (s *AuthService) GetGoogleLoginURL(state string) string {
	return s.googleOauthClient.AuthCodeURL(state)
}

// Google OAuth: Fetch user info, auto-login or register
func (s *AuthService) GetUserInfoFromGoogle(ctx context.Context, code string) (map[string]interface{}, error) {
	token, err := s.googleOauthClient.Exchange(ctx, code)
	if err != nil {
		return nil, err
	}

	client := s.googleOauthClient.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	// Parse fields safely
	email, _ := userInfo["email"].(string)
	username, _ := userInfo["name"].(string)
	googleID, _ := userInfo["id"].(string) // use ID as password fallback

	if email == "" || username == "" || googleID == "" {
		return nil, errors.New("invalid user info from Google")
	}

	// Check if user exists
	existingUser, err := s.store.GetUserByEmailAndUserName(ctx, email, username)
	if err != nil {
		return nil, err
	}

	var user *types.User

	if existingUser == nil {
		hashedPassword, err := utils.HashPassword(googleID) 
		if err != nil {
			return nil, err
		}
		user, err = s.store.CreateUser(ctx, &types.CreateUserRequest{
			Email:    email,
			UserName: username,
			Password: googleID, 
		}, hashedPassword)
		if err != nil {
			return nil, err
		}
	} else {
		user = existingUser
	}

	return map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{	
			"user": map[string]interface{}{
				"id": user.ID,
				"email": user.Email,
				"username": user.UserName,
				"role": "user",
			},
			"token": s.generateToken(user.UserName, user.Email, "user", user.ID),
			"message": "User logged in successfully",
		},
	}, nil
}
