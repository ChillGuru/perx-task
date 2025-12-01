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
grpcurl -plaintext -d "{\"user_id\":\"test_user\",\"items\":[{\"product_id\":\"prod1\",\"quantity\":2,\"price\":10.0}]}" localhost:50051 order.OrderService/CreateOrder

# Получаем заказ (id подставляем свой)
grpcurl -plaintext -d "{\"order_id\":\"d547a579-65fb-4183-ba73-34e115099d06\"}" localhost:50051 order.OrderService/GetOrder

# Меняем статус на PAID
grpcurl -plaintext -d "{\"order_id\":\"d547a579-65fb-4183-ba73-34e115099d06\",\"status\":\"PAID\"}" localhost:50051 order.OrderService/UpdateOrderStatus
```
