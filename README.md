## 1. Уровень 1.

## 2. Чистая архитектура.
- domain/ — модели заказов (Order, Item)
- usecase/ — бизнес-логика (расчёт суммы, валидация)
- delivery/ — gRPC обработчики
- infrastructure/ — хранилище в памяти и логгер.

## 3. Запуск сервиса:
```bash
# 1. Ставим protoc и плагины
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# 2. Генерируем grpc код
protoc --go_out=. --go-grpc_out=. proto/order.proto

# 3. Запускаем сервер
go run cmd/server/main.go
```

## 4. Доступно 3 метода: 
- CreateOrder — создаёт заказ (статус PENDING)
- GetOrder — возвращает заказ по ID
- UpdateOrderStatus — меняет статус (PAID, CANCELLED, FAILED)

Сумма заказа считается автоматически.

Тестирование:
```bash
# Создаём заказ (в командной строке cmd)
# Запрос лежит в корне микросервиса (create_order.json)
grpcurl -plaintext -d @ localhost:50051 order.OrderService/CreateOrder < create_order.json

# Получаем заказ
grpcurl -plaintext -d "{\"order_id\":\"83ce50c0-7325-4e97-8005-744a6de1446c\"}" localhost:50051 order.OrderService/GetOrder

# Меняем статус на PAID
grpcurl -plaintext -d "{\"order_id\":\"83ce50c0-7325-4e97-8005-744a6de1446c\",\"status\":\"PAID\"}" localhost:50051 order.OrderService/UpdateOrderStatus
```
