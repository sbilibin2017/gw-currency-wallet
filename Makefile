# Сборка бинарника сервера
build-server:
	cd cmd/server && go build -o server main.go

# Генерация моков для интерфейсов Go
gen-mock:
	# Используется mockgen для создания mock-реализаций интерфейсов
	mockgen -source=$(file) \
		-destination=$(dir $(file))/$(notdir $(basename $(file)))_mock.go \
		-package=$(shell basename $(dir $(file)))

# Запуск всех тестов и генерация покрытия кода
test:
	# Тесты для всех пакетов с включением отчета покрытия
	go test ./... -cover

# Генерация swagger-документации из хэндлеров
gen-swag:
	# Используется swag для анализа internal/handlers и генерации документации в api/http
	swag init -g ./cmd/main.go -o ./api

# Применение миграций к базе данных PostgreSQL
migrate:
	# Используется goose для выполнения всех миграций в директории ./migrations
	goose -dir ./migrations postgres "host=localhost port=5432 user=bil_message_user password=bil_message_password dbname=bil_message_db sslmode=disable" up

up-build:
	docker compose --env-file config.env up --build

prune:
	docker container prune -f
	docker volume prune -f
	docker volume rm itk-test-task_postgres_data