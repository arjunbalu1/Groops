package services

import (
	"context"
	"errors"
	"os"
	"time"

	"googlemaps.github.io/maps"
)

var (
	mapsClient  *maps.Client
	ErrNoAPIKey = errors.New("GOOGLE_MAPS_API_KEY environment variable not set")
)

// InitMapsClient initializes the Google Maps client
func InitMapsClient() error {
	apiKey := os.Getenv("GOOGLE_MAPS_API_KEY")
	if apiKey == "" {
		return ErrNoAPIKey
	}

	var err error
	mapsClient, err = maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return err
	}

	return nil
}

// ValidateLocation validates and standardizes location data using the Place ID
func ValidateLocation(placeID string) (*maps.PlaceDetailsResult, error) {
	if mapsClient == nil {
		if err := InitMapsClient(); err != nil {
			return nil, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	request := &maps.PlaceDetailsRequest{
		PlaceID: placeID,
		Fields: []maps.PlaceDetailsFieldMask{
			maps.PlaceDetailsFieldMaskGeometry,
			maps.PlaceDetailsFieldMaskFormattedAddress,
			maps.PlaceDetailsFieldMaskName,
			maps.PlaceDetailsFieldMaskPlaceID,
		},
	}

	response, err := mapsClient.PlaceDetails(ctx, request)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
