package main

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/thewerther/webserver/internal/auth"
	"github.com/thewerther/webserver/internal/database"
	"golang.org/x/crypto/bcrypt"
)

type UserCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserCreateResponse struct {
	Id        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	IsPremium bool      `json:"is_chirpy_red"`
}

type LoginRequest struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type LoginResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	IsPremium    bool      `json:"is_chirpy_red"`
}

type AuthRequest struct {
	Authorization string `json:"Authorization"`
}

type RefreshResponse struct {
	Token string `json:"token"`
}

func (cfg *ApiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	userReq := UserCreateRequest{}
	err := decodeRequestBody(&userReq, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request", err)
		return
	}

	if userReq.Password == "" {
		respondWithError(w, http.StatusBadRequest, "", errors.New("No password supplied!"))
		return
	}

	hashedPswd, err := bcrypt.GenerateFromPassword([]byte(userReq.Password), 4)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password", err)
		return
	}

	newUser, err := cfg.Database.CreateUser(req.Context(), database.CreateUserParams{
		Email:          userReq.Email,
		HashedPassword: string(hashedPswd),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user in database", err)
		return
	}
	log.Printf("Created user: %v\n", newUser)

	newUserResp := UserCreateResponse{
		Id:        newUser.ID,
		Email:     newUser.Email,
		IsPremium: newUser.IsPremium,
	}

	respondWithJSON(w, http.StatusCreated, newUserResp)
}

func (cfg *ApiConfig) loginUser(w http.ResponseWriter, req *http.Request) {
	userExists, err, statusCode := authorize(req, cfg)
	if err != nil {
		respondWithError(w, statusCode, "Error logging in user", err)
		return
	}

	signedToken, err := auth.MakeJWT(
		userExists.ID,
		cfg.JWT_Secret,
		time.Duration(60*time.Second),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating jwt token", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating refresh token", err)
		return
	}

	const expirationTimeRefreshTokenInSeconds = time.Hour * 24 * 60
	expiresAt := time.Now().UTC().Add(expirationTimeRefreshTokenInSeconds)
	_, err = cfg.Database.CreateRefreshToken(
		req.Context(),
		database.CreateRefreshTokenParams{
			Token:     refreshToken,
			UserID:    userExists.ID,
			ExpiresAt: expiresAt,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Refresh token creation failed", err)
		return
	}

	loginResp := LoginResponse{
		ID:           userExists.ID,
		Email:        userExists.Email,
		CreatedAt:    userExists.CreatedAt,
		UpdatedAt:    userExists.UpdatedAt,
		Token:        signedToken,
		RefreshToken: refreshToken,
		IsPremium:    userExists.IsPremium,
	}

	log.Println(loginResp)

	respondWithJSON(w, http.StatusOK, loginResp)
}

func (cfg *ApiConfig) refreshToken(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error trying to get refresh token from request header", err)
		return
	}

	dbRefreshToken, err := cfg.Database.GetRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Refresh token does not exist", err)
		return
	}

	if dbRefreshToken.ExpiresAt.Before(time.Now().UTC()) {
		respondWithError(w, http.StatusUnauthorized, "Refresh token expired", err)
		return
	}

	if dbRefreshToken.RevokedAt.Valid {
		respondWithError(w, http.StatusUnauthorized, "Refresh token has been revoked", err)
		return
	}

	user, err := cfg.Database.GetUserFromRefreshToken(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error querying user by refresh token", err)
		return
	}

	if reflect.DeepEqual(database.User{}, user) {
		respondWithError(w, http.StatusUnauthorized, "Invalid refresh token", err)
		return
	}

	newAccessToken, err := auth.MakeJWT(user.ID, cfg.JWT_Secret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating new acces token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, RefreshResponse{Token: newAccessToken})
}

func (cfg *ApiConfig) revokeRefreshToken(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error parsing Bearer token from request", err)
		return
	}

	err = cfg.Database.SetRefreshTokenRevokedAt(req.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error setting refresh token revoked_at", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, struct{}{})
}

func (cfg *ApiConfig) updateUser(w http.ResponseWriter, req *http.Request) {
	loginReq := LoginRequest{}
	err := decodeRequestBody(&loginReq, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request", err)
		return
	}

	userExists, err := authenticate(req, cfg)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
    return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(loginReq.Password), 4)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error hashing password from request", err)
		return
	}

	updatedUser, err := cfg.Database.UpdateUserCredentialsById(
		req.Context(),
		database.UpdateUserCredentialsByIdParams{
			Email:          loginReq.Email,
			HashedPassword: string(hashedPassword),
			ID:             userExists.ID,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error updating user credentials", err)
		return
	}

	loginResp := LoginResponse{
		ID:        updatedUser.ID,
		CreatedAt: updatedUser.CreatedAt,
		UpdatedAt: updatedUser.UpdatedAt,
		Email:     updatedUser.Email,
	}

	respondWithJSON(w, http.StatusOK, loginResp)
}
