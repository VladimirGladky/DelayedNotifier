# DelayedNotifier

Сервис для отложенной отправки уведомлений через Telegram с использованием RabbitMQ, PostgreSQL и Redis.

## Описание

DelayedNotifier — это микросервис для планирования и отправки уведомлений в Telegram с заданной задержкой. Сервис использует RabbitMQ с плагином delayed message exchange для отложенной доставки сообщений, PostgreSQL для хранения данных о уведомлениях и Redis для кэширования статусов.

### Веб-интерфейс

![Web Interface](docs/images/web-interface.png)

Сервис предоставляет удобный веб-интерфейс для:
- Создания новых уведомлений с указанием времени отправки
- Просмотра списка всех уведомлений
- Отслеживания статуса каждого уведомления (created, sent, failed)

### Основные возможности

- Создание отложенных уведомлений с указанием времени отправки
- Отправка уведомлений через Telegram Bot API
- Отслеживание статуса уведомлений (created, sent, failed)
- Веб-интерфейс для управления уведомлениями
- RESTful API для интеграции с другими сервисами
- Кэширование статусов в Redis для быстрого доступа
- Поддержка миграций базы данных
- Dead Letter Queue (DLQ) для обработки ошибок

## Технологический стек

- **Язык**: Go 1.25
- **Веб-фреймворк**: Gin
- **Брокер сообщений**: RabbitMQ (с плагином delayed message exchange)
- **База данных**: PostgreSQL
- **Кэш**: Redis
- **Логирование**: Uber Zap
- **Контейнеризация**: Docker, Docker Compose
- **Миграции**: golang-migrate

## Архитектура

Проект следует принципам Clean Architecture с разделением на слои:

```
DelayedNotifier/
├── cmd/                    # Точки входа приложения
│   ├── main.go            # Основное приложение
│   └── migrate/           # Утилита миграций
├── internal/              # Внутренняя бизнес-логика
│   ├── app/              # Инициализация приложения
│   ├── models/           # Модели данных
│   ├── repository/       # Слой работы с БД
│   ├── service/          # Бизнес-логика
│   ├── transport/        # HTTP handlers
│   ├── rabbitmq/         # Работа с RabbitMQ
│   ├── telegram/         # Telegram Bot клиент
│   └── migrations/       # Логика миграций
├── pkg/                   # Переиспользуемые пакеты
│   ├── logger/           # Настройка логирования
│   ├── postgres/         # Подключение к PostgreSQL
│   └── redis/            # Подключение к Redis
├── web/                   # Веб-интерфейс
│   ├── static/           # Статические файлы
│   └── templates/        # HTML шаблоны
├── migrations/            # SQL миграции
├── docker-compose.yml     # Конфигурация Docker Compose
├── Dockerfile            # Dockerfile для приложения
└── .env                  # Переменные окружения
```

## Требования

- Docker и Docker Compose
- Go 1.25+ (для локальной разработки)
- Telegram Bot Token (получить у [@BotFather](https://t.me/botfather))

## Установка и запуск

### 1. Клонирование репозитория

```bash
git clone <repository-url>
cd DelayedNotifier
```

### 2. Настройка переменных окружения

Создайте файл `.env` в корне проекта:

```env
# Telegram Bot
TELEGRAM_BOT_TOKEN=your_telegram_bot_token

# PostgreSQL
POSTGRES_USER=root
POSTGRES_PASSWORD=1234
POSTGRES_DBNAME=postgres
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
RABBITMQ_CONNECTION_NAME=delayed_notifier
CONNECT_TIMEOUT=10
HEARTBEAT=60

# RabbitMQ Exchanges and Queues
PUBLISHER_EXCHANGE=delayed_notifications
DLX_EXCHANGE=dlx_notifications
CONSUMER_QUEUE=notifications_queue
ROUTING_KEY=notification.send
DLQ_ROUTING_KEY=notification.failed

# Server
HOST=0.0.0.0
PORT=4051
```

### 3. Запуск с помощью Docker Compose

```bash
docker-compose up -d
```

Эта команда запустит следующие сервисы:
- **PostgreSQL** (порт 5435)
- **RabbitMQ** (порты 5672, 15672)
- **Redis** (порт 6379)
- **Redis Insight** (порт 5540)
- **Notification Service** (порт 4051)

### 4. Проверка работоспособности

После запуска сервисов:

- Веб-интерфейс: http://localhost:4051
- RabbitMQ Management: http://localhost:15672 (guest/guest)
- Redis Insight: http://localhost:5540

## API

### Доступные API эндпоинты

| Метод | Путь | Описание |
| :--- | :--- | :--- |
| POST | /api/v1/notify | Создание уведомления |
| GET | /api/v1/notify/:id | Получение статуса уведомления по ID |
| DELETE | /api/v1/notify/:id | Удаление уведомления по ID |
| GET | /api/v1/notifications | Получение списка всех уведомлений |

### Примеры запросов

#### Создание уведомления

```bash
curl -X POST http://localhost:4051/api/v1/notify \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Текст уведомления",
    "time": "2026-02-12T22:00:03+03:00",
    "chat_id": 123456789
  }'
```

**Ответ:**
```json
{
  "id": "uuid-notification-id"
}
```

#### Получение статуса уведомления

```bash
curl -X GET http://localhost:4051/api/v1/notify/{id}
```

**Ответ:**
```json
{
  "status": "created"
}
```

#### Удаление уведомления

```bash
curl -X DELETE http://localhost:4051/api/v1/notify/{id}
```

**Ответ:**
```json
{
  "status": "notify {id} is deleted"
}
```

#### Получение всех уведомлений

```bash
curl -X GET http://localhost:4051/api/v1/notifications
```

**Ответ:**
```json
[
  {
    "id": "uuid",
    "message": "Текст уведомления",
    "time": "2026-02-12T22:00:03+03:00",
    "status": "created",
    "chat_id": 123456789
  }
]
```

## Структура базы данных

В базе данных предусмотрена одна таблица `notifications`:

| Поле | Тип | Описание |
| :--- | :--- | :--- |
| id | VARCHAR(255) | Уникальный идентификатор уведомления |
| message | TEXT | Текст уведомления |
| time | VARCHAR(255) | Время отправки уведомления |
| status | VARCHAR(50) | Статус уведомления (created, sent, failed) |
| chat_id | BIGINT | Telegram Chat ID получателя |

## Миграции базы данных

Миграции применяются автоматически при запуске приложения. Для ручного управления миграциями используйте утилиту:

```bash
# Применить миграции
go run cmd/migrate/main.go up

# Откатить миграции
go run cmd/migrate/main.go down
```

Файлы миграций находятся в директории `migrations/`.

## Разработка

### Локальный запуск без Docker

1. Убедитесь, что PostgreSQL, RabbitMQ и Redis запущены и доступны
2. Настройте `.env` файл с корректными хостами
3. Выполните миграции:
   ```bash
   go run cmd/migrate/main.go up
   ```
4. Запустите приложение:
   ```bash
   go run cmd/main.go
   ```

### Структура зависимостей

Основные зависимости проекта:
- `github.com/gin-gonic/gin` - HTTP веб-фреймворк
- `github.com/rabbitmq/amqp091-go` - клиент RabbitMQ
- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram Bot API
- `github.com/wb-go/wbf` - фреймворк с утилитами для конфигурации
- `go.uber.org/zap` - структурированное логирование
- `github.com/golang-migrate/migrate/v4` - управление миграциями

## Как это работает

### Схема архитектуры

```
┌─────────────┐
│   Client    │
│  (Web/API)  │
└──────┬──────┘
       │ POST /api/v1/notify
       │ {message, time, chat_id}
       ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Notification Service                         │
│                                                                 │
│  ┌────────────┐      ┌──────────────┐     ┌──────────────┐      │
│  │  HTTP API  │──1──▶│   Service    │──2─▶│  Repository  │      │
│  │  (Gin)     │      │   Layer      │     │              │      │
│  └────────────┘      │              │     └──────┬───────┘      │
│                      │  ┌────────┐  │            │              │
│                      │  │ Redis  │  │            ▼              │
│                      │  │ Cache  │  │     ┌──────────────┐      │
│                      │  └────────┘  │     │  PostgreSQL  │      │
│                      └───────┬──────┘     └──────────────┘      │
│                              │ 3                                │
│                              ▼                                  │
│                      ┌──────────────┐                           │
│                      │   RabbitMQ   │                           │
│                      │   Producer   │                           │
│                      └──────┬───────┘                           │
└─────────────────────────────┼───────────────────────────────────┘
                              │
                              │ Publish with x-delay
                              ▼
                    ┌──────────────────────┐
                    │      RabbitMQ        │
                    │                      │
                    │  ┌────────────────┐  │
                    │  │ Delayed        │  │
                    │  │ Exchange       │  │
                    │  │ (x-delayed-    │  │
                    │  │  message)      │  │
                    │  └────────┬───────┘  │
                    │           │ delay    │
                    │           │ expired  │
                    │           ▼          │
                    │  ┌────────────────┐  │
                    │  │ Target Queue   │  │
                    │  └────────┬───────┘  │
                    └───────────┼──────────┘
                                │
                                │ Consume
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Notification Service                         │
│                                                                 │
│  ┌────────────┐      ┌──────────────┐     ┌──────────────┐      │
│  │  RabbitMQ  │──4──▶│   Service    │──5─▶│   Telegram   │      │
│  │  Consumer  │      │   Layer      │     │   Bot API    │      │
│  └────────────┘      │              │     └──────────────┘      │
│                      │  ┌────────┐  │                           │
│                      │  │ Redis  │  │                           │
│                      │  │ Cache  │  │                           │
│                      │  └────────┘  │                           │
│                      └───────┬──────┘                           │
│                              │ 6                                │
│                              ▼                                  │
│                      ┌──────────────┐                           │
│                      │  Repository  │                           │
│                      └──────┬───────┘                           │
│                             │                                   │
│                             ▼                                   │
│                      ┌──────────────┐                           │
│                      │  PostgreSQL  │                           │
│                      └──────────────┘                           │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Telegram User   │
                    └──────────────────┘
```

### Детальный процесс работы

#### Создание и планирование уведомления

1. **HTTP запрос**: Клиент отправляет POST запрос на `/api/v1/notify` с данными:
   ```json
   {
     "message": "Напоминание о встрече",
     "time": "2026-02-12T22:00:03+03:00",
     "chat_id": 123456789
   }
   ```

2. **Сохранение в БД**:
   - Генерируется уникальный UUID для уведомления
   - Запись сохраняется в PostgreSQL со статусом `"created"`
   - Данные доступны для проверки статуса через API

3. **Расчет задержки и публикация**:
   - Вычисляется задержка (delay) = `время_отправки - текущее_время` в миллисекундах
   - Сообщение публикуется в RabbitMQ delayed exchange `delayed_notifications` (тип: `x-delayed-message`)
   - Устанавливается заголовок `x-delay` с вычисленной задержкой в миллисекундах

4. **Ожидание в exchange**:
   - RabbitMQ удерживает сообщение в delayed message exchange
   - Exchange использует плагин `rabbitmq_delayed_message_exchange`
   - После истечения задержки сообщение автоматически перенаправляется в целевую очередь `notifications_queue`

#### Обработка и отправка уведомления

5. **Получение из очереди**:
   - Consumer подписан на очередь `notifications_queue`
   - При поступлении сообщения начинается обработка

6. **Отправка через Telegram**:
   - Сервис вызывает Telegram Bot API метод `SendMessage`
   - Передается `chat_id` и текст сообщения
   - Обрабатывается ответ от Telegram API

7. **Обновление статуса**:
   - При успехе: статус меняется на `"sent"` в PostgreSQL
   - При ошибке: статус меняется на `"failed"`
   - Статус кэшируется в Redis для быстрого доступа (TTL кэша: 1 час)

#### Обработка ошибок

8. **Dead Letter Queue (DLQ)**:
   - Если отправка не удалась после retry-попыток
   - Сообщение перенаправляется в DLX exchange `dlx_notifications`
   - Попадает в Dead Letter Queue для ручного анализа
   - Администратор может просмотреть failed сообщения через RabbitMQ Management UI

### Кэширование

Для оптимизации производительности используется Redis:
- **Ключ**: `notification:status:{id}`
- **Значение**: статус уведомления (`created`, `sent`, `failed`)
- **TTL**: 3600 секунд (1 час)
- При запросе статуса сначала проверяется Redis, затем PostgreSQL

## Мониторинг и отладка

- **Логи**: Сервис использует структурированное логирование Uber Zap
- **RabbitMQ Management**: Доступен по адресу http://localhost:15672 для мониторинга очередей
- **Redis Insight**: Доступен по адресу http://localhost:5540 для просмотра кэша
