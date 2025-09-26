# gw-currency-wallet

**gw-currency-wallet** — микросервис для управления пользовательскими кошельками и обмена валют.  

---

## Цель проекта

Создать надежный и масштабируемый сервис для:  
- регистрации и аутентификации пользователей,  
- управления балансом кошельков,  
- пополнения и снятия средств,  
- получения актуальных курсов валют и обмена между ними.  

---

## Задачи

1. Реализация REST API для работы с пользователями и кошельками.  
2. Интеграция с gRPC сервисом для получения курсов валют.  
3. Кеширование курсов валют в Redis для повышения производительности.  
4. Обеспечение безопасности с помощью JWT аутентификации.  
5. Логирование и обработка ошибок с централизованным логгером.  
6. Покрытие юнит-тестами и интеграционными тестами всех ключевых модулей.  

---

## Технологии

- **Go** – основной язык разработки  
- **PostgreSQL** – хранение данных о пользователях и кошельках  
- **Redis** – кеширование курсов валют  
- **gRPC** – интеграция с сервисом обмена валют  
- **Chi** – HTTP роутер  
- **Swagger** – документация REST API  
- **Testcontainers** – интеграционные тесты  
- **JWT** – аутентификация пользователей  

---

## REST API

| #  | Метод | URL | Заголовки | Тело запроса | Успех | Ошибка | Описание |
|----|-------|-----|-----------|--------------|-------|--------|----------|
| 1  | POST  | /api/v1/register | — | `{ "username": "string", "password": "string", "email": "string" }` | `201 Created`<br>`{ "message": "User registered successfully" }` | `400 Bad Request`<br>`{ "error": "Username or email already exists" }` | Регистрация нового пользователя. Проверяется уникальность имени и email. Пароль шифруется. |
| 2  | POST  | /api/v1/login | — | `{ "username": "string", "password": "string" }` | `200 OK`<br>`{ "token": "JWT_TOKEN" }` | `401 Unauthorized`<br>`{ "error": "Invalid username or password" }` | Авторизация пользователя. Возвращается JWT для последующих запросов. |
| 3  | GET   | /api/v1/balance | `Authorization: Bearer JWT_TOKEN` | — | `200 OK`<br>`{ "balance": { "USD": "float", "RUB": "float", "EUR": "float" } }` | — | Получение текущего баланса пользователя. |
| 4  | POST  | /api/v1/wallet/deposit | `Authorization: Bearer JWT_TOKEN` | `{ "amount": 100.00, "currency": "USD" }` | `200 OK`<br>`{ "message": "Account topped up successfully", "new_balance": { "USD": "float", "RUB": "float", "EUR": "float" } }` | `400 Bad Request`<br>`{ "error": "Invalid amount or currency" }` | Пополнение счета. Проверяется корректность суммы и валюты. Баланс обновляется в БД. |
| 5  | POST  | /api/v1/wallet/withdraw | `Authorization: Bearer JWT_TOKEN` | `{ "amount": 50.00, "currency": "USD" }` | `200 OK`<br>`{ "message": "Withdrawal successful", "new_balance": { "USD": "float", "RUB": "float", "EUR": "float" } }` | `400 Bad Request`<br>`{ "error": "Insufficient funds or invalid amount" }` | Вывод средств. Проверяется наличие средств и корректность суммы. Баланс обновляется в БД. |
| 6  | GET   | /api/v1/exchange/rates | `Authorization: Bearer JWT_TOKEN` | — | `200 OK`<br>`{ "rates": { "USD": "float", "RUB": "float", "EUR": "float" } }` | `500 Internal Server Error`<br>`{ "error": "Failed to retrieve exchange rates" }` | Получение актуальных курсов валют. Используется кэш Redis и/или gRPC вызов к сервису exchange. |
| 7  | POST  | /api/v1/exchange | `Authorization: Bearer JWT_TOKEN` | `{ "from_currency": "USD", "to_currency": "EUR", "amount": 100.00 }` | `200 OK`<br>`{ "message": "Exchange successful", "exchanged_amount": 85.00, "new_balance": { "USD": 0.00, "EUR": 85.00 } }` | `400 Bad Request`<br>`{ "error": "Insufficient funds or invalid currencies" }` | Обмен валют. Используется кэш курсов или gRPC для актуального курса. Проверяется наличие средств. Баланс обновляется. |

---

## Структура проекта

```
.
├── api                     # Пакет для API документации
│   ├── docs.go             # Генерация Swagger документации из комментариев
│   ├── swagger.json        # Сгенерированная JSON документация Swagger
│   └── swagger.yaml        # Сгенерированная YAML документация Swagger
├── cmd                     # Основной исполняемый пакет
│   ├── main.go             # Точка входа приложения, конфигурация и запуск сервиса
│   └── main_test.go        # Тесты для main.go (например, проверка конфигурации и run)
├── go.mod                  # Модуль Go с зависимостями
├── go.sum                  # Контрольные суммы зависимостей
├── internal                # Внутренние пакеты приложения (бизнес-логика)
│   ├── facades             # Фасады для внешних сервисов (например, gRPC exchange)
│   │   ├── exchange_rate.go      # Фасад для работы с курсами валют
│   │   └── exchange_rate_test.go # Тесты фасада
│   ├── handlers            # HTTP обработчики для REST API
│   │   ├── balance.go           # Обработчик получения баланса
│   │   ├── balance_mock.go      # Мок баланс-обработчика для тестов
│   │   ├── balance_test.go      # Тесты для balance.go
│   │   ├── deposit.go           # Обработчик пополнения счета
│   │   ├── deposit_mock.go      # Мок deposit для тестов
│   │   ├── deposit_test.go      # Тесты deposit.go
│   │   ├── exchange.go          # Обработчик обмена валют
│   │   ├── exchange_mock.go     # Мок exchange для тестов
│   │   ├── exchange_rate.go     # Обработчик получения курса валют
│   │   ├── exchange_rate_mock.go# Мок для exchange_rate
│   │   ├── exchange_rate_test.go# Тесты exchange_rate.go
│   │   ├── exchange_test.go     # Тесты обмена валют
│   │   ├── login.go             # Обработчик авторизации
│   │   ├── login_mock.go        # Мок login для тестов
│   │   ├── login_test.go        # Тесты login.go
│   │   ├── register.go          # Обработчик регистрации
│   │   ├── register_mock.go     # Мок register для тестов
│   │   ├── register_test.go     # Тесты register.go
│   │   ├── withdraw.go          # Обработчик вывода средств
│   │   ├── withdraw_mock.go     # Мок withdraw для тестов
│   │   └── withdraw_test.go     # Тесты withdraw.go
│   ├── jwt                  # Работа с JWT-токенами
│   │   ├── jwt.go            # Генерация и проверка JWT
│   │   └── jwt_test.go       # Тесты JWT
│   ├── logger               # Логирование
│   │   ├── logger.go         # Инициализация логгера (zap)
│   │   └── logger_test.go    # Тесты логгера
│   ├── middlewares          # HTTP middleware
│   │   ├── auth.go           # Middleware аутентификации JWT
│   │   ├── auth_mock.go      # Мок auth для тестов
│   │   ├── auth_test.go      # Тесты auth middleware
│   │   ├── logging.go        # Middleware логирования запросов
│   │   ├── logging_test.go   # Тесты logging middleware
│   │   ├── tx.go             # Middleware для работы с транзакциями БД
│   │   └── tx_test.go        # Тесты tx middleware
│   ├── models               # Сущности и структуры данных
│   │   ├── user.go          # Структура пользователя
│   │   └── wallet.go        # Структура кошелька и баланса
│   ├── repositories         # Репозитории для работы с БД и кэшем
│   │   ├── exchange_rate.go      # Репозиторий курсов валют
│   │   ├── exchange_rate_test.go # Тесты exchange_rate.go
│   │   ├── user.go               # Репозиторий пользователей
│   │   ├── user_test.go          # Тесты user.go
│   │   ├── wallet.go             # Репозиторий кошельков
│   │   └── wallet_test.go        # Тесты wallet.go
│   └── services             # Бизнес-логика приложения
│       ├── auth.go          # Сервис авторизации и регистрации
│       ├── auth_mock.go     # Мок auth service
│       ├── auth_test.go     # Тесты auth service
│       ├── wallet.go        # Сервис управления кошельком
│       ├── wallet_mock.go   # Мок wallet service
│       └── wallet_test.go   # Тесты wallet service
├── Makefile                 # Скрипты сборки, запуска и миграций
├── migrations               # SQL миграции для БД
│   ├── 000001_create_users_table.sql    # Создание таблицы пользователей
│   └── 000002_create_wallets_table.sql  # Создание таблицы кошельков
└── README.md                # Документация проекта, инструкции и описание API
```

---

## Покрытие

| Пакет | Покрытие |
|-------|-----------|
| github.com/sbilibin2017/gw-currency-wallet/cmd | 76.6% |
| github.com/sbilibin2017/gw-currency-wallet/internal/facades | 100.0% |
| github.com/sbilibin2017/gw-currency-wallet/internal/handlers | 96.1% |
| github.com/sbilibin2017/gw-currency-wallet/internal/jwt | 84.0% |
| github.com/sbilibin2017/gw-currency-wallet/internal/logger | 90.0% |
| github.com/sbilibin2017/gw-currency-wallet/internal/middlewares | 100.0% |
| github.com/sbilibin2017/gw-currency-wallet/internal/models | [no test files] |
| github.com/sbilibin2017/gw-currency-wallet/internal/repositories | 79.1% |
| github.com/sbilibin2017/gw-currency-wallet/internal/services | 97.6% |

## Инструкция по развертыванию

```shell
GOOS=linux GOARCH=amd64 go build -o main ./cmd
./main -c config.env
```