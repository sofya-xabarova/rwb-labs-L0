# Orders Service (Go + PostgreSQL + NATS Streaming)

Микросервис для обработки заказов с использованием **Go**, **PostgreSQL** и **NATS Streaming**.  
Проект запускается в контейнерах Docker через `docker-compose`.

---

## Быстрый старт

### 1. Клонировать репозиторий

```bash
git clone https://github.com/sofya-xabarova/rwb-labs-L0.git
cd rwb-labs-L0
```
### 2. Собрать и запустить контейнеры
```bash
docker compose build
docker compose up
```

## Docker поднимет три контейнера:
| Контейнер       | Назначение | Порт |
| :--------- | :------: | ----: |
| orders_postgres |   PostgreSQL база данных   | 5432 |
| nats_streaming    |   NATS Streaming сервер   | 4222, 8222 |
| orders_service    |   Go API-сервис заказов   | 8080 |

### Сервис будет запущен локально, по адресу:
http://localhost:8080