package render

import (
  "net/http"
  "github.com/gin-gonic/gin"
)

func Render(c *gin.Context, data gin.H, templateName string) {
  c.HTML(http.StatusOK, templateName, data)
}