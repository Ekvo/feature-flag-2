package service

import (
	"context"
	"feature-flag-2/entity"
	"feature-flag-2/models"
	"feature-flag-2/repository/db"
	"feature-flag-2/utils"
)

type ServiceFlag struct {
	repoDB *db.RepoFlagDB
}

func NewServiceFlag(db *db.RepoFlagDB) *ServiceFlag {
	return &ServiceFlag{repoDB: db}
}

func (sf *ServiceFlag) CreateNewFlag(
	ctx context.Context,
	newFlag models.Flag,
) (*entity.FlagResponse, error) {
	if err := sf.repoDB.CreateFlag(ctx, newFlag); err != nil {
		return nil, err
	}
	return entity.NewFlagResponse(newFlag), nil
}

func (sf *ServiceFlag) GetFlagByName(
	ctx context.Context,
	flagName string,
) (*entity.FlagResponse, error) {
	flag, err := sf.repoDB.GetFlagByName(ctx, flagName)
	if err != nil {
		return nil, err
	}
	return entity.NewFlagResponse(flag), nil
}

func (sf *ServiceFlag) UpdateFlag(
	ctx context.Context,
	newFlag models.Flag,
) (*entity.FlagResponse, error) {
	if err := sf.repoDB.UpdateFlag(ctx, newFlag); err != nil {
		return nil, err
	}
	return entity.NewFlagResponse(newFlag), nil
}

func (sf *ServiceFlag) DeleteFlag(
	ctx context.Context,
	flagName string,
) error {
	if err := sf.repoDB.DeleteFlag(ctx, flagName); err != nil {
		return err
	}
	return nil
}

func (sf *ServiceFlag) RetrieveListOfAllFlags(ctx context.Context) (*entity.ListOfFlagResponse, error) {
	listOfFlags, err := sf.repoDB.ListOfAllFlags(ctx)
	if err != nil {
		return nil, err
	}
	return entity.NewListOfFlagResponse(listOfFlags), nil
}

func (sf *ServiceFlag) RetrieveListOfFlagsByNames(
	ctx context.Context,
	flagNames []string,
) (*entity.ListOfFlagResponse, error) {
	listOfFlags, err := sf.repoDB.ListOfFlagByNames(ctx, utils.UniqueWords(flagNames))
	if err != nil {
		return nil, err
	}
	return entity.NewListOfFlagResponse(listOfFlags), nil
}
