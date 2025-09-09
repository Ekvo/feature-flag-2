package service

import (
	"context"
	"errors"
	"feature-flag-2/entity"
	"feature-flag-2/models"
	"feature-flag-2/repository/cache"
	"feature-flag-2/repository/db"
	"fmt"
	"strings"
)

var (
	ErrServiceInternalError = errors.New("internal error")

	ErrServiceNotFound = errors.New("not found")

	ErrServiceUnknownFlag = errors.New("unknown flag")

	ErrServiceUpdateFlagDataCreatedNotEqual = errors.New("update data flag created-at or created_user not equal")
)

type ServiceFlag struct {
	repoDB    *db.RepoFlagDB
	repoCache *cache.RepoCacheFlag
}

func NewServiceFlag(db *db.RepoFlagDB, cache *cache.RepoCacheFlag) *ServiceFlag {
	return &ServiceFlag{repoDB: db, repoCache: cache}
}

func (sf *ServiceFlag) CreateNewFlag(
	ctx context.Context,
	flag models.Flag,
) (*entity.FlagResponse, error) {
	if err := sf.repoDB.CreateFlag(ctx, flag); err != nil {
		return nil, err
	}

	return entity.NewFlagResponse(flag), nil
}

func (sf *ServiceFlag) GetFlagByName(
	ctx context.Context,
	flagName string,
) (*entity.FlagResponse, error) {
	if flag, ok := sf.repoCache.GetFlagByName(flagName); ok {
		return entity.NewFlagResponse(flag), nil
	}

	flag, err := sf.repoDB.GetFlagByName(ctx, flagName)
	if err != nil {
		return nil, ErrServiceNotFound
	}

	sf.repoCache.AddFlag(flag)

	return entity.NewFlagResponse(flag), nil
}

func (sf *ServiceFlag) UpdateFlag(
	ctx context.Context,
	newFlag models.Flag,
) (*entity.FlagResponse, error) {
	//var oldFlag models.Flag
	//var err error
	//var ok bool
	//oldFlag, ok = sf.repoCache.GetFlagByName(newFlag.FlagName)
	//if !ok {
	//	oldFlag, err = sf.repoDB.GetFlagByName(ctx, newFlag.FlagName)
	//	if err != nil {
	//		return nil, ErrServiceNotFound
	//	}
	//}

	//if newFlag.UpdatedAt.UTC() != oldFlag.UpdatedAt.UTC() ||
	//	newFlag.CreatedUser != oldFlag.CreatedUser {
	//	return nil, ErrServiceUpdateFlagDataCreatedNotEqual
	//}

	if err := sf.repoDB.UpdateFlag(ctx, newFlag); err != nil {
		return nil, ErrServiceInternalError
	}

	sf.repoCache.RemoveFlag(newFlag.FlagName)

	return entity.NewFlagResponse(newFlag), nil
}

func (sf *ServiceFlag) DeleteFlag(
	ctx context.Context,
	flagName string,
) error {
	if err := sf.repoDB.DeleteFlag(ctx, flagName); err != nil {
		return ErrServiceNotFound
	}

	sf.repoCache.RemoveFlag(flagName)

	return nil
}

func (sf *ServiceFlag) RetrieveListOfAllFlags(ctx context.Context) (*entity.ListOfFlagResponse, error) {
	listOfFlags, err := sf.repoDB.ListOfAllFlags(ctx)
	if err != nil {
		return nil, ErrServiceInternalError
	}

	if len(listOfFlags) == 0 {
		return nil, ErrServiceNotFound
	}

	for _, flag := range listOfFlags {
		sf.repoCache.AddFlag(flag)
	}

	return entity.NewListOfFlagResponse(listOfFlags), nil
}

func (sf *ServiceFlag) RetrieveListOfFlagsByNames(
	ctx context.Context,
	flagNames []string,
) (*entity.ListOfFlagResponse, error) {
	uniqFlagsNames := make(map[string]struct{})
	for _, flagName := range flagNames {
		uniqFlagsNames[strings.TrimSpace(flagName)] = struct{}{}
	}

	listOfFlags := make([]models.Flag, 0, len(uniqFlagsNames))
	findFlagsByNamesFromDB := make([]string, 0, len(uniqFlagsNames))

	for _, nameOfFlag := range flagNames {
		if flag, ok := sf.repoCache.GetFlagByName(nameOfFlag); ok {
			listOfFlags = append(listOfFlags, flag)
			continue
		}
		findFlagsByNamesFromDB = append(findFlagsByNamesFromDB, nameOfFlag)
	}

	if len(findFlagsByNamesFromDB) > 0 {
		flagsFromDB, err := sf.repoDB.ListOfFlagByNames(ctx, findFlagsByNamesFromDB)
		if err != nil {
			return nil, ErrServiceInternalError
		}
		listOfFlags = append(listOfFlags, flagsFromDB...)
	}

	if len(listOfFlags) == 0 {
		return nil, ErrServiceNotFound
	}

	if len(listOfFlags) != len(uniqFlagsNames) {
		unknownFlags := []string{}
	link:
		for flagName := range uniqFlagsNames {
			for _, flag := range listOfFlags {
				if flag.FlagName == flagName {
					continue link
				}
			}
			unknownFlags = append(unknownFlags, flagName)
		}
		return nil, fmt.Errorf(
			"error - {%v}, flagNames - {%s}",
			ErrServiceUnknownFlag,
			strings.Join(unknownFlags, ", "),
		)
	}

	for _, flag := range listOfFlags {
		sf.repoCache.AddFlag(flag)
	}

	return entity.NewListOfFlagResponse(listOfFlags), nil
}
