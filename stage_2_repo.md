# Проектирование сервиса Feature Flags

## Этап разработки репозиториев.



### SQL БД работаем через ORM
* PostgreSQL
* gopkg.in/reform.v1 -> ORM
* jackc/pgx v3.6.2   -> драйвер

Модель флага для команды: `go generate`
```go
//reform:public.flags
type Flag struct {
	FlagName    string          `reform:"flag_name,pk"`
	IsEnable    bool            `reform:"is_enable"`
	ActiveFrom  time.Time       `reform:"active_from"`
	Data        json.RawMessage `reform:"data"`
	DefaultData json.RawMessage `reform:"default_data"`
	CreatedUser uuid.UUID       `reform:"created_user"`
	CreatedAt   time.Time       `reform:"created_at"`
	UpdatedAt   time.Time       `reform:"updated_at"`
}
```

**Репозитори БД и методы**
```go
type RepoFlag struct {
    db *reform.DB
}

// NewRepoFlagDB - конструктор
func NewRepoFlagDB(db *reform.DB) *RepoFlagDB

// CreateFlag создает новый флаг 
// обертка для функции reform.Querier.Insert
func (rb *RepoFlag) CreateFlag(ctx context.Context, flag models.Flag) error

// GetByFlagName возвращает флаг по имени
// создаем &models.Flag и передаем вместс flagName в функцию reform.Querier.FindByPrimaryKeyTo
func (rb *RepoFlag) GetFlagByName(ctx context.Context, flagName string) (models.Flag, error)

// UpdateFlag обновляет флаг
// обертка для функции reform.Querier.Update
func (rb *RepoFlag) UpdateFlag(ctx context.Context, flag models.Flag) error

// Deleteflag удаляет флаг
// обертка для функции reform.Querier.Delete
func (rb *RepoFlag) DeleteFlag(ctx context.Context, flagName string) error

// ListOfAllFkags возвращает список всех флагов
// вызываем reform.Querier.SelectAllFrom(models.FlagTable,"")
// вызываем 'convertReformStructToFlag'
func (rb *RepoFlag) ListOfAllFkags(ctx context.Context) ([]models.Flag, error)

//  ListOfFkagByNames возвращает список всех флагов
// преобразовываем falgNames []string в []any 
// вызываем reform.Querier.Delete.FindAllFrom9(models.FlagTable, "flag_name", args...)
// вызываем 'convertReformStructToFlag'
func (rb *RepoFlag) ListOfFkagByNames(ctx context.Context, flagNames []string) ([]models.Flag, error)

// convertReformStructToFlag создаем массив флагов из []reform.Struct
// создаем слайс listOfFlags, заполняем его проходя в цикле по dataFromDB
func convertReformStructToFlag(dataFromDB []reform.Struct) []models.Flag
```

**Репозитори Кеш LRU** - для работы на **Service** слое
```go
type RepoCacheFlag struct {
	cache *expirable.LRU[string, models.Flag]
}

// NewRepoCacheFlag конструктор
func NewRepoCacheFlag(cache *expirable.LRU[string, models.Flag]) *RepoCacheFlag

// AddFlag добавить флаг в кеш
// обертка для expirable.LRU.Add
func (rc *RepoCacheFlag) AddFlag(flag models.Flag)

// RemoveFlag удалить флаг из кеш
// обертка для expirable.LRU.Remove
func (rc *RepoCacheFlag) RemoveFlag(flagName string)

// RemoveFlag получить флаг из кеш по имени фага
// обертка для expirable.LRU.Get
func (rc *RepoCacheFlag) GetFlagByName(flagName string) (models.Flag, bool)
```

**fiber + fiber/middleware/cache** - откаpался от репозитория
```go
// выбрал так как он прост и понятен,
// пришел к этому во время создания репозитория(получилась - ехидна) 7 строчек превратились в пакет, это тратило бы время других людей, чтобы разобраться

// в дальнейшем буду сетать поля для middlewareCache через конфиг
// вызываем в main
app := fiber.New()
middlewareCache := cache.New(cache.Config{
    Expiration:   5 * time.Minute, 
    CacheControl: true,
    Methods:      []string{"GET"},// кешировать весь список флагов  
})
app.Group("/flags").Use(middlewareCache)
```






```http request
curl -X POST   http://localhost:8000/flag   -H 'Content-Type: application/json'   -d '{
"flag_name": "feature_new_ui",
"is_enable": true,
"active_from": "2025-04-05T00:00:00Z",
"data": {"color": "blue", "size": "large"},
"default_data": {"color": "gray", "size": "medium"},
"created_user": "123e4567-e89b-12d3-a456-426614174000",
"created_at": "2025-04-01T10:00:00Z",
"updated_at": "2025-04-01T10:00:00Z"
}'

curl -X PUT  http://localhost:8000/flag/feature_new_ui   -H 'Content-Type: application/json'   -d '{
"flag_name": "feature_new_ui",
"is_enable": false,
"active_from": "2025-04-05T00:00:00Z",
"data": {"color": "blue", "size": "large"},
"default_data": {"color": "gray1", "size": "medium1"},
"created_user": "123e4567-e89b-12d3-a456-426614174000",
"created_at": "2025-04-01T10:00:00Z",
"updated_at": "2025-04-01T10:00:00Z"
}'

curl http://localhost:8000/flag/feature_new_ui 

curl -X DELETE http://localhost:8000/flag/feature_new_ui 


# unknown flags
curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "feature_new_ui",
      "dark_mode",
      "beta_access",
      "promo_banner_2025"
    ]
  }'

# exist flags
curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "2feature_new_ui"      
    ]
  }'
  
  curl -X POST \
  http://localhost:8000/flags \
  -H 'Content-Type: application/json' \
  -d '{
    "flag_names": [
      "2feature_new_ui",
      "feature_new_ui1"      
    ]
  }'
```










