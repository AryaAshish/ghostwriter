package handlers

import (
	"net/http"
	"strconv"

	"github.com/ashisharyan/ghostwriter-prompt-engine/models"
	"github.com/ashisharyan/ghostwriter-prompt-engine/services"
	"github.com/gin-gonic/gin"
)

type PersonaHandler struct {
	PersonaService services.PersonaService
}

func NewPersonaHandler(personaService services.PersonaService) *PersonaHandler {
	return &PersonaHandler{PersonaService: personaService}
}

func (h *PersonaHandler) GetPersona(c *gin.Context) {
	creatorID, err := parseCreatorID(c.Param("creator_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	persona, err := h.PersonaService.GetPersona(creatorID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Persona not found"})
		return
	}
	c.JSON(http.StatusOK, persona)
}

func (h *PersonaHandler) UpdatePersona(c *gin.Context) {
	creatorID, err := parseCreatorID(c.Param("creator_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req models.UpdatePersonaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	persona, err := h.PersonaService.UpdatePersona(creatorID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, persona)
}

func (h *PersonaHandler) ReanalyzeVoice(c *gin.Context) {
	creatorID, err := parseCreatorID(c.Param("creator_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var req models.ReanalyzeVoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON: " + err.Error()})
		return
	}
	persona, err := h.PersonaService.ReanalyzeVoice(creatorID, req.WritingSamples)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, persona)
}

func parseCreatorID(raw string) (uint, error) {
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(id), nil
}
