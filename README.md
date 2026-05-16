# ImageProcessor

Очередь фоновой обработки изображений на Go с Kafka, файловым хранилищем и простым веб-интерфейсом.

## Состав

- `backend/cmd/api` — HTTP API для загрузки, получения статуса, выдачи и удаления изображений.
- `backend/cmd/worker` — фоновый обработчик задач из Kafka.
- `backend/internal/storage` — файловое хранилище исходников, результатов и JSON-метаданных.
- `backend/internal/processor` — resize, генерация миниатюры и watermark.
- `frontend` — один экран на HTML/CSS/JS.
- `docs/openapi.yaml` — OpenAPI/Swagger описание.

## Запуск

```bash
docker compose up --build
```

После запуска:

- frontend: http://localhost:3000
- API: http://localhost:8080
- Swagger UI: http://localhost:8081
- OpenAPI файл: `docs/openapi.yaml`

## API

- `POST /upload` — загрузить `jpg`, `png` или `gif` в поле `image`.
- `GET /images` — список изображений и их статусы.
- `GET /status/{id}` — статус одной задачи.
- `GET /image/{id}` — готовое обработанное изображение или `202`, если оно еще в работе.
- `GET /image/{id}/thumbnail` — готовая миниатюра.
- `DELETE /image/{id}` — удалить исходник, результат, миниатюру и метаданные.

## Проверка

```bash
cd backend
GOCACHE=/tmp/imageprocessor-go-cache go test ./...
```
