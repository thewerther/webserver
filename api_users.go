package main

import (
	"encoding/json"
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

type userCreateRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userCreateResponse struct {
	Id    uuid.UUID `json:"id"`
	Email string    `json:"email"`
}

type loginRequest struct {
	Password         string `json:"password"`
	Email            string `json:"email"`
}

type loginResponse struct {
	ID           uuid.UUID `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
}

type authRequest struct {
	Authorization string `json:"Authorization"`
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	userReq := userCreateRequest{}
	err := decoder.Decode(&userReq)
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

	newUser, err := cfg.database.CreateUser(req.Context(), database.CreateUserParams{
		Email:          userReq.Email,
		HashedPassword: string(hashedPswd),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating user in database", err)
		return
	}
	log.Printf("Created user: %v\n", newUser)

	newUserResp := userCreateResponse{
		Id:    newUser.ID,
		Email: newUser.Email,
	}

	respondWithJSON(w, http.StatusCreated, newUserResp)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	loginReq := loginRequest{}
	err := decoder.Decode(&loginReq)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request", err)
		return
	}

	userExists, err := cfg.database.GetUserByEmail(req.Context(), loginReq.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error querying user by email", err)
		return
	}

	log.Printf("loginUser: Found user with email %v in database, id: %v", userExists.Email, userExists.ID)

	if reflect.DeepEqual(userExists, database.User{}) {
		respondWithError(w, http.StatusBadRequest, "User does not exist.", err)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(userExists.HashedPassword), []byte(loginReq.Password))
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Incorrect email or password", err)
		return
	}

	signedToken, err := auth.MakeJWT(
		userExists.ID,
		cfg.JWT_Secret,
		time.Duration(60 * time.Second),
	)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating jwt token", err)
		return
	}

	loginResp := loginResponse{
		ID:        userExists.ID,
		Email:     userExists.Email,
		CreatedAt: userExists.CreatedAt,
		UpdatedAt: userExists.UpdatedAt,
		Token:     signedToken,
	}

	respondWithJSON(w, http.StatusOK, loginResp)
}

/*
func (cfg *apiConfig) updateUser(w http.ResponseWriter, req *http.Request) {
	headerAuth := req.Header.Get("Authorization")
	if headerAuth == "" {
		respondWithError(w, http.StatusUnauthorized, "No authorization token provided!")
		return
	}

	authToken := strings.TrimPrefix(headerAuth, "Bearer ")
	// wrong
	if authToken == headerAuth {
		respondWithError(w, http.StatusUnauthorized, "No authorization token provided!")
		return
	}
}
*/
