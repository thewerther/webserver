package main

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/google/uuid"
	"github.com/thewerther/webserver/internal/auth"
	"github.com/thewerther/webserver/internal/database"
)

type UserPremiumUpgrade struct {
	Event string `json:"event"`
	Data  struct {
		UserID uuid.UUID `json:"user_id"`
	} `json:"data"`
}

const userUpgradedEvent = "user.upgraded"

func (cfg *ApiConfig) paymentHandler(w http.ResponseWriter, req *http.Request) {
  apiKey, err := auth.GetAPIKey(req.Header)
  if err != nil {
    respondWithError(w, http.StatusUnauthorized, "Error getting api key from request header", err)
    return
  }

  if apiKey != cfg.PolkaKey {
    respondWithError(w, http.StatusUnauthorized, "Unauthorization error", errors.New("Wrong polka API Key"))
    return
  }

	webhookReq := UserPremiumUpgrade{}
	err = decodeRequestBody(&webhookReq, req)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error decoding request", err)
		return
	}

  if webhookReq.Event != userUpgradedEvent {
    respondWithError(w, http.StatusNoContent, "Error handling webhook event", errors.New(fmt.Sprintf("%v is not a valid event", webhookReq.Event)))
    return
  }

  userExists, err := cfg.Database.GetUserById(req.Context(), webhookReq.Data.UserID)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error querying user by id", err)
    return
  }

  if reflect.DeepEqual(userExists, database.User{}) {
    respondWithError(w, http.StatusNotFound, "", errors.New("Invalid user"))
    return
  }

  err = cfg.Database.SetUserPremium(req.Context(), userExists.ID)
  if err != nil {
    respondWithError(w, http.StatusInternalServerError, "Error updating user to premium", err)
    return
  }

  respondWithJSON(w, http.StatusNoContent, struct{}{})
}
