package controllers

import (
	"bytes"
	"event-analytics/models"
	"event-analytics/utils"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCreateEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("User not authenticated", func(t *testing.T) {
		router := gin.Default()
		router.POST("/events", models.CreateEvent)

		req, _ := http.NewRequest(http.MethodPost, "/events", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/auth/login?error=auth_required")
	})

	t.Run("Invalid input data", func(t *testing.T) {
		router := gin.Default()
		router.POST("/events", func(c *gin.Context) {
			utils.SetUserInSession(c, &models.User{ID: 1})
			CreateEvent(c)
		})

		form := url.Values{}
		form.Add("title", "")
		form.Add("description", "")
		form.Add("start_time", "")
		form.Add("end_time", "")
		form.Add("location", "")
		form.Add("status", "")

		req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/events/new?error=Please fill all required fields correctly")
	})

	t.Run("Invalid datetime format", func(t *testing.T) {
		router := gin.Default()
		router.POST("/events", func(c *gin.Context) {
			utils.SetUserInSession(c, &models.User{ID: 1})
			CreateEvent(c)
		})

		form := url.Values{}
		form.Add("title", "Test Event")
		form.Add("description", "Test Description")
		form.Add("start_time", "invalid-datetime")
		form.Add("end_time", "invalid-datetime")
		form.Add("location", "Test Location")
		form.Add("status", "draft")

		req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/events/new?error=Invalid start datetime format")
	})

	t.Run("End datetime before start datetime", func(t *testing.T) {
		router := gin.Default()
		router.POST("/events", func(c *gin.Context) {
			utils.SetUserInSession(c, &models.User{ID: 1})
			CreateEvent(c)
		})

		form := url.Values{}
		form.Add("title", "Test Event")
		form.Add("description", "Test Description")
		form.Add("start_time", "2023-10-10T10:00")
		form.Add("end_time", "2023-10-10T09:00")
		form.Add("location", "Test Location")
		form.Add("status", "draft")

		req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/events/new?error=End datetime must be after start datetime")
	})

	t.Run("Successful event creation", func(t *testing.T) {
		router := gin.Default()
		router.POST("/events", func(c *gin.Context) {
			utils.SetUserInSession(c, &models.User{ID: 1})
			CreateEvent(c)
		})

		form := url.Values{}
		form.Add("title", "Test Event")
		form.Add("description", "Test Description")
		form.Add("start_time", "2023-10-10T10:00")
		form.Add("end_time", "2023-10-10T12:00")
		form.Add("location", "Test Location")
		form.Add("status", "draft")

		req, _ := http.NewRequest(http.MethodPost, "/events", bytes.NewBufferString(form.Encode()))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusFound, w.Code)
		assert.Contains(t, w.Header().Get("Location"), "/user/dashboard")
	})
}
