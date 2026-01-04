package handlers

import (
	"github.com/gin-gonic/gin"

	"pehlione.com/app/internal/http/middleware"
	"pehlione.com/app/internal/http/render"
	"pehlione.com/app/templates/pages"
)

// CompanyHandler handles the company info page
type CompanyHandler struct{}

func NewCompanyHandler() *CompanyHandler {
	return &CompanyHandler{}
}

// Get returns the company page
func (h *CompanyHandler) Get(c *gin.Context) {
	flash := middleware.GetFlash(c)
	render.Component(c, 200, pages.Company(flash))
}
