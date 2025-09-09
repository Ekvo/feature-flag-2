package db

import (
	"context"
	"database/sql"
	"errors"
	"feature-flag-2/models"
	"github.com/hashicorp/golang-lru/v2/expirable"
	"gopkg.in/reform.v1"
)

var (
	ErrDBAlreadyExists = errors.New("already exists")

	ErrDBCreatedByIsNotEqual = errors.New("created_by from service is not equal created_by from base")

	ErrDBFlagIsDeleted = errors.New("flag is deleted")

	ErrDBInternal = errors.New("internal error")
)

type RepoFlagDB struct {
	db    *reform.DB
	cache *expirable.LRU[string, models.Flag]
}

func NewRepoFlagDB(
	db *reform.DB,
	cache *expirable.LRU[string, models.Flag],
) *RepoFlagDB {
	return &RepoFlagDB{db: db, cache: cache}
}

// Create создает новый флаг
func (r *RepoFlagDB) CreateFlag(ctx context.Context, newFlag models.Flag) error {
	exec := func(tx *reform.TX) error {
		var oldFlag models.Flag
		if err := tx.WithContext(ctx).SelectOneTo(
			&oldFlag,
			`WHERE flag_name = $1 FOR UPDATE`,
			newFlag.FlagName,
		); err != nil {
			return err
		}
		if !oldFlag.IsDeleted {
			return ErrDBAlreadyExists
		}
		if err := tx.WithContext(ctx).Update(&newFlag); err != nil {
			return err
		}
		r.cache.Remove(newFlag.FlagName)

		return nil
	}

	if err := r.db.InTransactionContext(ctx, nil, exec); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return r.db.WithContext(ctx).Insert(&newFlag)
		}
		return err
	}

	return nil
}

// GetByFlagName возвращает флаг по имени
func (r *RepoFlagDB) GetFlagByName(ctx context.Context, flagName string) (models.Flag, error) {
	flag, ok := r.cache.Get(flagName)
	if ok {
		return flag, nil
	}

	flag.FlagName = flagName
	if err := r.db.WithContext(ctx).FindByPrimaryKeyTo(&flag, flagName); err != nil {
		return flag, err
	}

	return flag, nil
}

// Update обновляет флаг
func (r *RepoFlagDB) UpdateFlag(ctx context.Context, newFlag models.Flag) error {
	exec := func(tx *reform.TX) error {
		var oldFlag models.Flag
		if err := tx.WithContext(ctx).SelectOneTo(
			&oldFlag,
			`WHERE is_deleted = false AND flag_name = $1 FOR UPDATE`,
			newFlag.FlagName,
		); err != nil {
			return err
		}

		if oldFlag.IsDeleted {
			return ErrDBFlagIsDeleted
		}
		if oldFlag.CreatedBy != newFlag.CreatedBy {
			return ErrDBCreatedByIsNotEqual
		}

		return tx.WithContext(ctx).Update(&newFlag)
	}
	if err := r.db.InTransactionContext(ctx, nil, exec); err != nil {
		return err
	}

	r.cache.Remove(newFlag.FlagName)

	return nil
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
