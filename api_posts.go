package main

import (
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/thewerther/webserver/internal/database"
)

type ChirpRequest struct {
	Body string `json:"body"`
}

type ChirpResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	UserID    uuid.UUID `json:"user_id"`
	Body      string    `json:"body"`
}

func (cfg *ApiConfig) createChirp(w http.ResponseWriter, req *http.Request) {
	chirpReq := ChirpRequest{}
	err := decodeRequestBody(&chirpReq, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error Decoding request", err)
		return
	}

	userExists, err := authenticate(req, cfg)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	if len(chirpReq.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "", errors.New("Chirp is too long!"))
		return
	}

	newChirp, err := cfg.Database.CreateChirp(
		req.Context(),
		database.CreateChirpParams{
			Body:   cleanBody(chirpReq.Body),
			UserID: userExists.ID,
		})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error creating Chirp in database", err)
		return
	}

	response := ChirpResponse{
		ID:        newChirp.ID,
		CreatedAt: newChirp.CreatedAt,
		UpdatedAt: newChirp.UpdatedAt,
		Email:     userExists.Email,
		UserID:    userExists.ID,
		Body:      newChirp.Body,
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

func (cfg *ApiConfig) getChirps(w http.ResponseWriter, req *http.Request) {
	dbChirps, err := cfg.Database.GetChirps(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error querying chirps by user id from database", err)
		return
	}

  authorID := uuid.Nil
	authorIDParam := req.URL.Query().Get("author_id")
	if authorIDParam != "" {
		authorID, err = uuid.Parse(authorIDParam)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Error parsing user id fro mrequest", err)
			return
		}
	}

	// default is "asc"
	sortDir := "asc"
	sortParam := req.URL.Query().Get("sort")
	if sortParam == "desc" {
		sortDir = "desc"
	}

	sortedChirps := []ChirpResponse{}
	for _, chirp := range dbChirps {
		if authorID != uuid.Nil && chirp.UserID != authorID {
			continue
		}

		sortedChirps = append(sortedChirps, ChirpResponse{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			UserID:    chirp.UserID,
			Body:      chirp.Body,
		})
	}

	sort.Slice(sortedChirps, func(i, j int) bool {
		if sortDir == "desc" {
			return sortedChirps[i].CreatedAt.After(sortedChirps[j].CreatedAt)
		}
		return sortedChirps[i].CreatedAt.Before(sortedChirps[j].CreatedAt)
	})

	respondWithJSON(w, http.StatusOK, sortedChirps)
}

func (cfg *ApiConfig) getChirpByID(w http.ResponseWriter, req *http.Request) {
	chirpID := req.PathValue("chirpID")
	id, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error getting chirp id from request", err)
		return
	}

	chirp, err := cfg.Database.GetChirpByID(req.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error querying chirp by id", err)
		return
	}

	respondWithJSON(w, http.StatusOK, chirp)
}

func (cfg *ApiConfig) deleteChirpByID(w http.ResponseWriter, req *http.Request) {
	userExists, err := authenticate(req, cfg)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	chirpID := req.PathValue("chirpID")
	id, err := uuid.Parse(chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error getting chirp id from request", err)
		return
	}

	chirpExists, err := cfg.Database.GetChirpByID(req.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Error querying chirp by id", err)
		return
	}

	if chirpExists.UserID != userExists.ID {
		respondWithError(w, http.StatusForbidden, "Cannot delete another users chirp", err)
		return
	}

	err = cfg.Database.DeleteChirpByID(req.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp from database", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, struct{}{})
}
