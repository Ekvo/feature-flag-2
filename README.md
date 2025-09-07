# Feature flags service

Для начала скажу, что пока отбрасываем абстракцию, интерфейсы - если их нужно писать, максимально простой вариант, для получение начального результата.

Делаем _классический микросервис_ с использованием технологии **gRPC**.

|   инструменты   |              библиотека              |                          комментарии                           |
|:---------------:|:------------------------------------:|:--------------------------------------------------------------:|
|    Postgres     |       github.com/jackc/pgx/v4        | развивается, проверенная, знакомая, мало достойных альтернатив |
|   Migrations    | github.com/golang-migrate/migrate/v4 |                    простая, быстрый запуск                     |
|  Inmemory TTL   |    github.com/patrickmn/go-cache     |                  простая, понятная,все знают                   |
| Router Contract |       protobuf                       |           Скорость, кодогенерация, готовый контракт            |

## Cкелет:
```txt
├── internal
│   ├── config
│   ├── db
│   ├── model
│   ├── lib
│   ├── listen
│   └── services
├── migrations
├── pkg
│   └── utils
```

### config 
Содержит объекты для получения данных для старта приложения. Для каждого компонента свой тип конфига с методами валидациии, и также основной для логики получения, парсинга и выдачи результата (объекта либо ошибки).

Пример как это может выглядеть.
```go
// Config представляет основную конфигурацию приложения, включающую настройки базы данных, миграций, сервера и кэша.
// Поля заполняются из переменных окружения с использованием префиксов.
type Config struct {
    DB         DataBaseConfig  `envPrefix:"DB_"`
    Migrations MigrationConfig `envPrefix:"MIGRATION_"`
    Server     ServerConfig    `envPrefix:"SRV_"`
    Cache      CacheConfig     `envPrefix:"CACHE_"`
    
    // передаем в каждую функцию во время валидации каждого объекта в Config
    // собираем полную информацию об ошибках - если есть
    // type Message map[string]any
	// помогает нам проверить объекты по уникальными условиями
    msgErr utils.Message `env:"-"`
}

// NewConfig создаёт новую конфигурацию, загружая переменные окружения из указанного файла.
// Возвращает указатель на Config и ошибку, если что-то пошло не так.
func NewConfig(pathToEnv string) (*Config, error)

// parse загружает и парсит конфигурацию из переменных окружения и файла .env.
// Использует библиотеку для маппинга env-переменных в поля структур.
func (cfg *Config) parse(pathToEnv string) error

// validConfig вызывает для каждого объекта с "envPrefix" собсвенную функцию для валидации передвая в каждую
// msgErr
func (cfg *Config) validConfig() bool

// У каждого полседующего объекта есть функция для валидации по типу
// func (cfg *SomeConfig) validConfig(msgErr utils.Message)
// проверяем объект по нужным на параметрам


// DataBaseConfig содержит параметры подключения к базе данных: хост, порт, учетные данные, пул соединений и таймауты.
// Также включает сформированный DSN (URL), который генерируется на основе этих параметров.
type DataBaseConfig struct {
    Host     string `env:"HOST"`
    Port     uint16 `env:"PORT"`
    User     string `env:"USER"`
    Password string `env:"PASSWORD"`
    Name     string `env:"NAME"`
    
    URL string `env:"-"`
    
    MaxConn           uint16        `env:"MAX_CONN"`
    MinConn           uint16        `env:"MIN_CONN"`
    ConnMaxLifeTime   time.Duration `env:"CONN_MAX_LIFE_TIME"`
    ConnMaxIdleTime   time.Duration `env:"CONN_MAX_IDLE_TIME"`
    ConnTime          time.Duration `env:"CONN_TIMEOUT"`
    HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD"`
}

// CacheConfig определяет параметры работы кэша: время жизни элементов и интервал очистки устаревших записей.
type CacheConfig struct {
    DefaultExpiration time.Duration `env:"EXPIRATION"`
    CleanupInterval   time.Duration `env:"INTERVAL"`
}

// MigrationConfig содержит путь к файлам миграций базы данных и сформированный URL для подключения (не из env).
// Используется инструментами миграции.
type MigrationConfig struct {
    PathToMigrations string `env:"PATH"`
    DBURL            string `env:"-"`
}

// ServerConfig задаёт сетевые параметры HTTP-сервера: порт и тип сети (обычно "tcp").
type ServerConfig struct {
    Port    uint16 `env:"PORT"`
    Network string `env:"NETWORK"`
}
```
### db
Postgres, go-cache. 





