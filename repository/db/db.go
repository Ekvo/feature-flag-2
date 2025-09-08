package db

import (
	"context"
	"feature-flag-2/models"
	"github.com/hashicorp/golang-lru/v2/expirable"

	"gopkg.in/reform.v1"
)

type RepoFlagDB struct {
	db    *reform.DB
	cache *expirable.LRU[string, models.Flag]
}

func NewRepoFlagDB(db *reform.DB) *RepoFlagDB {
	return &RepoFlagDB{db: db}
}

// Create создает новый флаг
func (r *RepoFlagDB) CreateFlag(ctx context.Context, flag models.Flag) error {
	return r.db.WithContext(ctx).Insert(&flag)
}

// GetByFlagName возвращает флаг по имени
func (r *RepoFlagDB) GetFlagByName(ctx context.Context, flagName string) (models.Flag, error) {
	flag := models.Flag{}
	if err := r.db.WithContext(ctx).FindByPrimaryKeyTo(&flag, flagName); err != nil {
		return flag, err
	}
	return flag, nil
}

// Update обновляет флаг
func (r *RepoFlagDB) UpdateFlag(ctx context.Context, flag models.Flag) error {
	return r.db.WithContext(ctx).Update(&flag)
}

// Delete удаляет флаг
func (r *RepoFlagDB) DeleteFlag(ctx context.Context, flagName string) error {
	return r.db.WithContext(ctx).Delete(&models.Flag{
		FlagName: flagName,
	})
}

// ListOfAllFkags возвращает список всех флагов
func (r *RepoFlagDB) ListOfAllFlags(ctx context.Context) ([]models.Flag, error) {
	flags, err := r.db.WithContext(ctx).SelectAllFrom(models.FlagTable, "")
	if err != nil {
		return nil, err
	}
	listOfFlags := convertReformStructToFlag(flags)
	return listOfFlags, nil
}

func (r *RepoFlagDB) ListOfFlagByNames(
	ctx context.Context,
	flagNames []string,
) ([]models.Flag, error) {
	args := make([]any, 0, len(flagNames))
	for _, name := range flagNames {
		args = append(args, name)
	}
	flags, err := r.db.WithContext(ctx).FindAllFrom(models.FlagTable, "flag_name", args...)
	if err != nil {
		return nil, err
	}
	listOfFlags := convertReformStructToFlag(flags)
	return listOfFlags, nil
}

// convertReformStructToFlag создаем массив флагов из []reform.Struct
func convertReformStructToFlag(dataFromDB []reform.Struct) []models.Flag {
	listOfFlags := make([]models.Flag, 0, len(dataFromDB))
	for _, f := range dataFromDB {
		flag := *(f.(*models.Flag))
		listOfFlags = append(listOfFlags, flag)
	}
	return listOfFlags
}
