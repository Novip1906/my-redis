# Go In-Memory Key-Value Database
Аналог Redis на минималках на чистом TCP


## Доступные команды

- **SET key value** — Сохранить значение.

- **GET key** — Получить значение.

- **DEL key** — Удалить ключ.

- **QUIT** — Отключиться.

### Запуск
```
docker build -t my-redis .
docker run -d -p 6379:6379 my-redis
```

