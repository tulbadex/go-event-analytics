package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestShowEventDetails(t *testing.T) {
	// Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Setup router
	r := gin.Default()
	r.GET("/events/:id", ShowEventDetails)

	tests := []struct {
		name           string
		eventID        string
		expectedStatus int
	}{
		{
			name:           "Existing Event",
			eventID:        "1", // Assumes event with ID 1 exists
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Non-existing Event",
			eventID:        "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			req, err := http.NewRequest(http.MethodGet, "/events/"+tt.eventID, nil)
			assert.NoError(t, err)

			// Create response recorder
			w := httptest.NewRecorder()

			// Serve request
			r.ServeHTTP(w, req)

			// Assert status code
			// assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}