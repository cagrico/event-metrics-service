# Event Metrics Service
A high-performance, test-driven microservice (Hexagonal Architecture) for event ingestion and analytics.

---

# English

## Overview
Event Metrics Service is a lightweight and scalable backend service designed to:

- Ingest user events (`POST /events`)
- Support bulk ingestion (`POST /events/bulk`)
- Provide analytical metrics (`GET /metrics`)
- Guarantee idempotent event writes
- Perform efficient time-range analytics
- Run fully containerized with Docker Compose
- Serve complete API documentation via Swagger (`/docs`)
- Maintain high test coverage following TDD principles

Technologies used:
**Go (Fiber), PostgreSQL, Hexagonal Architecture, Docker, Swagger (swaggo/fiber-swagger).**

---

## Architecture (Hexagonal / Ports & Adapters)

```
internal/
  events/
    core/
      domain/
      ports/
      usecase/
    adapters/
      http/fiber/
      postgres/

  metrics/
    core/
      domain/
      ports/
      usecase/
    adapters/
      http/fiber/
      postgres/

cmd/api/main.go
migrations/
Dockerfile
docker-compose.yml
docs/ (Swagger auto-generated)
```

Reasons for using Hexagonal Architecture:
- Domain rules are isolated from frameworks
- HTTP / DB adapters can be replaced independently
- Testability increases drastically
- Dependencies always point to the core (domain → ports → usecases)

---

## Test-Driven Development (TDD)

### Event Usecase
- Reject empty required fields
- Reject future timestamps
- Dedupe key generation
- Duplicate handling via `ON CONFLICT DO NOTHING`
- Repository error propagation
- Metadata & tags validation

### Metrics Usecase
- Required parameter validation
- Time range validation
- `group_by` and `interval` validation
- Repository error propagation
- Correct aggregation mapping

---

# Swagger API Documentation
Swagger UI is automatically generated via `swaggo/swag` and served with `fiber-swagger`.

After starting the service via docker:

👉 **http://localhost:8080/docs/index.html**

Swagger includes:
- Full endpoint documentation (`/events`, `/events/bulk`, `/metrics`)
- Request & response models
- Error models (`ErrorResponse`)
- Example payloads

---

# API Endpoints

## 1. Create Event
**POST /events**

Request:
```json
{
  "event_name": "product_view",
  "channel": "web",
  "campaign_id": "cmp_1",
  "user_id": "user_123",
  "timestamp": 1700000000,
  "tags": ["electronics"],
  "metadata": { "product_id": "p1" }
}
```

Responses:
```json
{ "status": "created" }
```

```json
{ "status": "duplicate" }
```

---

## 2. Bulk Create Events
**POST /events/bulk**

Request:
```json
{
  "events": [
    { "...": "..." },
    { "...": "..." }
  ]
}
```

Response:
```json
{
  "created": 10,
  "duplicates": 2
}
```

---

## 3. Get Metrics
**GET /metrics?event_name=...&from=...&to=...&group_by=channel**

Example response (current implementation):

```json
{
  "buckets": [
    {
      "key": "web",
      "count": 1200,
      "unique_users": 345
    }
  ]
}
```

---

# Running with Docker

Start the service:
```bash
docker compose up --build
```

Run migration:
```bash
docker compose exec postgres psql -U user -d eventdb -f /migrations/001_create_events_table.sql
```

Service URL:  
👉 http://localhost:8080  
Swagger:  
👉 http://localhost:8080/docs

---

# Türkçe

## Genel Bakış
Event Metrics Service, kullanıcı event’lerini toplayıp anlamlı metriklere dönüştürmek için tasarlanmış, yüksek performanslı ve ölçeklenebilir bir mikroservistir.

Sunulan özellikler:
- Tekil event oluşturma (`POST /events`)
- Toplu event yükleme (`POST /events/bulk`)
- Metrik sorgulama (`GET /metrics`)
- Idempotent event yazımı (duplicate engelleme)
- PostgreSQL üzerinde verimli saklama
- TDD yaklaşımı ile yüksek test coverage
- Swagger ile otomatik API dokümantasyonu (`/docs`)
- Docker Compose ile tamamen container üzerinde çalıştırma

Kullanılan teknolojiler:  
**Go (Fiber), PostgreSQL, Hexagonal Architecture, Docker, Swagger.**

---

## Mimari (Hexagonal / Ports & Adapters)

```
internal/
  events/
    core/
      domain/
      ports/
      usecase/
    adapters/
      http/fiber/
      postgres/

  metrics/
    core/
      domain/
      ports/
      usecase/
    adapters/
      http/fiber/
      postgres/

cmd/api/main.go
migrations/
Dockerfile
docker-compose.yml
docs/ (Swagger tarafından üretilir)
```

Hexagonal mimarinin avantajları:
- Domain tamamen bağımsızdır
- HTTP ve veritabanı adapter’leri değiştirilebilir
- Test yazımı kolay ve hızlıdır
- Bağımlılık yönü tek taraflıdır (core merkezde)

---

## TDD Süreci

### Event Usecase
- Zorunlu alan doğrulaması
- Gelecek zaman reddi
- Idempotency için dedupe key üretimi
- Duplicate event kontrolü (`ON CONFLICT DO NOTHING`)
- Repository hatalarının doğru yönetimi
- Tags & metadata kontrolü

### Metrics Usecase
- Parametre doğrulaması
- Zaman aralığı doğrulaması
- `group_by` ve `interval` kuralları
- Repository hata kontrolü
- Aggregation mapping doğrulama

---

# Swagger API Dokümantasyonu

Swagger UI otomatik olarak oluşturulur.

Servis çalıştıktan sonra:

👉 **http://localhost:8080/docs/index.html**

Swagger içerisinde:
- Tüm endpointlerin açıklamaları
- Request/response modelleri
- Error modelleri (`ErrorResponse`)
- Örnek kullanım senaryoları

---

# API Endpointleri

## 1. Event Oluşturma
`POST /events`

Yanıtlar:
```json
{ "status": "created" }
```
```json
{ "status": "duplicate" }
```

---

## 2. Toplu Event Yükleme
`POST /events/bulk`

Yanıt:
```json
{
  "created": X,
  "duplicates": Y
}
```

---

## 3. Metrik Sorgulama
`GET /metrics?...`

Örnek yanıt:
```json
{
  "buckets": [
    {
      "key": "web",
      "count": 1200,
      "unique_users": 345
    }
  ]
}
```

---

# Docker ile Çalıştırma

Servisi başlat:
```bash
docker compose up --build
```

Migration çalıştırma:
```bash
docker compose exec postgres psql -U user -d eventdb -f migrations/001_create_events_table.sql
```

