package tests

import (
	"context"
	"errors"
	"event-analytics/config"
	"event-analytics/models"
	"time"

	// "fmt"
	// "log"
	"net/http"
	"net/http/httptest"

	// "os"

	"net/url"
	"strings"
	"testing"

	// "github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestCreateEvent_Unauthenticated(t *testing.T) {
	ClearTestData(testDB)
	req := httptest.NewRequest("POST", "/events/create", nil)
	w := httptest.NewRecorder()

	r := SetupTestRouter()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
}

func TestCreateEvent_MissingFields(t *testing.T) {
	ClearTestData(testDB)

	// Create a test user
	user := CreateTestUser(t)

	// Setup the router
	r := SetupTestRouter()

	// Create session token
	sessionToken := uuid.New().String()

	// Store session in Redis
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)

	// Create the request
	form := url.Values{}
	req := httptest.NewRequest("POST", "/events/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add session cookie to request
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/events/new?error=Please+fill+all+required+fields+correctly")
}

func TestCreateEvent_InvalidDateFormat(t *testing.T) {
	ClearTestData(testDB)

	// Create a test user
	user := CreateTestUser(t)

	// Setup the router
	r := SetupTestRouter()

	// Create session token
	sessionToken := uuid.New().String()

	// Store session in Redis
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)

	form := url.Values{
		"title":        {"Test Event"},
		"description": {"Test description"},
		"location":    {"Test Location"},
		"start_time":  {"invalid-date"},
		"end_time":    {"invalid-date"},
		"status":    {"draft"},
	}

	req := httptest.NewRequest("POST", "/events/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/events/new?error=Invalid+start+datetime+format")
}

func TestCreateEvent_NameUniqueness(t *testing.T) {
	ClearTestData(testDB)

	// Create a test user
	user := CreateTestUser(t)

	// Setup the router
	r := SetupTestRouter()

	// Create session token
	sessionToken := uuid.New().String()

	// Store session in Redis
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)

	// Create existing event
	existingEvent := &models.Event{
		Title:        	"Test Event",
		Description: 	"Existing event",
		Location:    	"Test Location",
		StartTime:   	time.Now(),
		EndTime:     	time.Now().Add(time.Hour * 2),
		CreatedBy:   	user.ID,
		Status: 		"published",
	}
	assert.NoError(t, testDB.Create(existingEvent).Error)

	// Try to create event with same name
	form := url.Values{
		"title":        {"Test Event"},
		"description": 	{"Test description"},
		"location":    	{"Test Location"},
		"start_time":  	{time.Now().Format("2006-01-02T15:04")},
		"end_time":    	{time.Now().Add(time.Hour * 2).Format("2006-01-02T15:04")},
		"status":		{"published"},
	}

	req := httptest.NewRequest("POST", "/events/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/events/new?error=Event title must be unique")
}

func TestCreateEvent_ValidEvent(t *testing.T) {
	ClearTestData(testDB)

	// Create a test user
	user := CreateTestUser(t)

	// Setup the router
	r := SetupTestRouter()

	// Create session token
	sessionToken := uuid.New().String()

	// Store session in Redis
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)

	startTime := time.Now().Add(time.Hour * 24) // Tomorrow
	endTime := startTime.Add(time.Hour * 2)     // 2 hours later

	form := url.Values{
		"title":        {"Unique Event"},
		"description": 	{"Valid description"},
		"location":    	{"Test Location"},
		"start_time":  	{startTime.Format("2006-01-02T15:04")},
		"end_time":    	{endTime.Format("2006-01-02T15:04")},
		"status":		{"published"},
	}

	req := httptest.NewRequest("POST", "/events/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/user/dashboard")

	// Verify event was created
	var event models.Event
	result := testDB.Where("title = ?", "Unique Event").First(&event)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Unique Event", event.Title)
	assert.Equal(t, user.ID, event.CreatedBy)
}

func TestCreateEvent_Success(t *testing.T) {
	ClearTestData(testDB)

	// Create a test user
	user := CreateTestUser(t)

	// Setup the router
	r := SetupTestRouter()

	// Create session token
	sessionToken := uuid.New().String()

	// Store session in Redis
	err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
	assert.NoError(t, err)

	// Create the request with valid event data
	form := url.Values{
		"title":        {"Test Event"},
		"description": 	{"Test Description"},
		"location":    	{"Test Location"},
		"start_time":  	{"2025-01-10T10:00"},
		"end_time":    	{"2025-01-10T11:00"},
		"status":    	{"published"},
	}
	req := httptest.NewRequest("POST", "/events/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	
	// Add session cookie to request
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: sessionToken,
	})

	// Create response recorder
	w := httptest.NewRecorder()

	// Serve the request
	r.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusFound, w.Code)
	assert.Contains(t, w.Header().Get("Location"), "/user/dashboard")

	// Verify event was created in database
	var event models.Event
	result := testDB.Where("title = ?", "Test Event").First(&event)
	assert.NoError(t, result.Error)
	assert.Equal(t, "Test Event", event.Title)
	assert.Equal(t, user.ID, event.CreatedBy)
}

func TestUpdateEvent_Unauthenticated(t *testing.T) {
    ClearTestData(testDB)
    req := httptest.NewRequest("POST", "/events/update/1", nil)
    w := httptest.NewRecorder()

    r := SetupTestRouter()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
}

func TestUpdateEvent_NotFound(t *testing.T) {
    ClearTestData(testDB)
    user := CreateTestUser(t)
    
    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
    assert.NoError(t, err)

    form := url.Values{
        "title":       {"Updated Event"},
        "description": {"Updated description"},
        "location":    {"Updated Location"},
        "start_time":  {"2025-01-10T10:00"},
        "end_time":    {"2025-01-10T11:00"},
        "status":      {"published"},
    }

    req := httptest.NewRequest("POST", "/events/update/999", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Event not found")
}

func TestUpdateEvent_UnauthorizedUser(t *testing.T) {
    ClearTestData(testDB)
    owner := CreateTestUser(t)
    unauthorizedUser := CreateSecondTestUser(t)

    event := &models.Event{
        Title:      "Original Event",
        Description: "Original description",
        Location:    "Original Location",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   owner.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event).Error)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, unauthorizedUser.ID.String(), 0).Err()
    assert.NoError(t, err)

    form := url.Values{
        "title":       {"Updated Event"},
        "description": {"Updated description"},
        "location":    {"Updated Location"},
        "start_time":  {"2025-01-10T10:00"},
        "end_time":    {"2025-01-10T11:00"},
        "status":      {"published"},
    }

    req := httptest.NewRequest("POST", "/events/update/"+event.ID.String(), strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Permission denied")
}

func TestUpdateEvent_DuplicateTitle(t *testing.T) {
    ClearTestData(testDB)
    user := CreateTestUser(t)

    // Create first event
    event1 := &models.Event{
        Title:      "Event One",
        Description: "First event",
        Location:    "Location One",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   user.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event1).Error)

    // Create second event
    event2 := &models.Event{
        Title:      "Event Two",
        Description: "Second event",
        Location:    "Location Two",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   user.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event2).Error)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
    assert.NoError(t, err)

    // Try to update event2 with event1's title
    form := url.Values{
        "title":       {"Event One"},
        "description": {"Updated description"},
        "location":    {"Updated Location"},
        "start_time":  {"2025-01-10T10:00"},
        "end_time":    {"2025-01-10T11:00"},
        "status":      {"published"},
    }

    req := httptest.NewRequest("POST", "/events/update/"+event2.ID.String(), strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/events/edit/"+event2.ID.String()+"?error=Event title must be unique")
}

func TestUpdateEvent_Success(t *testing.T) {
    ClearTestData(testDB)
    user := CreateTestUser(t)

    event := &models.Event{
        Title:      "Original Event",
        Description: "Original description",
        Location:    "Original Location",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   user.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event).Error)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
    assert.NoError(t, err)

    form := url.Values{
        "title":       {"Updated Event"},
        "description": {"Updated description"},
        "location":    {"Updated Location"},
        "start_time":  {"2025-01-10T10:00"},
        "end_time":    {"2025-01-10T11:00"},
        "status":      {"published"},
    }

    req := httptest.NewRequest("POST", "/events/update/"+event.ID.String(), strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard")

    // Verify event was updated
    var updatedEvent models.Event
    result := testDB.First(&updatedEvent, "id = ?", event.ID)
    assert.NoError(t, result.Error)
    assert.Equal(t, "Updated Event", updatedEvent.Title)
    assert.Equal(t, "Updated description", updatedEvent.Description)
    assert.Equal(t, "Updated Location", updatedEvent.Location)
    assert.Equal(t, "published", updatedEvent.Status)
    assert.NotNil(t, updatedEvent.PublishedDate)
}

func TestDeleteEvent_Unauthenticated(t *testing.T) {
    ClearTestData(testDB)
    req := httptest.NewRequest("POST", "/events/delete/1", nil)
    w := httptest.NewRecorder()

    r := SetupTestRouter()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
}

func TestDeleteEvent_NotFound(t *testing.T) {
    ClearTestData(testDB)
    user := CreateTestUser(t)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
    assert.NoError(t, err)

    req := httptest.NewRequest("POST", "/events/delete/999", nil)
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
	// Event not found
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Error fetching event")
}

func TestDeleteEvent_UnauthorizedUser(t *testing.T) {
    ClearTestData(testDB)
    owner := CreateTestUser(t)
    unauthorizedUser := CreateSecondTestUser(t)

    event := &models.Event{
        Title:      "Test Event",
        Description: "Test description",
        Location:    "Test Location",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   owner.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event).Error)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, unauthorizedUser.ID.String(), 0).Err()
    assert.NoError(t, err)

    req := httptest.NewRequest("POST", "/events/delete/"+event.ID.String(), nil)
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard?error=Permission denied")
}

func TestDeleteEvent_Success(t *testing.T) {
    ClearTestData(testDB)
    user := CreateTestUser(t)

    event := &models.Event{
        Title:      "Test Event",
        Description: "Test description",
        Location:    "Test Location",
        StartTime:   time.Now(),
        EndTime:     time.Now().Add(time.Hour * 2),
        CreatedBy:   user.ID,
        Status:      "draft",
    }
    assert.NoError(t, testDB.Create(event).Error)

    r := SetupTestRouter()
    sessionToken := uuid.New().String()
    err := config.RedisClient.Set(context.Background(), sessionToken, user.ID.String(), 0).Err()
    assert.NoError(t, err)

    req := httptest.NewRequest("POST", "/events/delete/"+event.ID.String(), nil)
    req.AddCookie(&http.Cookie{
        Name:  "session_token",
        Value: sessionToken,
    })
    w := httptest.NewRecorder()

    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code)
    assert.Contains(t, w.Header().Get("Location"), "/user/dashboard")

    // Verify event was deleted
    var deletedEvent models.Event
    result := testDB.First(&deletedEvent, "id = ?", event.ID)
    assert.Error(t, result.Error)
    assert.True(t, errors.Is(result.Error, gorm.ErrRecordNotFound))
}