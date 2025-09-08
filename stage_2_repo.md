# Проектирование сервиса Feature Flags

## Этап разработки репозиториев, маршрутизатора, адаптера humafiber для fiber/v3.

### SQL БД работаем через ORM
* Задача - хранение данных флагов

* PostgreSQL
* gopkg.in/reform.v1 -> ORM
* jackc/pgx v3.6.2   -> драйвер


**Модель флага для команды:** `go generate`
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
// создаем &models.Flag и передаем вместе с flagName в функцию reform.Querier.FindByPrimaryKeyTo
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

// convertReformStructToFlag перобразуем []reform.Struct в []models.Flag
// создаем слайс listOfFlags, заполняем его проходя в цикле по dataFromDB
func convertReformStructToFlag(dataFromDB []reform.Struct) []models.Flag
```

### Cache для Service слоя
* Задача - хранение данных флагов в кеш, уменьшить частоту запросов в БД

* golang-lru/v2 

```go
type RepoCacheFlag struct {
	cache *expirable.LRU[string, models.Flag]
}

// NewRepoCacheFlag конструктор
// можно будет создавать с заполненным кешом
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

### fiber, middleware/cache
* Задача - реализовать кеширование для маршрутизатора

* fiber/v3
* fiber/v3/middleware/cache"

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
// кешируем в middlewareCache только по пути /flags - пишем в кеш только весь список флагов
app.Group("/flags").Use(middlewareCache)
```

### Сущности для Request & Response
* Задача - реализовать кеширование для маршрутизатора

* fiber/v3
* fiber/v3/middleware/cache"

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
		Flags []models.Flag `json:"flags"`
	}
}

// NewListOfFlagResponse - конструктор
func NewListOfFlagResponse(flags []models.Flag) *ListOfFlagResponse

// FlagNamesDecode для получения списков имен флагов
type FlagNamesDecode struct {
    FlagNames []string `json:"flag_names"`
}
```

### Service - ServiceFlag
* Задача - логика обработки объекта флаг

```go
// бизнес логика - обработки флага
// в начале ищем в кеш, потом идем в базу
// при запросах на получение models.Flag пишем в кеш
type ServiceFlag struct {
	repoDB    *db.RepoFlagDB
	repoCache *cache.RepoCacheFlag
}

// NewServiceFlag - конструктор
func NewServiceFlag(db *db.RepoFlagDB, cache *cache.RepoCacheFlag) *ServiceFlag

// CreateNewFlag - создания флага
// идем в кеш, если есть возвращаем ошибку (Already Exists)
// если нет -> идем в БД  -> если ошибка делаем (Already Exists) - (нужно потом писать в логи сами ошибки)
// если все ok -> делаем и возвращаем ответ с данными созданного флага
func (sf *ServiceFlag) CreateNewFlag(
    ctx context.Context,
    flag models.Flag,
) (*entity.FlagResponse, error)

// GetFlagByName - получение флага
// идем в кеш, если есть делаем и возвращаем объект для ответа
// если нет -> идем в БД  -> если ошибка делаем (Not found)
// если все ok -> пишем флаг в кеш
// делаем и возвращаем объект дл ответ с данными флага из БД 
func (sf *ServiceFlag) GetFlagByName(
    ctx context.Context,
    flagName string,
) (*entity.FlagResponse, error)

// UpdateFlag - обвновление флага
// идем в кеш, если есть, берем старый флаг
// если нет -> идем в БД  -> если ошибка делаем (Not found)
// сравниваем время создания и создателя флага в новых и старых данных
// если не равны -> делаем (кастомная ошибка)
// если все ok -> пишем флаг в БД -> если ошибка делаем (Internal)
// удаляем флаг из кеш
// делаем и возвращаем объект для ответа с данными нового флага 
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
// длина полученного списка == 0, делаем ошибку (Mot found)
// пишем флаги в кеш
// делаем и возвращаем объект для ответ с данными флагов
func (sf *ServiceFlag) RetrieveListOfAllFlags(ctx context.Context) (*entity.ListOfFlagResponse, error)

// RetrieveListOfAllFlags - получение флагов по списку имен
// создаем uniqFlagsNames := make(map[string]struct{}) - отсекает дубликаты
// делаем массивы для ответа(listOfFlags), и массив с именами для запроса в БД(findFlagsByNamesFromDB)
// идем в кеш и в цикле по имена флагов -> если есть пишем listOfFlags, если нет пишем имя флага в findFlagsByNamesFromDB

// длина findFlagsByNamesFromDB > 0 -> идем в БД  -> если ошибка делаем (Internal)
// длина полученого списка(listOfFlags) == 0, делаем ошибку (Not found)

// len(listOfFlags) != len(uniqFlagsNames)  -> есть unknown флаги
// делаем список имен неизвестных флагов -> делаем ошибку со списком unknown флагов

// все хорошо -> пишем флаги в кеш
// делаем и возвращаем объект для ответа с данными флагов
func (sf *ServiceFlag) RetrieveListOfFlagsByNames(
    ctx context.Context,
    flagNames []string,
) (*entity.ListOfFlagResponse, error)
```

### huma 
* Задача - логика контрактов, генерации OpenAPI

* huma/v2

```go
// работаем с флагами через service.ServiceFlag
api := humafiberv3.New(app, huma.DefaultConfig("Feature Flags API", "1.0.0"))

// Контракт для получения полного списка флагов

// получаем список флагов через service.RetrieveListOfAllFlags 
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "get-list-of-flags",
    Method:      "GET",
    Path:        "/flags",
    Summary:     "get list of flags and cached",
}, func (ctx context.Context, input *struct{}) (*entity.ListOfFlagResponse, error)

// Контракт для получения списка флагов по именам

// получаем имена флагов через input.Body
// идем в service.RetrieveListOfFlagsByNames со списком имен 
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "post-list-of-flags",
    Method:      "POST",
    Path:        "/flags",
    Summary:     "get list of flags by names and cached",
}, func(ctx context.Context, input *struct {
    Body entity.FlagNamesDecode `json:"body"`
}) (*entity.ListOfFlagResponse, error)

// Контракт для создания флага

// получаем флаг через input.Body
// идем в service.CreateNewFlag с полученным флагом
// если ошибка делаем huma.NewError
// все хорошо отдаем ответ полученный из сервисного слоя (новый флаг)
huma.Register(api, huma.Operation{
    OperationID:   "post-new-flag",
    Method:        "POST",
    DefaultStatus: 201,
    Path:          "/flag",
    Summary:       "create a new flag",
}, func(ctx context.Context, input *struct {
    Body models.Flag `json:"body"`
}) (*entity.FlagResponse, error)

// Контракт для получения флага по имени 

// получаем имя флага из параметра
// идем в service.GetFlagByName с полученным именем флага
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя
huma.Register(api, huma.Operation{
    OperationID: "get-flag-by-name",
    Method:      "GET",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and return flag",
}, func(ctx context.Context, input *struct {
    Name string `path:"name" maxLength:"30" example:"world"`
}) (*entity.FlagResponse, error)

// Контракт для изменения флага по имени 

// получаем имя флага из параметра и сам флаг 
// сравниваем имя из параметра и имя из объекта флага
// если не равно -> делаем  huma.Error400BadRequest
// идем в service.UpdateFlag с полученным флагов и пытаемся обновить данные
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем ответ полученный из сервисного слоя (обвноленный флаг)
huma.Register(api, huma.Operation{
    OperationID: "get-flag-by-name",
    Method:      "PUT",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and return flag",
}, func(ctx context.Context, input *struct {
    Name string      `path:"name"`
    Body models.Flag `json:"body"`
}) (*entity.FlagResponse, error)

// Контракт для удаления флага по имени 

// получаем имя флага из параметра
// идем в service.DeleteFlag с полученным флагом и пытаемся удалить
// если ошибка делаем huma.Error404NotFound
// все хорошо отдаем nil
huma.Register(api, huma.Operation{
    OperationID: "get-flag-by-name",
    Method:      "DELETE",
    Path:        "/flag/{name}",
    Summary:     "get flag name from param and return flag",
}, func(ctx context.Context, input *struct {
    Name string `path:"name"`
}) (*struct{}, error)
```

### humafiberv3 
* Задача - реализация адаптера из humafiber для fiber/v3

* huma/v2/adapters/humafiber

** **
```go
package humafiberv3

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofiber/fiber/v3"
)

// Unwrap extracts the underlying Fiber context from a Huma context. If passed a
// context from a different adapter it will panic. Keep in mind the limitations
// of the underlying Fiber/fasthttp libraries and how that impacts
// memory-safety: https://docs.gofiber.io/#zero-allocation. Do not keep
// references to the underlying context or its values!
func Unwrap(ctx huma.Context) fiber.Ctx {
	for {
		if c, ok := ctx.(interface{ Unwrap() huma.Context }); ok {
			ctx = c.Unwrap()
			continue
		}
		break
	}
	if c, ok := ctx.(*fiberWrapper); ok {
		return c.Unwrap()
	}
	panic("not a humafiber context")
}

type fiberAdapter struct {
	tester requestTester
	router router
}

type fiberWrapper struct {
	op     *huma.Operation
	status int
	orig   fiber.Ctx // с указателя ни интерфейс
	ctx    context.Context
}

// check that fiberCtx implements huma.Context
var _ huma.Context = &fiberWrapper{}

func (c *fiberWrapper) Unwrap() fiber.Ctx {
	return c.orig
}

func (c *fiberWrapper) Operation() *huma.Operation {
	return c.op
}

func (c *fiberWrapper) Matched() string {
	return c.orig.Path()
}

func (c *fiberWrapper) Context() context.Context {
	return c.ctx
}

func (c *fiberWrapper) Method() string {
	return c.orig.Method()
}

func (c *fiberWrapper) Host() string {
	return c.orig.Hostname()
}

func (c *fiberWrapper) RemoteAddr() string {
	return c.orig.RequestCtx().RemoteAddr().String() // Context().RemoteAddr().String()
}

func (c *fiberWrapper) URL() url.URL {
	u, _ := url.Parse(string(c.orig.Request().RequestURI()))
	return *u
}

func (c *fiberWrapper) Param(name string) string {
	return c.orig.Params(name)
}

func (c *fiberWrapper) Query(name string) string {
	return c.orig.Query(name)
}

func (c *fiberWrapper) Header(name string) string {
	return c.orig.Get(name)
}

func (c *fiberWrapper) EachHeader(cb func(name, value string)) {
	c.orig.Request().Header.VisitAll(func(k, v []byte) {
		cb(string(k), string(v))
	})
}

func (c *fiberWrapper) BodyReader() io.Reader {
	var orig = c.orig
	if orig.App().Server().StreamRequestBody {
		// Streaming is enabled, so send the reader.
		return orig.Request().BodyStream()
	}
	return bytes.NewReader(orig.BodyRaw())
}

func (c *fiberWrapper) GetMultipartForm() (*multipart.Form, error) {
	return c.orig.MultipartForm()
}

func (c *fiberWrapper) SetReadDeadline(deadline time.Time) error {
	// Note: for this to work properly you need to do two things:
	// 1. Set the Fiber app's `StreamRequestBody` to `true`
	// 2. Set the Fiber app's `BodyLimit` to some small value like `1`
	// Fiber will only call the request handler for streaming once the limit is
	// reached. This is annoying but currently how things work.
	return c.orig.RequestCtx().Conn().SetReadDeadline(deadline) // Context().Conn().SetReadDeadline(deadline)
}

func (c *fiberWrapper) SetStatus(code int) {
	var orig = c.orig
	c.status = code
	orig.Status(code)
}

func (c *fiberWrapper) Status() int {
	return c.status
}
func (c *fiberWrapper) AppendHeader(name string, value string) {
	c.orig.Append(name, value)
}

func (c *fiberWrapper) SetHeader(name string, value string) {
	c.orig.Set(name, value)
}

func (c *fiberWrapper) BodyWriter() io.Writer {
	return c.orig
}

func (c *fiberWrapper) TLS() *tls.ConnectionState {
	return c.orig.RequestCtx().TLSConnectionState() // Context().TLSConnectionState()
}

func (c *fiberWrapper) Version() huma.ProtoVersion {
	return huma.ProtoVersion{
		Proto: c.orig.Protocol(),
	}
}

type router interface {
	Add(methods []string, path string, handler fiber.Handler, middleware ...fiber.Handler) fiber.Router
}

type requestTester interface {
	Test(req *http.Request, config ...fiber.TestConfig) (*http.Response, error)
}

type contextWrapperValue struct {
	Key   any
	Value any
}

type contextWrapper struct {
	values []*contextWrapperValue
	context.Context
}

var (
	_ context.Context = &contextWrapper{}
)

func (c *contextWrapper) Value(key any) any {
	var raw = c.Context.Value(key)
	if raw != nil {
		return raw
	}
	for _, pair := range c.values {
		if pair.Key == key {
			return pair.Value
		}
	}
	return nil
}

func (a *fiberAdapter) Handle(op *huma.Operation, handler func(huma.Context)) {
	// Convert {param} to :param
	path := op.Path
	path = strings.ReplaceAll(path, "{", ":")
	path = strings.ReplaceAll(path, "}", "")
	a.router.Add([]string{op.Method}, path, func(c fiber.Ctx) error {
		var values []*contextWrapperValue
		c.RequestCtx().VisitUserValuesAll(func(key, value any) { //Context().VisitUserValuesAll(func(key, value any) { // ошибка
			values = append(values, &contextWrapperValue{
				Key:   key,
				Value: value,
			})
		})
		handler(&fiberWrapper{
			op:   op,
			orig: c,
			ctx: &contextWrapper{
				values:  values,
				Context: userContext(c), //.UserContext(), // ошибка
			},
		})
		return nil
	})
}

// костыль 
func userContext(c fiber.Ctx) context.Context {
	ctx, ok := c.RequestCtx().UserValue(0).(context.Context) // fasthttp.UserValue(userContextKey).(context.Context)
	if !ok {
		ctx = context.Background()
		c.RequestCtx().SetUserValue(0, ctx) //SetUserContext(ctx)
	}

	return ctx
}

func (a *fiberAdapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// b, _ := httputil.DumpRequest(r, true)
	// fmt.Println(string(b))
	resp, err := a.tester.Test(r)
	if resp != nil && resp.Body != nil {
		defer func() {
			_ = resp.Body.Close()
		}()
	}
	if err != nil {
		panic(err)
	}
	h := w.Header()
	for k, v := range resp.Header {
		for item := range v {
			h.Add(k, v[item])
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func New(r *fiber.App, config huma.Config) huma.API {
	return huma.NewAPI(config, &fiberAdapter{tester: r, router: r})
}

func NewWithGroup(r *fiber.App, g fiber.Router, config huma.Config) huma.API {
	return huma.NewAPI(config, &fiberAdapter{tester: r, router: g})
}
```

Следующай этап разработки:   
**включает**: миграции, .env, создание config. 








