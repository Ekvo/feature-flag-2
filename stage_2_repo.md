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
    FlagName    string          `json:"flag_name" reform:"flag_name,pk"`
    IsEnable    bool            `json:"is_enable" reform:"is_enable"`
    ActiveFrom  time.Time       `json:"active_from" reform:"active_from"`
    Data        json.RawMessage `json:"data" reform:"data"`
    DefaultData json.RawMessage `json:"default_data" reform:"default_data"`
    CreatedUser uuid.UUID       `json:"created_user" reform:"created_user"`
    CreatedAt   time.Time       `json:"created_at" reform:"created_at"`
    UpdatedAt   time.Time       `json:"updated_at" reform:"updated_at"`
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
// можно будуте создавть с заполеным кешом
// так у нас будет возможность получить кол-во данных из БД, и задать размер *expirable.LRU[string, models.Flag]
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

**fiber + fiber/middleware/cache** - отказался от репозитория
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
// кешируем в middlewareCache только по пути /flags
app.Group("/flags").Use(middlewareCache)
```

**Сущности для Request & Response**
```go
// FlagResponse - формат для ответа флага
type FlagResponse struct {
	Body struct {
		Flag models.Flag `json:"flag"`
	}
}
// NewFlagResponse - конструктор
func NewFlagResponse(flag models.Flag) *FlagResponse

// ListOfFlagResponse - формат для ответа списка флагов
type ListOfFlagResponse struct {
	Body struct {
		Flags []models.Flag `json:"flag"`
	}
}

// NewListOfFlagResponse - конструктор
func NewListOfFlagResponse(flags []models.Flag) *ListOfFlagResponse

// FlagNamesDecode для получения списков флагов по именам
type FlagNamesDecode struct {
    FlagNames []string `json:"flag_names"`
}
```

**Service ServiceFlag** - сервисный слой для вызова в маршрутизаторах
```go
// бизне логика - обработки флага
// основная в начале ищем в кеш, потом идем в базу
// при запросах на получение models.Flag пишем в кеш
type ServiceFlag struct {
	repoDB    *db.RepoFlagDB
	repoCache *cache.RepoCacheFlag
}

// NewServiceFlag - конструктор
func NewServiceFlag(db *db.RepoFlagDB, cache *cache.RepoCacheFlag) *ServiceFlag


// CreateNewFlag - создания флага
// идем в кеш, если есть возвращаем ошибку (Already Exists)
// если нет -> идем в БД  -> если ошибка делаем (Already Exists) - (можно потом писать в логи сами ошибки)
// если все ok -> делаем и возвращаем ответ с данными созданного флага
func (sf *ServiceFlag) CreateNewFlag(
    ctx context.Context,
    flag models.Flag,
) (*entity.FlagResponse, error)

// GetFlagByName - получение флага
// идем в кеш, если есть делаем и возвращаем ответ
// если нет -> идем в БД  -> если ошибка делаем (Not found)
// если все ok -> пишем флаг в кеш
// делаем и возвращаем ответ с данными флага из БД 
func (sf *ServiceFlag) GetFlagByName(
    ctx context.Context,
    flagName string,
) (*entity.FlagResponse, error)

// UpdateFlag - обвноление флага
// идем в кеш, если есть, берем старый флаг
// если нет -> идем в БД  -> если ошибка делаем (Not found)
// сравниваем время создания и создателя флага в новых и старых данных
// если все ok -> пишем флаг в БД -> если ошибка делаем (Internal)
// удаляем флаг из кеш
// делаем и возвращаем ответ с данными новго флага 
func (sf *ServiceFlag) UpdateFlag(
    ctx context.Context,
    newFlag models.Flag,
) (*entity.FlagResponse, error)

// DeleteFlag - удаление флага
// идем в БД  -> если ошибка делаем (Not found)
// удаляем флаг из кеш 
func (sf *ServiceFlag) DeleteFlag(
    ctx context.Context,
    flagName string,
) (error)

// RetrieveListOfAllFlags - получение всех флагов
// идем в БД  -> если ошибка делаем (Internal)
// длина полученого списка == 0, делаем ошибку (Mot found)
// пишем флаги в кеш
// делаем и возвращаем ответ с данными флагов
func (sf *ServiceFlag) RetrieveListOfAllFlags(ctx context.Context) (*entity.ListOfFlagResponse, error)

// RetrieveListOfAllFlags - получение флагов по списку имен
// создаем uniqFlagsNames := make(map[string]struct{}) - отсикае дубликаты
// делаем массивы для ответа(listOfFlags), и массив с именами для запроса в БД(findFlagsByNamesFromDB)
// идем в кеш и в цикле по имена флагов -> если есть пишем listOfFlags, если нет пишем имя флага в findFlagsByNamesFromDB

// длина findFlagsByNamesFromDB > 0 -> идем в БД  -> если ошибка делаем (Internal)
// длина полученого списка == 0, делаем ошибку (Mot found)

// len(listOfFlags) != len(uniqFlagsNames)  -> есть unknown флаги
// делаем список имен неизвестных флагов -> делаем ошибку со списком unknown флагов

// все хорошо -> пишем флаги в кеш
// делаем и возвращаем ответ с данными флагов
func (sf *ServiceFlag) RetrieveListOfFlagsByNames(
    ctx context.Context,
    flagNames []string,
) (*entity.ListOfFlagResponse, error)

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










