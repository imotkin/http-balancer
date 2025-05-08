### Тестовое задание для Cloud.ru

[Условие задания](https://github.com/Go-Cloud-Camp/test-assignment)

#### Запуск с помощью Docker Compose:

```sh 
docker-compose up -d --build
```

При запуске Docker Compose работают 5 контейнеров: один для балансировщика, один для PostgreSQL, и ещё три для тестовых серверов [nginxdemos/hello:plain-text](https://hub.docker.com/r/nginxdemos/hello/).

После этого для проверки можно отправить запрос на `localhost:8080`, чтобы убедиться в том, что балансировщик работает и тестовый сервер возвращает полученный им HTTP-запрос:

![Фото](/images/image3.png)

#### Для локального запуска можно использовать `go`:

```sh 
go run ./... -config config-local.json
```

#### Для запуска тестов балансировщика:

```sh 
go test ./internal/balancer -bench=. -benchmem
```

Результаты бенчмарков представлены в файлах [ab-bench](/ab-bench) и [hey-bench](/hey-bench).

Для бенчмарков были использованы следующие команды:

```sh
ab  -m GET -n 5000 -c 1000 -H "X-API-Key: ecf01bf4-2382-4011-8255-cfb507e0da2b" http://localhost:8080/ > ab-bench
hey -m GET -n 5000 -c 1000 -H "X-API-Key: 686ef237-3d80-483b-a3b9-d064c93efcba" http://localhost:8080/ > hey-bench
```

Результаты бенчмарков и тестов в виде фотографий:

![1](/images/image1.png)
![2](/images/image2.jpeg)

Для создания клиента отправляется POST-запрос на /clients:

```sh
curl -X POST localhost:8080/client -d '{"name":"ilya", "capacity": 1000, "rate": 10}'
```

В качестве ответа будет получен ключ, который необходимо передавать для всех запросов клиента в заголовке `X-API-Key`:

```sh
{"key":"686ef237-3d80-483b-a3b9-d064c93efcba"}
```

Для работы балансировщика необходим конфигурационный файл в формате JSON:

```
{  
    "logging": "error",  // уровень логирования
    "port": 8080,        // порт для сервера балансировщика
    "endpoints": [       // список URL для серверов балансировщика
        "http://endpoint-first:80",
        "http://endpoint-second:80",
        "http://endpoint-third:80"
    ],
    "strategy": "round-robin", // стратегия работы балансирощика
    "healthInterval": "10s",   // интервал проверки здоровья серверов (ping)
    "refillInterval": "300ms", // интервал пополнения токенов для TokenBucket
    "defaults": {              // стандартные значения для ёмкости и скорости пополнения Token Bucket
        "capacity": 10,
        "rate": 1
    },
    "mode": "remote",               // режим работы балансировщика, remote - PostgreSQL, local - SQLite
    "migrationsPath": "migrations", // путь для директории с миграциями для базы данных
    "filePath": "clients.sqlite"    // путь для локального файла SQLite (режим - local)
}
```

Проект разделён на несколько пакетов:

- balancer (балансировщик);
- client (клиент);
- config (конфигурация);
- limiter (Rate limiting сервис);
- migrations (миграции для БД);
- server (сервер с graceful shutdown).

Как уже было отмечено ранее, балансировщик может работать в двух режимах:
- local;
- remote.

В режиме `local` для хранения данных клиентов (ключ, имя, ёмкость, скорость пополнения) применяется SQLite и все данные хранятся в одном файле. 

В режиме `remote` используется PostgreSQL, для него необходим файл `.env`, в котором указываются данные (хост, пользователь, пароль и т.д.) для соединения с СУБД.
