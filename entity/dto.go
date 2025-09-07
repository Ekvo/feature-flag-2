package entity

import "feature-flag-2/models"

type FlagResponse struct {
	Body struct {
		Flag models.Flag `json:"flag"`
	}
}
type ListOfFlagResponse struct {
	Body struct {
		Flags []models.Flag `json:"flag"`
	}
}
