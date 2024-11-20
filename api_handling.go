package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"

	"github.com/thewerther/webserver/internal/auth"
	"github.com/thewerther/webserver/internal/database"
	"golang.org/x/crypto/bcrypt"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
  if err != nil {
    log.Printf("%v: %v", msg, err)
  }
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(dat)
}

func decodeRequestBody(dataStruct any, req *http.Request) error {
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&dataStruct)
	if err != nil {
    return err
	}

  return nil
}

func authenticate(req *http.Request, cfg *ApiConfig) (database.User, error) {
  accessToken, err := auth.GetBearerToken(req.Header)
  if err != nil {
    return database.User{}, err
  }

  userIdFromToken, err := auth.ValidateJWT(accessToken, cfg.JWT_Secret)
  if err != nil {
    return database.User{}, err
  }

  userExists, err := cfg.Database.GetUserById(req.Context(), userIdFromToken)
  if err != nil {
    return database.User{}, err
  }

  if reflect.DeepEqual(userExists, database.User{}) {
    return database.User{}, errors.New("User does not exist")
  }

  return userExists, nil
}

func authorize(req *http.Request, cfg *ApiConfig) (database.User, error, int) {
	loginReq := LoginRequest{}
	err := decodeRequestBody(&loginReq, req)
	if err != nil {
		return database.User{}, errors.New("Error decoding request"), http.StatusInternalServerError
	}

	userExists, err := cfg.Database.GetUserByEmail(req.Context(), loginReq.Email)
	if err != nil {
		return database.User{}, errors.New("Error querying user by email"), http.StatusInternalServerError
	}

	if reflect.DeepEqual(userExists, database.User{}) {
    return database.User{}, errors.New("User does not exist"), http.StatusBadRequest
	}

	err = bcrypt.CompareHashAndPassword([]byte(userExists.HashedPassword), []byte(loginReq.Password))
	if err != nil {
		return database.User{}, errors.New("Incorrect email or password"), http.StatusUnauthorized
	}

  return userExists, nil, 0
}
