package render

import (
	"html/template"
	"net/http"
	"github.com/gin-gonic/gin"
)

var templates *template.Template

// InitTemplates loads all templates with layout support
func InitTemplates(funcMap template.FuncMap) error {
	var err error
	templates, err = template.New("").Funcs(funcMap).ParseGlob("templates/**/*.html")
	if err != nil {
		return err
	}
	templates, err = templates.ParseGlob("templates/*.html")
	return err
}

// Render renders a template with base layout
func Render(c *gin.Context, data gin.H, templateName string) {
	c.HTML(http.StatusOK, templateName, data)
}

// RenderWithLayout renders template with specified layout
func RenderWithLayout(c *gin.Context, data gin.H, templateName, layoutName string) {
	if templates == nil {
		c.String(http.StatusInternalServerError, "Templates not initialized")
		return
	}
	
	err := templates.ExecuteTemplate(c.Writer, layoutName, data)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
	}
}
