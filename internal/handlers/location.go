package handlers

import (
	"groops/internal/models"
	"groops/internal/services"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ValidateLocation validates a Google Place ID and returns standardized location data
func ValidateLocation(c *gin.Context) {
	placeID := c.Query("place_id")
	if placeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "place_id parameter is required"})
		return
	}

	placeDetails, err := services.ValidateLocation(placeID)
	if err != nil {
		log.Printf("Error validating location: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate location"})
		return
	}

	// Convert Google Maps API response to our Location model
	location := models.Location{
		PlaceID:          placeDetails.PlaceID,
		Name:             placeDetails.Name,
		FormattedAddress: placeDetails.FormattedAddress,
		Latitude:         placeDetails.Geometry.Location.Lat,
		Longitude:        placeDetails.Geometry.Location.Lng,
	}

	c.JSON(http.StatusOK, location)
}
