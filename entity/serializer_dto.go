package entity

import "feature-flag-2/models"

type FlagResponse struct {
	Body struct {
		Flag models.Flag `json:"flag"`
	}
}

func NewFlagResponse(flag models.Flag) *FlagResponse {
	responseFlag := &FlagResponse{}
	responseFlag.Body.Flag = flag
	return responseFlag
}

type ListOfFlagResponse struct {
	Body struct {
		Flags []models.Flag `json:"flag"`
	}
}

func NewListOfFlagResponse(flags []models.Flag) *ListOfFlagResponse {
	responseListOfFlag := &ListOfFlagResponse{}
	responseListOfFlag.Body.Flags = flags
	return responseListOfFlag
}
