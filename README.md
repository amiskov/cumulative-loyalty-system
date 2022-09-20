# How it works

## Миграции
Использую `go-migrate`, миграции лежат в папке `migrations/`. В `Makefile` есть примеры запуска. Мигарция запускается при старте приложения.

## Сессии
Использую JWT-токены. При этом сессии храню в базе и проверяю их при логине. Сделал так, чтоб можно было разлогинить хулигана если, например, он угонит чужой токен.

Аутентификацию делаю через мидлвар, который проверяет пользователя и добавляет структуру сессии в контекст реквеста.

## Логирование
Использую Zap. Логггер использует контекст для хранения идентификатора запроса, чтобы при асинхронной обработке можно было в логах соотнести, что происходило.

## Контекст
Контекст стараюсь прокидывать через весь реквест, от хендлера до базы в первую очередь для подробного логирования. Потенциально к контексту перед запросами в базу можно было бы добавить таймаут, но я решил, что и так нормально.

# go-musthave-diploma-tpl

Шаблон репозитория для индивидуального дипломного проекта курса «Go-разработчик»

# Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без
   префикса `https://`) для создания модуля

# Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m master template https://github.com/yandex-praktikum/go-musthave-diploma-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/master .github
```

Затем добавьте полученные изменения в свой репозиторий.
