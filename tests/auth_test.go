package tests

import (
	"context"
	"event-analytics/config"
	"event-analytics/models"
	"io/ioutil"
	// "log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
)

func TestMain(m *testing.M) {
	TestingSuite(m)
}

func TestSuccessfulRegistration(t *testing.T) {
    // ClearTestData(testDB)

    // Mocking the template directory
    gin.DefaultWriter = ioutil.Discard // Disable Gin's default logger output during tests
    r := SetupTestRouter()

    // Create a test role in the database
    role := models.Role{Name: "user"}
    testDB.Create(&role)

    // Test case for successful registration
    form := url.Values{
        "username": {"newuser"},
        "email":    {"newuser@example.com"},
        "password": {"Password@1"},
    }

    req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(form.Encode()))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    // Assert on status code
    assert.Equal(t, http.StatusOK, w.Code)

    // Assert on the HTML response
    body := w.Body.String()
    assert.Contains(t, body, "User created successfully")
    assert.Contains(t, body, "A verification email has been sent to your email address")
}

func TestFailedRegistration(t *testing.T) {
    // ClearTestData(testDB)

    r := SetupTestRouter()

    // Test case for invalid email
    form := url.Values{
        "username": {"newuser"},
        "email":    {"invalid-email"},
        "password": {"short"},
    }

    req := httptest.NewRequest("POST", "/auth/register", strings.NewReader(form.Encode()))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    // Assert on status code
    assert.Equal(t, http.StatusBadRequest, w.Code)

    // Assert on the HTML response
    body := w.Body.String()
    assert.Contains(t, body, "Invalid input")
    assert.Contains(t, body, "Register")
}

func TestSuccessfulLogin(t *testing.T) {
    // ClearTestData(testDB)

    r := SetupTestRouter()

    // Pre-register a user
    password := "Password@1"
    user := models.User{
        Username:   "testuser",
        Email:      "testuser@example.com",
        Password:   password,
        IsVerified: true, // Ensure user is verified
    }
	user.HashPassword()
    testDB.Create(&user)

    // Test case for successful login
    form := url.Values{
        "identifier": {"testuser@example.com"}, // Can be email or username
        "password":   {password},
    }

    req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusFound, w.Code) // Expecting a redirect to /user/dashboard
    location := w.Header().Get("Location")
    assert.Equal(t, "/user/dashboard", location) // Ensure correct redirection
}

func TestFailedLoginInvalidCredentials(t *testing.T) {
    // ClearTestData(testDB)

    r := SetupTestRouter()

    // Pre-register a user
    password := "Password@1"
    user := models.User{
        Username:   "testuser",
        Email:      "testuser@example.com",
        Password:   password,
        IsVerified: true, // Ensure user is verified
    }
	user.HashPassword()
    testDB.Create(&user)

    // Test case for login with invalid credentials
    form := url.Values{
        "identifier": {"testuser@example.com"},
        "password":   {"WrongPassword"},
    }

    req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    w := httptest.NewRecorder()
    r.ServeHTTP(w, req)

    assert.Equal(t, http.StatusUnauthorized, w.Code) // Expecting Unauthorized status
    assert.Contains(t, w.Body.String(), "Invalid credentials") // Check error message
}

func TestLogout(t *testing.T) {
	ClearTestData(testDB)

	// Step 1: Register and verify user
	password := "Password@1"
	user := models.User{
		Username:   "testuser",
		Email:      "testuser@example.com",
		Password:   password,
		IsVerified: true,
	}
	user.HashPassword()
	testDB.Create(&user)

	// Step 2: Log in to create a session token
	r := SetupTestRouter()

	form := url.Values{
		"identifier": {"testuser@example.com"},
		"password":   {password},
	}

	loginReq := httptest.NewRequest("POST", "/auth/login", strings.NewReader(form.Encode()))
	loginReq.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	loginRes := httptest.NewRecorder()
	r.ServeHTTP(loginRes, loginReq)

	assert.Equal(t, http.StatusFound, loginRes.Code)

	// Capture the session token from the login response cookie
	cookies := loginRes.Result().Cookies()
	var sessionToken string
	for _, cookie := range cookies {
		if cookie.Name == "session_token" {
			sessionToken = cookie.Value
			break
		}
	}
	assert.NotEmpty(t, sessionToken, "Session token should not be empty after login")

	// Verify the session token exists in Redis
	storedValue, err := config.RedisClient.Get(context.Background(), sessionToken).Result()
	assert.NoError(t, err, "Session token should exist in Redis")
	assert.Equal(t, storedValue, user.ID.String(), "Stored user ID should match")

	// Step 3: Log out
    logoutReq := httptest.NewRequest("POST", "/user/logout", nil) // Changed from GET /auth/logout to POST /user/logout
    logoutReq.AddCookie(&http.Cookie{Name: "session_token", Value: sessionToken})
    logoutRes := httptest.NewRecorder()
    r.ServeHTTP(logoutRes, logoutReq)

    assert.Equal(t, http.StatusFound, logoutRes.Code, "Logout should redirect to login page")

    // Verify the session token is deleted from Redis
    _, err = config.RedisClient.Get(context.Background(), sessionToken).Result()
    assert.Error(t, err, "Session token should be deleted from Redis")

    // Verify the session token cookie is cleared
    logoutCookies := logoutRes.Result().Cookies()
    var clearedSessionToken string
    for _, cookie := range logoutCookies {
        if cookie.Name == "session_token" {
            clearedSessionToken = cookie.Value
            break
        }
    }
    assert.Empty(t, clearedSessionToken, "Session token cookie should be cleared after logout")
    
    // Check if redirected to login page
    assert.Equal(t, "/auth/login", logoutRes.Header().Get("Location"), "Should redirect to login page")
}