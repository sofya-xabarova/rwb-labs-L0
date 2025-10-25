# 🧾 Orders Service (Go + PostgreSQL + NATS Streaming)

Микросервис для обработки заказов с использованием **Go**, **PostgreSQL** и **NATS Streaming**.  
Проект запускается в контейнерах Docker через `docker-compose`.

---

## Быстрый старт

### 1. Клонировать репозиторий

```bash
git clone https://github.com/<your-username>/orders-service.git
cd orders-service
```
### 2. Собрать и запустить контейнеры
```bash
docker compose build
docker compose up
```

Docker поднимет три контейнера:

Контейнер | Назначение | Порт
orders_postgres | PostgreSQL база данных | 5432
nats_streaming | NATS Streaming сервер | 4222, 8222
orders_service | Go API-сервис заказов | 8080

## Добавление демо данных
Чтобы добавить 10 демо записей таблицу, в отдельном терминале выполни:

```bash
docker exec -it orders_postgres psql -U testuser -d orders_db -f /docker-entrypoint-initdb.d/init.sql
```
