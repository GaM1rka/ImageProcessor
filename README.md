# ImageProcessor

Очередь фоновой обработки изображений на Go с Kafka, файловым хранилищем и простым веб-интерфейсом.

## Состав

- `backend/cmd/api` — HTTP API для загрузки, получения статуса, выдачи и удаления изображений.
- `backend/cmd/worker` — фоновый обработчик задач из Kafka.
- `backend/internal/storage` — файловое хранилище исходников, результатов и JSON-метаданных.
- `backend/internal/processor` — resize, подготовка результата и watermark.
- `frontend` — один экран на HTML/CSS/JS.
- `docs/openapi.yaml` — OpenAPI/Swagger описание.

## Запуск

```bash
docker compose up --build
```

После запуска:

- frontend: http://localhost:3000
- API: http://localhost:8080
- OpenAPI: `docs/openapi.yaml`
