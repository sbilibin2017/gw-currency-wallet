gen-mock:
	# Используется mockgen для создания mock-реализаций интерфейсов
	mockgen -source=$(file) \
		-destination=$(dir $(file))/$(notdir $(basename $(file)))_mock.go \
		-package=$(shell basename $(dir $(file)))

test:
	# Тесты для всех пакетов с включением отчета покрытия
	go test ./... -cover

gen-swag:
	# Используется swag для анализа internal/handlers и генерации документации в api/http
	swag init -g ./cmd/main.go -o ./api