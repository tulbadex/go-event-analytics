package tests

import (
	"context"
	"event-analytics/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestShowNewEvent_Unauthenticated(t *testing.T) {
	r := SetupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/events/new", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/auth/login")
}

func TestShowEventDetails_Unauthenticated(t *testing.T) {
	r := SetupTestRouter()
	
	req := httptest.NewRequest(http.MethodGet, "/events/1", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/auth/login")
}

func TestShowEventDetails_EventNotFound(t *testing.T) {
	// Create test user
	ClearTestData(testDB)
	user := CreateTestUser(t)
	
	r := SetupTestRouter()
	
	// Create session
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/999", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestShowEventDetails_Success(t *testing.T) {
	ClearTestData(testDB)
	// Create test user
	user := CreateTestUser(t)
	
	// Create test event
	event := CreateTestEvent(t, user.ID)
	
	r := SetupTestRouter()
	
	// Create session
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/"+event.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowCreateEventPage_Unauthenticated(t *testing.T) {
	ClearTestData(testDB)
	
	r := SetupTestRouter()
	
	req := httptest.NewRequest(http.MethodGet, "/events/new", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
}

func TestShowCreateEventPage_Success(t *testing.T) {
	ClearTestData(testDB)
	user := CreateTestUser(t)
	
	r := SetupTestRouter()
	
	// Create session
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/new", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestShowEditEventPage_Unauthenticated(t *testing.T) {
	r := SetupTestRouter()
	
	req := httptest.NewRequest(http.MethodGet, "/events/edit/1", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
}

func TestShowEditEventPage_EventNotFound(t *testing.T) {
	ClearTestData(testDB)
	user := CreateTestUser(t)
	
	r := SetupTestRouter()
	
	// Create session
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/edit/999", nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Event not found")
}

func TestShowEditEventPage_UnauthorizedUser(t *testing.T) {

	ClearTestData(testDB)
	owner := CreateTestUser(t)
	unauthorizedUser := CreateSecondTestUser(t)
	
	r := SetupTestRouter()

	event := CreateTestEvent(t, owner.ID)
	
	// Create session for unauthorized user
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, unauthorizedUser.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/edit/"+event.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Permission denied")
}

func TestShowEditEventPage_Success(t *testing.T) {
	ClearTestData(testDB)
	user := CreateTestUser(t)
	event := CreateTestEvent(t, user.ID)
	
	r := SetupTestRouter()
	
	// Create session
	sessionToken := uuid.New().String()
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)
	
	req := httptest.NewRequest(http.MethodGet, "/events/edit/"+event.ID.String(), nil)
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}