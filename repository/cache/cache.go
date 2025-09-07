package cache

import (
	"feature-flag-2/models"
	"github.com/hashicorp/golang-lru/v2/expirable"
)

type RepoCacheFlag struct {
	cache *expirable.LRU[string, models.Flag]
}

func NewRepoCacheFlag(cache *expirable.LRU[string, models.Flag]) *RepoCacheFlag {
	return &RepoCacheFlag{cache: cache}
}

func (r *RepoCacheFlag) AddFlag(flag models.Flag) bool {
	return r.cache.Add(flag.FlagName, flag)
}

func (r *RepoCacheFlag) RemoveFlag(flagName string) bool {
	return r.cache.Remove(flagName)
}

func (r *RepoCacheFlag) GetFlagByName(flagName string) (models.Flag, bool) {
	return r.cache.Get(flagName)
}
