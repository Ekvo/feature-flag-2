package db

import (
	"context"
	"feature-flag-2/models"

	"gopkg.in/reform.v1"
)

type RepoFlagDB struct {
	db *reform.DB
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
func (r *RepoFlagDB) ListOfAllFkags(ctx context.Context) ([]models.Flag, error) {
	flags, err := r.db.WithContext(ctx).SelectAllFrom(models.FlagTable, "")
	if err != nil {
		return nil, err
	}
	listOfFlags := convertReformStructToFlag(flags)
	return listOfFlags, nil
}

func (r *RepoFlagDB) ListOfFkagByNames(
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
