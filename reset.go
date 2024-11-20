package main

import (
	"errors"
	"fmt"
	"net/http"
)

func (cfg *ApiConfig) resetServer(w http.ResponseWriter, req *http.Request) {
	if !cfg.IsAdmin {
		respondWithError(w, http.StatusForbidden, "", errors.New("Only allowed in dev environment!"))
		return
	}
  // reset metrics
	cfg.FileServerHits.Store(0)

	// this will also delete all chirps due to database constraints requiring
	// an existing user in the users db
	numUsersDel, err := cfg.Database.DeleteUsers(req.Context())
	if err != nil {
		fmt.Println(err)
	}
  fmt.Printf("--------------------- Cleared database ---------------------\nNum of deleted Users: %v\n------------------------------------------------------------\n", numUsersDel)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hits reset to 0\nDeleted %v users from database", numUsersDel)))
}
