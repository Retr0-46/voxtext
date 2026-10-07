# Voxtext

Voxtext распознаёт речь из аудиофайла и разбивает результат по говорящим. Проект состоит из Go API, статического фронтенда и PostgreSQL. В текущей конфигурации Docker запускает **только базу данных**; API и фронтенд работают на компьютере через `go run`.

## Что нужно

- Windows и Docker Desktop с запущенным Docker Engine.
- Go 1.26 или Go с включённым `GOTOOLCHAIN=auto` (при необходимости нужная версия загрузится автоматически).
- FFmpeg в `PATH` и рабочая среда для CGO.
- Модели ONNX и две DLL, перечисленные ниже. В текущей рабочей папке они уже есть.

Команды ниже предназначены для PowerShell. Запускайте их из каталога проекта, если не указан другой путь.

## Первый запуск

1. Подготовьте настройки API, если файла `backend/.env` ещё нет:

   ```powershell
   Copy-Item backend/.env.example backend/.env
   ```

   Откройте `backend/.env` и замените пример `JWT_SECRET` на случайную длинную строку. `DATABASE_URL` в примере уже соответствует настройкам Docker Compose. Если `.env` уже настроен, этот шаг пропустите.

2. Запустите PostgreSQL из корня проекта:

   ```powershell
   docker compose up -d --wait db
   ```

   База доступна на `127.0.0.1:5433`, имя базы и пользователя — `voxtext`, пароль для локальной разработки — `1234`. Docker не использует порт `5432`, на котором уже может работать PostgreSQL в Windows. Compose автоматически применяет [миграцию пользователей](backend/internal/modules/auth/migrations/01_create_users_up.sql) при **первом** создании тома БД.

3. Запустите API:

   ```powershell
   cd backend
   $env:PATH = "$(Resolve-Path internal/modules/voxtext);$env:PATH"
   go run ./cmd/api
   ```

   Запуск из `backend/` важен: от этого каталога читается `.env`, находятся модели и подключается `../frontend`. Добавление каталога в `PATH` нужно Windows для загрузки `sherpa-onnx-c-api.dll` и `ort_voxtext.dll`.

4. Откройте http://127.0.0.1:8080/. Фронтенд обслуживается тем же API; отдельный сервер или сборка фронтенда не нужны. Зарегистрируйтесь, войдите и загрузите аудиофайл для распознавания.

На последующих запусках достаточно шагов 2–4. Проверить БД можно командой `docker compose ps` из корня проекта.

## Остановка

Остановите API клавишами `Ctrl+C` в окне с `go run`. Затем из корня проекта выполните:

```powershell
docker compose stop db
```

Данные БД хранятся в Docker-томе `postgres_data` и сохраняются после остановки контейнера. SQL-файл из `docker-entrypoint-initdb.d` не применяется повторно к существующей базе: для последующих изменений схемы нужна новая миграция.

## Модели и DLL

Файлы ONNX исключены из Git: один из них больше 225 МБ. При переносе проекта на другой компьютер положите файлы в те же пути относительно `backend/`:

| Файл | Назначение |
| --- | --- |
| `internal/modules/voxtext/models/asr/gigaam-ctc/model.int8.onnx` | распознавание речи |
| `internal/modules/voxtext/models/diarization/pyannote/model.int8.onnx` | сегментация говорящих |
| `internal/modules/voxtext/models/embeddings/titanet/model.onnx` | сравнение голосов |

Файл `internal/modules/voxtext/models/asr/gigaam-ctc/tokens.txt` и DLL `internal/modules/voxtext/{sherpa-onnx-c-api.dll,ort_voxtext.dll}` не исключены из Git. Они тоже нужны для работы на Windows. Если приложение сообщает об отсутствии модели или не может загрузить DLL, проверьте файлы и команду с `PATH` из раздела «Первый запуск».

## Настройки

Пример всех переменных находится в [backend/.env.example](backend/.env.example). Основные параметры:

| Переменная | Для чего нужна |
| --- | --- |
| `DATABASE_URL` | подключение к PostgreSQL на `127.0.0.1:5433` |
| `JWT_SECRET` | ключ подписи токенов, обязательно заменить пример |
| `HTTP_ADDR` | адрес API, по умолчанию `:8080` |
| `MODELS_DIR` | каталог моделей относительно `backend/` |
| `FFMPEG` | путь к исполняемому файлу FFmpeg |
| `MAX_UPLOAD_BYTES` | максимальный размер HTTP-запроса с файлом |
| `UPLOAD_TMP_DIR` | каталог временных загрузок; по умолчанию системный временный каталог |

Параметры `NUM_THREADS`, `DIARIZATION_THRESHOLD`, `MIN_DURATION_ON` и `MIN_DURATION_OFF` тоже доступны в примере `.env`. Если задаёте `VOXTEXT_DB_PASSWORD` для Compose, укажите тот же пароль в `DATABASE_URL` **до первого создания БД**. Изменение переменной после инициализации тома само по себе пароль пользователя PostgreSQL не меняет.

## API и структура

| Маршрут | Назначение |
| --- | --- |
| `GET /` | фронтенд |
| `POST /auth/register` | регистрация (`username`, `usersurname`, `email`, `password`) |
| `POST /auth/login` | вход (`email`, `password`), ответ с `access_token` |
| `GET /auth/me` | текущий пользователь; нужен `Authorization: Bearer <token>` |
| `POST /v1/transcriptions` | распознавание; нужен токен и `multipart/form-data` с полем `file` |

Результат распознавания содержит массив `utterances` с `start`, `end`, `speaker` и `text`.

- `backend/cmd/api/` — сборка приложения и регистрация маршрутов.
- `backend/internal/modules/auth/` — регистрация, вход и пользователи.
- `backend/internal/modules/middleware/` — проверка токена для защищённых маршрутов.
- `backend/internal/modules/voxtext/` — обработка аудио, распознавание и диаризация.
- `frontend/` — HTML, CSS и JavaScript без отдельной системы сборки.
- `compose.yaml` — локальный PostgreSQL.

Для проверки Go-кода из каталога `backend/` выполните `go test ./...`. Реальное распознавание дополнительно требует модели, FFmpeg и DLL.
