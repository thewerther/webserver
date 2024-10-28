package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thewerther/webserver/internal/auth"
	"github.com/thewerther/webserver/internal/database"
)

type chirpRequest struct {
	Body string `json:"body"`
}

type chirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
  UserID    uuid.UUID `json:"user_id"`
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	chirpReq := chirpRequest{}
	err := decoder.Decode(&chirpReq)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Decoding request", err)
		return
	}

	authToken, err := auth.GetBearerToken(req.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error getting Bearer token in request header", err)
		return
	}

	userID, err := auth.ValidateJWT(authToken, cfg.JWT_Secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Error validating token from request header" ,err)
		return
	}

	if len(chirpReq.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "", errors.New("Chirp is too long!"))
		return
	}

  existingUser, err := cfg.database.GetUserByID(req.Context(), userID)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error querying User Id from database", err)
    return
  }

	newChirp, err := cfg.database.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   cleanBody(chirpReq.Body),
			UserID: userID,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating Chirp in database", err)
		return
	}

  response := chirpResponse{
    ID: newChirp.ID,
    CreatedAt: newChirp.CreatedAt,
    UpdatedAt: newChirp.UpdatedAt,
    Email: existingUser.Email,
    UserID: existingUser.ID,
    Token: authToken,
  }

	respondWithJSON(w, http.StatusCreated, response)
}

func cleanBody(body string) string {
	profaneWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
		"Kerfuffle": {},
		"Sharbert":  {},
		"Fornax":    {},
	}

	msgWords := strings.Split(body, " ")
	for idx, word := range msgWords {
		if _, exists := profaneWords[word]; exists {
			msgWords[idx] = strings.ReplaceAll(msgWords[idx], word, strings.Repeat("*", 4))
		}
	}

	return strings.Join(msgWords, " ")
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	chirps, err := cfg.database.GetChirps(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error querying chirps from database", err)
		return
	}

	respondWithJSON(w, http.StatusOK, chirps)
}

func (cfg *apiConfig) getChirpByID(w http.ResponseWriter, req *http.Request) {
	chirpId := req.PathValue("chirpId")
	id, err := uuid.Parse(chirpId)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error getting chirp id from request", err)
		return
	}

	chirp, err := cfg.database.GetChirpByID(req.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error querying chirp by id", err)
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)
}
