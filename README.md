# Торговая платформа — Мультиагентная система

**ФИО:** Евланичев Максим Юрьевич  
**Группа:** 221131  
**Лабораторная работа №13, Вариант 7**  
**Сложность:** повышенная

[![Python](https://img.shields.io/badge/python-3.12-blue)](https://www.python.org/)
[![Go](https://img.shields.io/badge/go-1.22-00ADD8)](https://golang.org/)
[![NATS](https://img.shields.io/badge/nats-2.10-27AAE1)](https://nats.io/)
[![Redis](https://img.shields.io/badge/redis-7-DC382D)](https://redis.io/)

---

## Описание

Мультиагентная система торговой платформы, реализующая полный pipeline обработки торговых запросов: от сбора котировок до анализа с помощью LLM (Google Gemini). Агенты взаимодействуют через брокер сообщений NATS, состояние сохраняется в Redis, трейсинг осуществляется через Jaeger (OpenTelemetry).

---

## Стек технологий

| Компонент | Технология | Версия |
|-----------|-----------|--------|
| Брокер сообщений | NATS | 2.10 |
| Go-агенты | Go + nats.go | 1.22 / 1.37 |
| Python LLM-агент | Python + nats-py | 3.12 / 2.9 |
| Оркестратор | FastAPI + uvicorn | 0.115 / 0.32 |
| LLM | Google Gemini 1.5 Flash | — |
| Хранилище состояния | Redis | 7 |
| Трейсинг | OpenTelemetry + Jaeger | 1.28 / 1.57 |
| Веб-мониторинг | FastAPI + Jinja2 | — |
| Контейнеризация | Docker + Docker Compose | — |

---

## Архитектура pipeline

```
HTTP запрос (POST /trade)
    │
    ▼
FastAPI Orchestrator (:8000)
    │
    ├── TradingPipeline
    │       │
    │       ▼
    │   NATS: trading.quotes.collect
    │       │
    │       ├── quotes-agent-1 (Go, queue group)  ─┐
    │       └── quotes-agent-2 (Go, queue group)  ─┘ один обрабатывает
    │
    │   NATS: trading.quotes.done ──► pipeline
    │       │
    │       ▼
    │   NATS: trading.market.analyze
    │       │
    │       └── market-analyzer (Go)
    │
    │   NATS: trading.market.done ──► pipeline
    │       │
    │       ▼
    │   NATS: trading.risk.assess
    │       │
    │       └── risk-manager (Go) ◄──► Redis
    │
    │   NATS: trading.risk.done ──► pipeline
    │       │
    │       ▼
    │   NATS: trading.trade.execute
    │       │
    │       └── trade-executor (Go)
    │
    │   NATS: trading.trade.done ──► pipeline
    │       │
    │       ▼
    │   NATS: trading.llm.analyze
    │       │
    │       └── llm-analyst (Python + Gemini API)
    │
    │   NATS: trading.llm.done ──► pipeline
    │
    ├── AuctionManager  (выбор агента по наименьшему score)
    └── DynamicScaler   (автомасштабирование по длине очереди)
```

---

## Агенты

### quotes-agent (Go)
- **Топик:** `trading.quotes.collect` (queue group `quotes-workers`)
- **Функция:** Для каждого символа генерирует случайную котировку (цена 100–1000, объём 1000–100000)
- **Выход:** JSON с массивом котировок в `trading.quotes.done`
- **Масштабирование:** Запускается в 2 экземплярах (queue group — один обрабатывает каждый запрос)

### market-analyzer (Go)
- **Топик:** `trading.market.analyze` (queue group `market-workers`)
- **Функция:** Вычисляет среднюю цену, определяет тренд (bullish/bearish/neutral), формирует рекомендацию BUY/SELL/HOLD
- **Логика:** bullish + volume > 50000 → BUY; bearish → SELL; иначе → HOLD

### risk-manager (Go)
- **Топик:** `trading.risk.assess` (queue group `risk-workers`)
- **Функция:** Оценивает риск сделки, сохраняет статистику в Redis (TTL 1 час)
- **Логика:** BUY + avg_price > 800 → HIGH + rejected; BUY + volume < 5000 → MEDIUM; иначе → LOW + approved
- **Redis ключи:** `risk:total_assessments`, `risk:high_risk_count`, `risk:last:{task_id}`

### trade-executor (Go)
- **Топик:** `trading.trade.execute` (queue group `trade-workers`)
- **Функция:** Исполняет сделку (если approved=true) или отклоняет. Генерирует order_id и цену исполнения.

### llm-analyst (Python + Gemini)
- **Топик:** `trading.llm.analyze`
- **Функция:** Анализирует результат торговой операции с помощью Google Gemini 1.5 Flash
- **Выход:** Краткий профессиональный комментарий на русском языке

### orchestrator (Python)
- **REST API:** `POST /trade`, `GET /agents`, `GET /metrics`, `GET /auction/demo`, `GET /health`
- **Паттерн:** request-reply через asyncio.Future + pending dict
- **Retry:** до 3 попыток с backoff 1 сек
- **AuctionManager:** выбор агента по формуле `score = cost * (1 - availability)`
- **DynamicScaler:** мониторинг очереди каждые 10 сек, масштабирование при превышении порога

---

## Запуск

### Предварительные требования
- Docker Desktop
- Docker Compose v2+
- Ключ Google Gemini API

### Шаги

```bash
# 1. Клонировать / перейти в директорию
cd lab13MTP

# 2. Создать .env файл
cp .env.example .env
# Вставить свой GEMINI_API_KEY в .env

# 3. Запустить все сервисы
docker compose up --build

# 4. Подождать ~30 сек пока все агенты запустятся
```

### Порты

| Сервис | URL |
|--------|-----|
| Оркестратор (REST API) | http://localhost:8001 |
| Swagger UI | http://localhost:8001/docs |
| Веб-мониторинг | http://localhost:8080 |
| Jaeger (трейсы) | http://localhost:16686 |
| NATS Management | http://localhost:8222 |
| Redis | localhost:6379 |

---

## Endpoints оркестратора

### POST /trade
Запустить торговый pipeline.

**Тело запроса:**
```json
{
  "symbols": ["AAPL", "GOOGL", "TSLA"]
}
```

**Ответ (пример):**
```json
{
  "task_id": "uuid",
  "symbols": ["AAPL", "GOOGL"],
  "quotes": [
    {"symbol": "AAPL", "price": 152.34, "volume": 67000, "timestamp": "..."}
  ],
  "market_analysis": {
    "avg_price": 152.34,
    "trend": "bullish",
    "total_volume": 67000,
    "recommendation": "BUY"
  },
  "risk_assessment": {
    "risk_level": "LOW",
    "risk_score": 0.23,
    "approved": true
  },
  "trade_execution": {
    "order_id": "a1b2c3d4-...",
    "status": "EXECUTED",
    "execution_price": 153.10
  },
  "llm_analysis": "Сделка успешно исполнена по рыночной цене...",
  "final_status": "EXECUTED"
}
```

### GET /health
```json
{"status": "healthy", "service": "orchestrator"}
```

### GET /agents
Список всех агентов с топиками.

### GET /metrics
Метрики оркестратора (длина очереди).

### GET /auction/demo
Демонстрация аукционного механизма выбора агента.

---

## Тесты

```bash
cd orchestrator
pip install -r requirements.txt
pytest tests/ -v
```

Тесты покрывают:
- `test_pipeline_execute_success` — успешное исполнение сделки
- `test_pipeline_rejected_trade` — отклонение по высокому риску
- `test_auction_returns_winner` — аукцион с 3 агентами
- `test_auction_empty_pool` — аукцион с пустым пулом
- `test_auction_single_agent` — аукцион с одним агентом

---

## Мониторинг

Веб-дашборд (http://localhost:8080) отображает:
- Статус оркестратора
- Список всех агентов с типами (Go/Python) и топиками
- Текущую длину очереди задач
- Форму для запуска торговой операции прямо из браузера

Трейсы всех агентов доступны в Jaeger (http://localhost:16686).

---

## Структура файлов

```
lab13MTP/
├── agents/
│   ├── quotes-agent/       # Go, queue group, генерация котировок
│   ├── market-analyzer/    # Go, анализ тренда и рекомендация
│   ├── risk-manager/       # Go, оценка риска + Redis state
│   ├── trade-executor/     # Go, исполнение/отклонение сделки
│   └── llm-analyst/        # Python, Gemini LLM анализ
├── orchestrator/
│   ├── api.py              # FastAPI endpoints
│   ├── orchestrator.py     # NATS request-reply с Future
│   ├── pipeline.py         # 5-шаговый pipeline
│   ├── auction.py          # Аукционный механизм
│   ├── scaler.py           # Динамическое масштабирование
│   └── tests/              # pytest тесты
├── monitoring/             # FastAPI + Jinja2 дашборд
├── docs/
│   └── architecture.mmd   # Mermaid диаграмма архитектуры
├── docker-compose.yml
├── .env.example
└── PROMPT_LOG.md
```
