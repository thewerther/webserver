package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

const TEST_SECRET = "gsRBlZzXgD9nvGCWX0ba/iiIE0z/kNoa/67lv74Z50oKY6TcX/NURSb9BF+G+VoZWnLS5F7QPEbSRiayUGyMUQ=="

func createTestToken(id uuid.UUID) (string, error) {
	token, err := MakeJWT(id, TEST_SECRET, 10 * time.Second)
	if err != nil {
		return "", err
	}

  return token, nil
}

func TestValidJWT(t *testing.T) {
	id := uuid.New()
  token, err := createTestToken(id)
	if err != nil {
    t.Errorf("Test ValidJWT::MakeJWT failed with err: %v", err)
		return
	}

	returnedID, err := ValidateJWT(token, TEST_SECRET)
	if err != nil {
    t.Errorf("Test ValidJWT::ValidateJWT failed with err: %v", err)
		return
	}

	if id != returnedID {
    t.Errorf("Test ValidJWT failed: excpected ID: %v, got: %v", id, returnedID)
	}
}

func TestInvalidJWT(t *testing.T) {
  otherSecret := uuid.NewString()
	id := uuid.New()
  token, err := createTestToken(id)
	if err != nil {
    t.Errorf("Test InvalidJWT::MakeJWT failed with err: %v", err)
		return
	}

	returnedID, err := ValidateJWT(token, otherSecret)
	if err == nil {
    t.Errorf("Test InvalidJWT::ValidateJWT failed with err: %v", err)
	}

  if id == returnedID {
    t.Errorf("Test InvalidJWT failed")
  }
}

func TestValidBearerToken(t *testing.T) {
  id := uuid.New()
  header := http.Header{}
  testToken, err := createTestToken(id)
  if err != nil {
    t.Errorf("Test ValidBearerToken failed with err: %v", err)
  }
  header.Set("Authorization", "Bearer " + testToken)

  token, err := GetBearerToken(header)
  if err != nil {
    t.Errorf("Test ValidBearerToken failed with err: %v", err)
  }

  returnedId, err := ValidateJWT(token, TEST_SECRET)
  if err != nil {
    t.Errorf("Test ValidBearerToken failed with err: %v", err)
  }

  if id != returnedId {
    t.Errorf("Test ValidBearerToken failed with err: %v", err)
  }
}

func TestMakeRefreshToken(t *testing.T) {
  token, err := MakeRefreshToken()
  if err != nil {
    t.Errorf("Test MakeRefreshToken failed with err: %v", err)
  }

  if len(token) != 64 {
    t.Errorf("Test MakeRefreshToken failed because token length is not 32, actual length: %v", len(token))
  }
}
