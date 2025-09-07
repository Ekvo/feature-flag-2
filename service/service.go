package service

import (
	"context"
	"errors"
	"feature-flag-2/models"
	"feature-flag-2/repository/cache"
	"feature-flag-2/repository/db"
)

var (
	ErrServiceAlreadyExists = errors.New("already exists")
)

type ServiceFlag struct {
	repoDB    *db.RepoFlagDB
	repoCache *cache.RepoCacheFlag
}

func NewServiceFlag(db *db.RepoFlagDB, cache *cache.RepoCacheFlag) *ServiceFlag {
	return &ServiceFlag{repoDB: db, repoCache: cache}
}

func (sf *ServiceFlag) CreateNewFlag(ctx context.Context, flag models.Flag) error {
	if _, ok := sf.repoCache.GetFlagByName(flag.FlagName); ok {
		return ErrServiceAlreadyExists
	}

	if err := sf.repoDB.CreateFlag(ctx, flag); err != nil {
		return ErrServiceAlreadyExists
	}

	return nil
}
