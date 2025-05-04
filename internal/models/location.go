package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Location represents a location with Google Maps data
type Location struct {
	PlaceID          string  `json:"place_id" binding:"required"`
	Name             string  `json:"name"`
	FormattedAddress string  `json:"formatted_address" binding:"required"`
	Latitude         float64 `json:"latitude" binding:"required"`
	Longitude        float64 `json:"longitude" binding:"required"`
}

// Implement driver.Valuer for JSONB storage
func (l Location) Value() (driver.Value, error) {
	return json.Marshal(l)
}

// Implement sql.Scanner for JSONB retrieval
func (l *Location) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal Location: %v", value)
	}
	return json.Unmarshal(bytes, l)
}
