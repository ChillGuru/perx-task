## 1. Уровень 2.

## 2. Чистая архитектура.
- domain/ — модели заказов (Order, Item)
- usecase/ — бизнес-логика (расчёт суммы, валидация)
- delivery/ — gRPC обработчики
- infrastructure/ — хранилище в памяти, MongoDB, NATS и логгер.

## 3. Запуск сервиса:
```bash
docker-compose up --build
```

## 4. Доступно 3 метода: 
- CreateOrder — создаёт заказ (статус PENDING)
- GetOrder — возвращает заказ по ID
- UpdateOrderStatus — меняет статус (PAID, CANCELLED, FAILED)

Сумма заказа считается автоматически.

### Тестирование:
1. Переходим в корень проекта (perx-task)

2. Поднимаем docker-compose:
```bash
docker-compose up --build
```

3. В отдельном окне терминала c помощью NATS-CLI подписываемся на "order.created" (необязательно, это нужно для тестирования):
```bash
docker-compose exec nats-cli nats sub -s nats://nats:4222 order.created
```
P.S: В дальнейшем в этом окне сможем увидеть опубликованные события.

4. В отдельном окне терминала создаем заказ:
```bash
grpcurl -plaintext -d "{\"user_id\":\"test_user\",\"items\":[{\"product_id\":\"prod1\",\"quantity\":2,\"price\":10.0}]}" localhost:50051 order.OrderService/CreateOrder
```

5. Убеждаемся что событие было опубликовано, смотрим в окно терминала, которое слушает "order.created" (если открыли это окно на п.3).

6. Проверяем метод GetOrder (подставляем наш order_id):
```bash
grpcurl -plaintext -d "{\"order_id\":\"НАШ_ID\"}" localhost:50051 order.OrderService/GetOrder
```

7. Проверяем метод UpdateOrderStatus:
```bash
# Меняем статус на PAID
grpcurl -plaintext -d "{\"order_id\":\"НАШ_ID\",\"status\":\"PAID\"}" localhost:50051 order.OrderService/UpdateOrderStatus
```

8. Можем посмотреть логи order-service:
```bash
docker-compose logs order-service
```