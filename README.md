# Локальное электронное голосование на Ethereum

## 1. Краткое описание проекта

Проект реализует учебную систему электронного голосования на базе локальной приватной Ethereum-сети.

Главная идея: пользователь нажимает кнопку в web-интерфейсе, frontend отправляет HTTP-запрос в Go backend, backend подписывает Ethereum-транзакцию приватным ключом выбранного локального аккаунта, отправляет транзакцию в Geth, а смарт-контракт `Voting` меняет состояние блокчейна.

Проект полностью локальный:

- не используется публичный Ethereum;
- не используется Sepolia;
- не используется MetaMask;
- не используется Remix;
- не используются Infura, Alchemy, Etherscan;
- все запускается через Docker Compose.

Итоговая схема:

```text
Browser
  |
  v
Frontend: HTML/CSS/JavaScript
  |
  v
Go backend: REST API
  |
  v
go-ethereum / JSON-RPC
  |
  v
Geth: локальная приватная Ethereum-нода
  |
  v
Solidity smart contract Voting
  |
  v
Локальный blockchain
```

## 2. Что реализовано

В проекте реализованы:

- локальная приватная Ethereum-сеть на Geth;
- Solidity-смарт-контракт `Voting`;
- Go backend, который работает с Geth через `go-ethereum`;
- REST API для frontend;
- web-интерфейс администратора;
- web-интерфейс избирателя;
- страница результатов;
- страница просмотра блокчейна;
- Docker Compose для запуска всех сервисов;
- Makefile для типовых команд;
- скрипт компиляции контракта под EVM London.

## 3. Основные роли

В системе есть четыре локальных Ethereum-аккаунта:

| Роль | Назначение | Адрес |
| --- | --- | --- |
| `admin` | Администратор голосования | `0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266` |
| `voter1` | Первый избиратель | `0x362c91Ed8B3656a4594E05D0766451B07137a51e` |
| `voter2` | Второй избиратель | `0x18Cb7058DE3576BB42987077a8138dE3eB39e5Aa` |
| `voter3` | Третий избиратель | `0xC2BaAED42B3A954e0aC9EbCB77d8a956CaF13CD1` |

Важно:

```text
Пользователь != нода.
```

В проекте одна Geth-нода, но несколько пользователей. Пользователи отличаются не отдельными нодами, а разными Ethereum-адресами и приватными ключами.

## 4. Что хранится в блокчейне

В блокчейне хранится:

- адрес администратора;
- статус голосования: активно или остановлено;
- список кандидатов;
- количество голосов за каждого кандидата;
- информация о том, голосовал ли конкретный Ethereum-адрес.

В блокчейне не хранится:

- ФИО избирателей;
- паспортные данные;
- логины и пароли;
- email;
- обычные пользовательские профили.

Формулировка для защиты:

> Система обеспечивает псевдонимность пользователей: в блокчейне сохраняются Ethereum-адреса аккаунтов, но не сохраняются ФИО или иные персональные данные. Полная криптографическая анонимность голосования в рамках проекта не реализуется.

## 5. Структура проекта

```text
blockchain_tr/
├── docker-compose.yml
├── Makefile
├── README.md
│
├── blockchain/
│   ├── Dockerfile
│   ├── entrypoint.sh
│   ├── genesis.json
│   ├── password.txt
│   ├── accounts/
│   ├── data/
│   └── keystore/
│
├── contracts/
│   ├── Voting.sol
│   └── build/
│       ├── Voting.abi
│       └── Voting.bin
│
├── backend/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   ├── config.json
│   ├── config/
│   ├── eth/
│   ├── contract/
│   ├── handlers/
│   └── models/
│
├── frontend/
│   ├── index.html
│   ├── admin.html
│   ├── voter.html
│   ├── results.html
│   ├── blockchain.html
│   ├── styles.css
│   └── app.js
│
└── scripts/
    └── compile-contract.js
```

## 6. Как работает смарт-контракт

Файл контракта:

```text
contracts/Voting.sol
```

Контракт хранит основные данные:

```solidity
address public admin;
bool public isActive;
uint256 public candidatesCount;

struct Candidate {
    uint256 id;
    string name;
    uint256 voteCount;
}

mapping(uint256 => Candidate) public candidates;
mapping(address => bool) public hasVoted;
```

Что это значит:

- `admin` - адрес администратора, который задеплоил контракт;
- `isActive` - флаг, активно ли голосование;
- `candidatesCount` - количество кандидатов;
- `Candidate` - структура кандидата;
- `candidates` - таблица кандидатов по id;
- `hasVoted` - таблица, которая хранит, голосовал ли конкретный Ethereum-адрес.

### Функции контракта

| Функция | Кто вызывает | Что делает |
| --- | --- | --- |
| `addCandidate(name)` | Только `admin` | Добавляет кандидата |
| `startVoting()` | Только `admin` | Запускает голосование |
| `stopVoting()` | Только `admin` | Останавливает голосование |
| `vote(candidateId)` | Любой voter | Голосует за кандидата |
| `getCandidate(candidateId)` | Любой | Возвращает кандидата |
| `getCandidatesCount()` | Любой | Возвращает количество кандидатов |
| `hasAddressVoted(address)` | Любой | Проверяет, голосовал ли адрес |

### Защита от повторного голосования

Ключевая проверка:

```solidity
require(!hasVoted[msg.sender], "Already voted");
```

`msg.sender` - это Ethereum-адрес, который подписал транзакцию. Если `voter1` уже голосовал, то его адрес записан в `hasVoted`, и повторная транзакция будет отклонена.

После успешного голосования контракт делает:

```solidity
hasVoted[msg.sender] = true;
candidates[candidateId].voteCount++;
```

То есть сначала адрес помечается как проголосовавший, затем увеличивается счетчик голосов кандидата.

### Почему защита находится именно в контракте

Проверку повторного голосования нельзя надежно делать только на frontend или backend, потому что такие проверки можно обойти.

Правильное место для основной проверки - смарт-контракт. Он является источником истины, а его состояние хранится в блокчейне.

Формулировка для защиты:

> Защита от повторного голосования реализована на уровне смарт-контракта через mapping `hasVoted`. Даже если пользователь обойдет frontend и напрямую отправит транзакцию, контракт отклонит повторный голос.

## 7. Как работает локальная Ethereum-сеть

За сеть отвечает папка:

```text
blockchain/
```

Основные файлы:

| Файл | Назначение |
| --- | --- |
| `Dockerfile` | Собирает Docker-образ Geth |
| `entrypoint.sh` | Инициализирует сеть и запускает Geth |
| `genesis.json` | Начальная конфигурация блокчейна |
| `password.txt` | Пароль для keystore-аккаунтов |
| `accounts/*.key` | Приватные ключи локальных аккаунтов |
| `data/` | Runtime-данные блокчейна |
| `keystore/` | Импортированные аккаунты Geth |

### genesis.json

`genesis.json` - это первый блок и начальная конфигурация сети.

В нем задаются:

- `chainId: 2025`;
- включенные hard fork правила EVM;
- Clique/PoA consensus;
- начальные балансы аккаунтов;
- адрес администратора как signer.

Пример:

```json
"alloc": {
  "f39fd6e51aad88f6f4ce6ab8827279cfffb92266": {
    "balance": "1000000000000000000000"
  }
}
```

Это значит: аккаунту администратора выдано `1000 ETH` в локальной сети.

Эти ETH не являются настоящими деньгами. Они существуют только внутри локальной приватной сети.

### entrypoint.sh

`entrypoint.sh` делает следующее:

1. Проверяет, создана ли база блокчейна.
2. Если база не создана, импортирует аккаунты из `accounts/*.key`.
3. Выполняет `geth init /genesis.json`.
4. Запускает Geth с HTTP RPC на порту `8545`.
5. Разблокирует admin-аккаунт.
6. Включает создание блоков.

### Почему используется Geth v1.13.15

Сначала использовался более новый Geth `v1.14.x`, но он выдал ошибку:

```text
only PoS networks are supported, please transition old ones with Geth v1.13.x
```

Причина: проект использует учебную Clique/PoA-сеть, а новые версии Geth жестче относятся к старым pre-merge сетям.

Решение:

```dockerfile
FROM ethereum/client-go:v1.13.15
```

Формулировка для защиты:

> Для учебной приватной Clique-сети была зафиксирована версия Geth `v1.13.15`, так как Geth `v1.14.x` не запускает старые pre-merge PoA-сети без миграции.

## 8. Как работает backend

Backend находится в папке:

```text
backend/
```

Он написан на Go и выполняет роль посредника между frontend и блокчейном.

Frontend не общается с Geth напрямую. Он отправляет обычные HTTP-запросы в backend. Backend уже сам:

- выбирает нужный приватный ключ;
- создает transactor;
- подписывает Ethereum-транзакцию;
- отправляет транзакцию в Geth;
- ждет включения транзакции в блок;
- возвращает JSON-ответ frontend-у.

### main.go

Файл:

```text
backend/main.go
```

Что делает:

1. Загружает конфигурацию.
2. Подключается к Geth RPC.
3. Создает `handlers.App`.
4. Регистрирует REST API.
5. Запускает HTTP-сервер на `:8080`.

### config/config.go

Читает:

- `rpcUrl`;
- `chainId`;
- `contractAddress`;
- приватный ключ администратора;
- приватные ключи voters.

Для учебного проекта ключи лежат в `backend/config.json`.

Важно для защиты:

> Хранение приватных ключей в конфигурационном файле используется только для локального учебного прототипа. В промышленной системе требуется защищенное хранилище ключей.

### eth/client.go

Подключается к Geth:

```go
ethclient.Dial(rpcURL)
```

Используется retry, потому что при старте Docker Compose backend может запуститься раньше, чем Geth полностью поднимет RPC.

### eth/auth.go

Создает transactor из приватного ключа.

Transactor - это объект `go-ethereum`, который умеет подписывать транзакции конкретным Ethereum-аккаунтом.

Именно здесь определяется, от чьего имени будет отправлена транзакция:

- admin вызывает `deploy`, `addCandidate`, `startVoting`, `stopVoting`;
- voter1/voter2/voter3 вызывают `vote`.

### contract/voting.go

Это wrapper для работы с контрактом.

Обычно такой файл генерируется через `abigen`, но в проекте реализован ручной wrapper поверх ABI и `bind.BoundContract`.

Почему так сделано:

- на локальной машине может не быть `abigen`;
- проект должен собираться проще;
- ABI и bytecode уже лежат в проекте;
- wrapper дает те же основные операции: deploy, transact, call.

Файл содержит:

- ABI контракта;
- embedded bytecode `Voting.bin`;
- функцию `DeployVoting`;
- методы `AddCandidate`, `StartVoting`, `StopVoting`, `Vote`;
- методы чтения состояния `GetCandidate`, `GetCandidatesCount`, `IsActive`, `HasAddressVoted`.

### handlers/

Папка `handlers/` содержит HTTP-обработчики.

| Файл | Назначение |
| --- | --- |
| `admin.go` | deploy, start, stop |
| `candidates.go` | добавление и получение кандидатов |
| `voting.go` | голосование и проверка has-voted |
| `results.go` | результаты и статус |
| `blockchain.go` | просмотр блоков, транзакций и событий |
| `app.go` | общая структура приложения и маршруты |

## 9. REST API

| Метод | Endpoint | Назначение |
| --- | --- | --- |
| `POST` | `/api/deploy` | Деплой контракта |
| `POST` | `/api/candidates` | Добавление кандидата |
| `GET` | `/api/candidates` | Список кандидатов |
| `POST` | `/api/start` | Запуск голосования |
| `POST` | `/api/stop` | Остановка голосования |
| `POST` | `/api/vote` | Голосование |
| `GET` | `/api/results` | Результаты |
| `GET` | `/api/status` | Статус системы |
| `GET` | `/api/has-voted/{voter}` | Проверка голосования адреса |
| `GET` | `/api/blockchain?limit=20` | Просмотр блоков и транзакций |

### Пример голосования

```bash
curl -X POST http://localhost:8080/api/vote \
  -H "Content-Type: application/json" \
  -d "{\"voter\":\"voter1\",\"candidateId\":1}"
```

Ответ:

```json
{
  "success": true,
  "message": "Vote accepted",
  "txHash": "0x..."
}
```

`txHash` - это hash Ethereum-транзакции. По нему можно найти транзакцию на странице блокчейна.

## 10. Как работает frontend

Frontend находится в папке:

```text
frontend/
```

Он сделан без React, на обычных HTML/CSS/JavaScript.

Страницы:

| Страница | Назначение |
| --- | --- |
| `index.html` | Главная страница |
| `admin.html` | Панель администратора |
| `voter.html` | Страница голосования |
| `results.html` | Результаты |
| `blockchain.html` | Просмотр блокчейна |
| `app.js` | Вызовы REST API и логика интерфейса |
| `styles.css` | Стили |

### admin.html

Администратор может:

- задеплоить контракт;
- добавить кандидата;
- запустить голосование;
- остановить голосование;
- увидеть адрес контракта.

### voter.html

Пользователь может:

- выбрать `voter1`, `voter2` или `voter3`;
- выбрать кандидата;
- отправить голос;
- увидеть успешный ответ или ошибку `Already voted`.

### results.html

Показывает список кандидатов и количество голосов.

Данные берутся не из frontend-памяти, а через backend из смарт-контракта.

### blockchain.html

Это встроенный простой block explorer.

Он показывает:

- последние блоки с транзакциями;
- hash блока;
- hash транзакции;
- отправителя транзакции;
- получателя транзакции;
- вызванную функцию контракта;
- аргументы функции;
- статус receipt;
- gas used;
- события контракта.

Например:

```text
addCandidate
success
from admin
to Voting contract
name=Сергей
CandidateAdded
```

Это удобно для защиты, потому что можно показать, что действия frontend действительно превращаются в Ethereum-транзакции.

## 11. Как backend вызывает контракт

Разберем пример голосования.

Пользователь на странице `voter.html` выбирает:

```text
voter1
candidateId = 1
```

Frontend отправляет:

```http
POST /api/vote
Content-Type: application/json

{
  "voter": "voter1",
  "candidateId": 1
}
```

Backend:

1. Получает JSON.
2. Находит приватный ключ `voter1`.
3. Создает transactor.
4. Проверяет, активно ли голосование.
5. Проверяет, существует ли кандидат.
6. Проверяет, голосовал ли адрес раньше.
7. Отправляет транзакцию `vote(1)` в контракт.
8. Ждет, пока Geth включит транзакцию в блок.
9. Возвращает frontend-у JSON.

Контракт:

1. Проверяет `isActive`.
2. Проверяет `hasVoted[msg.sender]`.
3. Проверяет `candidateId`.
4. Записывает `hasVoted[msg.sender] = true`.
5. Увеличивает `voteCount`.
6. Создает событие `VoteCast`.

## 12. Что такое транзакция в этом проекте

Транзакция - это подписанная операция, которая меняет состояние блокчейна.

В проекте транзакциями являются:

- деплой контракта;
- добавление кандидата;
- запуск голосования;
- остановка голосования;
- голосование.

Чтение данных транзакцией не является:

- `GET /api/results`;
- `GET /api/candidates`;
- `GET /api/status`;
- `hasAddressVoted`.

Чтение выполняется бесплатно с точки зрения blockchain state, потому что оно не меняет блокчейн.

## 13. Что такое gas

`gas used` - это количество вычислительных единиц, которое EVM потратила на выполнение транзакции.

Это не wei и не ETH само по себе.

Стоимость транзакции считается так:

```text
стоимость = gas used * gas price
```

Например:

```text
gas used = 79893
gas price = 1 gwei
1 gwei = 1 000 000 000 wei
```

Тогда:

```text
79893 * 1 000 000 000 wei = 79 893 000 000 000 wei
```

В ETH:

```text
0.000079893 ETH
```

Но в этом проекте это не реальные деньги. Это учебные ETH в локальной приватной сети.

### Gas limit и gas used

`gas limit` - максимум gas, который отправитель разрешает потратить.

`gas used` - сколько реально было потрачено.

В backend установлен лимит:

```go
auth.GasLimit = 5_000_000
```

Это значит: транзакция может потратить до `5 000 000 gas`, но фактически тратит меньше.

## 14. Что такое miner в этом проекте

В публичном Ethereum раньше miner означал участника, который создавал блоки в Proof-of-Work.

В этом проекте используется локальная Clique/PoA-сеть. Поэтому здесь корректнее говорить не miner, а локальный валидатор или signer.

На странице блокчейна поле может отображаться так:

```text
miner 0x0000000000000000000000000000000000000000
```

Это нормально для такой конфигурации Geth/Clique. Блоки все равно создает локальная Geth-нода, а фактическая информация о signer хранится в служебных данных блока.

Формулировка для защиты:

> В данном учебном проекте блоки создает локальная Geth-нода в приватной Clique/PoA-сети. Поле miner в интерфейсе отображает техническое поле блока, но майнинга в смысле публичного Proof-of-Work здесь нет.

## 15. Как показать работу блокчейна преподавателю

Открыть:

```text
http://localhost:3000/blockchain.html
```

Показать сценарий:

1. На странице администратора нажать `Задеплоить контракт`.
2. Открыть страницу `Блокчейн`.
3. Показать транзакцию `constructor / deploy Voting`.
4. Добавить кандидата.
5. Показать транзакцию `addCandidate`.
6. Показать событие `CandidateAdded`.
7. Запустить голосование.
8. Показать транзакцию `startVoting`.
9. Проголосовать от `voter1`.
10. Показать транзакцию `vote`.
11. Показать событие `VoteCast`, где виден адрес voter и `candidateId`.
12. Повторить голос от `voter1`.
13. Показать ошибку `Already voted` в интерфейсе.

Пример объяснения:

> На этой странице видно, что действие в web-интерфейсе не просто изменило состояние frontend. Оно было оформлено как Ethereum-транзакция, подписано приватным ключом локального аккаунта, отправлено в Geth, включено в блок и выполнено смарт-контрактом. События контракта отображаются из receipt logs.

## 16. Как смотреть данные через curl

Статус:

```bash
curl http://localhost:8080/api/status
```

Кандидаты:

```bash
curl http://localhost:8080/api/candidates
```

Результаты:

```bash
curl http://localhost:8080/api/results
```

Блокчейн:

```bash
curl "http://localhost:8080/api/blockchain?limit=20"
```

Проверка Geth RPC напрямую:

```bash
curl -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}"
```

## 17. Docker Compose

Запуск:

```bash
docker compose up --build
```

Остановка:

```bash
docker compose down
```

Логи:

```bash
docker compose logs -f
```

Логи Geth:

```bash
docker compose logs -f geth
```

Адреса сервисов:

| Сервис | URL |
| --- | --- |
| Frontend | `http://localhost:3000` |
| Backend | `http://localhost:8080` |
| Geth RPC | `http://localhost:8545` |

## 18. Makefile

Команды:

```bash
make up
make down
make logs
make restart
make compile-contract
```

`make compile-contract` запускает:

```bash
node scripts/compile-contract.js
```

## 19. Компиляция контракта

Скрипт:

```text
scripts/compile-contract.js
```

Он компилирует `contracts/Voting.sol` через `solc-js` в standard JSON режиме.

Важно:

```json
"evmVersion": "london"
```

Почему это нужно:

Новые версии Solidity по умолчанию могут генерировать bytecode с opcode `PUSH0`, который появился в Shanghai. Но локальная сеть настроена как London. Если скомпилировать контракт под Shanghai и попытаться задеплоить в London-сеть, транзакция деплоя может завершиться:

```text
transaction ... reverted
```

Решение:

```text
Компилировать контракт под evmVersion = london.
```

Формулировка для защиты:

> При реализации была выявлена несовместимость bytecode Solidity 0.8.26 с London-конфигурацией локальной сети из-за Shanghai opcode `PUSH0`. Проблема решена компиляцией контракта через solc standard JSON с параметром `evmVersion: "london"`.

## 20. Важные проблемы, которые были решены

### 20.1. Несовпадение адресов аккаунтов

Проблема:

В `genesis.json` были указаны адреса, которые не соответствовали приватным ключам voter-аккаунтов.

Следствие:

Geth импортировал аккаунты, но реальные адреса voters не получали стартовый баланс.

Решение:

Адреса в `genesis.json` были приведены в соответствие с фактически импортируемыми приватными ключами:

```text
voter1 -> 0x362c91...
voter2 -> 0x18Cb70...
voter3 -> 0xC2BaAE...
```

### 20.2. Geth v1.14.x не запускал Clique-сеть

Проблема:

```text
only PoS networks are supported
```

Решение:

Использовать:

```dockerfile
FROM ethereum/client-go:v1.13.15
```

### 20.3. Revert при деплое контракта

Проблема:

Деплой возвращал:

```text
transaction ... reverted
```

Причина:

Контракт был скомпилирован под более новую EVM-версию, чем сеть.

Решение:

Компиляция с:

```json
"evmVersion": "london"
```

### 20.4. Данные Geth нужно пересоздавать после изменения genesis

`genesis.json` применяется только при первом `geth init`.

Если изменить genesis, аккаунты или версию Geth, нужно очистить runtime-данные:

```powershell
docker compose down
Get-ChildItem .\blockchain\data -Force | Where-Object Name -ne '.gitkeep' | Remove-Item -Recurse -Force
Get-ChildItem .\blockchain\keystore -Force | Where-Object Name -ne '.gitkeep' | Remove-Item -Recurse -Force
docker compose up --build
```

### 20.5. Ошибка на Mac: `Killed` при import и `no key for given address`

На Mac Docker может завершить `geth account import` сообщением:

```text
Killed
```

После этого Geth падает при запуске:

```text
Fatal: Failed to unlock account ... (no key for given address or file)
```

Причина:

- Geth попытался импортировать приватный ключ в keystore;
- процесс импорта был убит Docker-ом;
- keystore-файл администратора не появился;
- затем Geth не смог разблокировать admin-аккаунт для Clique/PoA.

Что сделано в проекте:

- в `blockchain/keystore/` добавлены готовые учебные keystore-файлы;
- `entrypoint.sh` сначала проверяет наличие admin-keystore;
- если keystore уже есть, повторный импорт не выполняется;
- если импорт все же нужен и падает, контейнер теперь падает сразу, а не продолжает запуск с поврежденным состоянием.

Если ошибка уже произошла, очистите runtime-данные и запустите заново:

```bash
docker compose down
rm -rf blockchain/data/geth blockchain/data/geth.ipc blockchain/data/history
docker compose up --build
```

Не удаляйте `blockchain/keystore/`, если используете готовые keystore-файлы из проекта.

### 20.6. Ошибка на Mac: `geth.ipc: bind: operation not supported`

На Docker Desktop for Mac bind-mounted папки могут не поддерживать создание Unix socket-файлов. Geth по умолчанию пытается создать IPC socket:

```text
/root/.ethereum/geth.ipc
```

Из-за этого контейнер может падать:

```text
Fatal: Error starting protocol stack: listen unix /root/.ethereum/geth.ipc: bind: operation not supported
```

IPC в этом проекте не нужен, потому что backend общается с Geth через HTTP JSON-RPC:

```text
http://geth:8545
```

Поэтому в `blockchain/entrypoint.sh` добавлен флаг:

```bash
--ipcdisable
```

После изменения перезапустите контейнер:

```bash
docker compose down
docker compose up --build
```

## 21. Демонстрационный сценарий защиты

1. Запустить проект:

```bash
docker compose up --build
```

2. Открыть:

```text
http://localhost:3000
```

3. Перейти в панель администратора.
4. Нажать `Задеплоить контракт`.
5. Показать адрес контракта.
6. Перейти на страницу `Блокчейн`.
7. Показать транзакцию деплоя.
8. Вернуться в админку.
9. Добавить кандидатов: `Иванов`, `Петров`, `Сидоров`.
10. Показать транзакции `addCandidate` и события `CandidateAdded`.
11. Запустить голосование.
12. Показать транзакцию `startVoting`.
13. Перейти на страницу голосования.
14. Выбрать `voter1` и проголосовать за Иванова.
15. Выбрать `voter2` и проголосовать за Петрова.
16. Повторить голосование от `voter1`.
17. Показать ошибку `Already voted`.
18. Перейти на страницу результатов.
19. Показать количество голосов.
20. Перейти на страницу `Блокчейн`.
21. Показать транзакции `vote` и события `VoteCast`.
22. Остановить голосование.
23. Показать транзакцию `stopVoting`.

## 22. Ключевые тезисы для защиты

Можно использовать такие формулировки:

> В проекте реализована локальная приватная Ethereum-сеть на базе Geth. Пользователи представлены разными Ethereum-аккаунтами, а не разными нодами.

> Смарт-контракт `Voting` хранит список кандидатов, количество голосов и mapping проголосовавших адресов. Это позволяет защититься от повторного голосования.

> Backend на Go подписывает транзакции приватными ключами локальных аккаунтов и отправляет их в Geth через JSON-RPC.

> Frontend не работает напрямую с блокчейном. Он использует REST API backend-а.

> Все изменения состояния голосования фиксируются в блокчейне как транзакции.

> Просмотр блокчейна реализован в отдельной странице, где декодируются функции контракта и события из receipt logs.

> Система является учебным прототипом и обеспечивает псевдонимность, но не полную криптографическую анонимность.

## 23. Ограничения проекта

Проект является учебным прототипом.

Ограничения:

- используется одна локальная Geth-нода;
- нет регистрации реальных граждан;
- нет проверки документов;
- нет zk-доказательств;
- нет blind signatures;
- нет commit-reveal схемы;
- Ethereum-адрес голосующего виден в блокчейне;
- приватные ключи хранятся в конфигурации только для удобства демонстрации.

## 24. Что можно улучшить в будущем

Возможные направления развития:

- добавить полноценную регистрацию пользователей;
- хранить ключи в защищенном хранилище;
- добавить роли и авторизацию администратора;
- добавить несколько Geth-нод;
- добавить commit-reveal схему;
- добавить zk-доказательства для анонимности;
- добавить независимый block explorer;
- добавить тесты смарт-контракта;
- добавить хранение адреса контракта после рестарта backend.

## 25. Что такое Geth

Geth - это официальная реализация Ethereum-ноды на языке Go.

Ethereum-нода умеет:

- хранить блоки;
- хранить состояние контрактов;
- принимать транзакции;
- проверять подписи транзакций;
- выполнять EVM bytecode;
- создавать новые блоки в приватной сети;
- предоставлять JSON-RPC API для внешних программ.

В этом проекте Geth работает как локальная приватная Ethereum-сеть.

Файл:

```text
blockchain/Dockerfile
```

задает Docker-образ:

```dockerfile
FROM ethereum/client-go:v1.13.15
```

Файл:

```text
blockchain/entrypoint.sh
```

запускает Geth с HTTP RPC:

```bash
--http
--http.addr 0.0.0.0
--http.port 8545
--http.api eth,net,web3,personal,miner,txpool,clique
```

Это значит, что backend может обращаться к Geth по HTTP и вызывать методы Ethereum JSON-RPC.

В Docker Compose Geth доступен:

```text
для backend внутри Docker: http://geth:8545
для браузера/терминала хоста: http://localhost:8545
```

Важно:

```text
Geth - это не смарт-контракт.
Geth - это Ethereum-нода, которая хранит блокчейн и выполняет контракты.
```

Формулировка для защиты:

> Geth в проекте выполняет роль локальной Ethereum-ноды. Он хранит блоки, принимает подписанные транзакции от backend-а, выполняет bytecode смарт-контракта в EVM и предоставляет JSON-RPC интерфейс на порту 8545.

## 26. Что такое go-ethereum

`go-ethereum` - это Go-библиотека из проекта Geth.

Она позволяет Go-приложению работать с Ethereum без ручной сборки JSON-RPC запросов.

В `backend/go.mod` подключена зависимость:

```go
require github.com/ethereum/go-ethereum v1.14.13
```

В проекте используются такие части `go-ethereum`:

| Пакет | Для чего нужен |
| --- | --- |
| `ethclient` | Подключение к Geth JSON-RPC |
| `accounts/abi` | Работа с ABI контракта |
| `accounts/abi/bind` | Deploy, call и transact с контрактом |
| `common` | Ethereum-адреса и hash |
| `core/types` | Транзакции, блоки, receipts |
| `crypto` | Приватные ключи и адреса |

Пример из проекта:

```go
client, err := ethclient.Dial(rpcURL)
```

Это создает Go-клиент к Geth.

Другой пример:

```go
auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
```

Это создает объект, который умеет подписывать транзакции приватным ключом.

Важно:

```text
go-ethereum - это библиотека для backend-а.
Geth - это запущенная Ethereum-нода.
```

То есть:

```text
Go backend использует go-ethereum, чтобы общаться с Geth.
```

Формулировка для защиты:

> Backend написан на Go и использует библиотеку `go-ethereum`. Через нее backend подключается к Geth, подписывает транзакции приватными ключами, деплоит контракт и вызывает функции контракта.

## 27. Что такое ABI

ABI означает Application Binary Interface.

ABI - это JSON-описание интерфейса смарт-контракта.

В ABI описано:

- какие функции есть у контракта;
- какие аргументы принимает каждая функция;
- какие значения возвращает функция;
- какие события может создавать контракт;
- какие поля события являются `indexed`.

Пример фрагмента ABI:

```json
{
  "inputs": [
    {
      "internalType": "uint256",
      "name": "candidateId",
      "type": "uint256"
    }
  ],
  "name": "vote",
  "outputs": [],
  "stateMutability": "nonpayable",
  "type": "function"
}
```

Этот фрагмент говорит backend-у:

```text
У контракта есть функция vote(uint256 candidateId).
```

Зачем ABI нужен backend-у:

1. Чтобы превратить вызов Go-кода `Vote(1)` в Ethereum calldata.
2. Чтобы понять, какую функцию вызвать в контракте.
3. Чтобы распаковать результат `getCandidate`.
4. Чтобы декодировать события `CandidateAdded`, `VoteCast`.

Пример:

Когда backend вызывает:

```go
instance.Vote(auth, big.NewInt(1))
```

go-ethereum через ABI кодирует это в бинарные данные транзакции:

```text
function selector + encoded argument
```

Упрощенно:

```text
vote(1) -> 0x0121b93f0000000000000000000000000000000000000000000000000000000000000001
```

Первые 4 байта - selector функции. Остальное - ABI-encoded аргументы.

На странице `blockchain.html` backend делает обратную операцию:

```text
calldata -> имя функции и аргументы
```

И поэтому в интерфейсе видно:

```text
vote
candidateId=1
```

## 28. Что такое abigen

`abigen` - это утилита из go-ethereum.

Она берет:

- ABI контракта;
- bytecode контракта;
- имя Go-пакета;
- имя Go-типа;

и генерирует Go-файл для удобной работы с контрактом.

Типичная команда:

```bash
abigen \
  --abi contracts/build/Voting.abi \
  --bin contracts/build/Voting.bin \
  --pkg contract \
  --type Voting \
  --out backend/contract/voting_abigen.go
```

После этого можно было бы получить автоматически сгенерированные методы:

```go
DeployVoting(...)
voting.Vote(...)
voting.AddCandidate(...)
voting.GetCandidate(...)
```

В нашем проекте `abigen` не обязателен.

Почему:

- на машине разработчика может не быть установлен `abigen`;
- для учебного проекта важно, чтобы backend собирался проще;
- мы вручную сделали wrapper в `backend/contract/voting.go`;
- этот wrapper использует тот же ABI и `bind.BoundContract`.

То есть наша реализация делает то же самое по смыслу, но без обязательной генерации файла через `abigen`.

Формулировка для защиты:

> В промышленной разработке Go bindings обычно генерируются через `abigen`. В проекте реализован ручной wrapper поверх ABI и `bind.BoundContract`, чтобы проект можно было собрать без установленного `abigen`. При этом принцип работы остается тем же: ABI используется для кодирования вызовов функций и декодирования ответов.

## 29. Откуда берутся ETH_RPC_URL, CHAIN_ID, ADMIN_PRIVATE_KEY и CONTRACT_ADDRESS

В проекте есть два источника конфигурации:

1. `backend/config.json`;
2. переменные окружения из `docker-compose.yml`.

Backend читает их в файле:

```text
backend/config/config.go
```

### 29.1. ETH_RPC_URL

`ETH_RPC_URL` - это адрес Geth JSON-RPC.

В `backend/config.json`:

```json
"rpcUrl": "http://geth:8545"
```

В `docker-compose.yml`:

```yaml
environment:
  ETH_RPC_URL: "http://geth:8545"
```

Почему именно `http://geth:8545`:

- `geth` - имя сервиса в Docker Compose;
- Docker Compose создает внутреннюю сеть;
- backend может обращаться к контейнеру Geth по имени сервиса;
- `8545` - HTTP RPC порт Geth.

Если запускать backend не в Docker, а с хоста, тогда обычно нужен:

```text
http://localhost:8545
```

### 29.2. CHAIN_ID

`CHAIN_ID` - идентификатор Ethereum-сети.

В `blockchain/genesis.json`:

```json
"chainId": 2025
```

В `backend/config.json`:

```json
"chainId": 2025
```

В `docker-compose.yml`:

```yaml
CHAIN_ID: "2025"
```

Зачем нужен chainId:

- он входит в подпись транзакции;
- защищает от replay attack;
- гарантирует, что транзакция предназначена именно для нашей локальной сети.

Если chainId в backend и genesis не совпадут, транзакции будут подписываться некорректно для этой сети.

### 29.3. ADMIN_PRIVATE_KEY

`ADMIN_PRIVATE_KEY` - приватный ключ аккаунта администратора.

В проекте он хранится в:

```text
backend/config.json
```

Поле:

```json
"adminPrivateKey": "ac0974..."
```

Этот же ключ импортируется Geth из:

```text
blockchain/accounts/admin.key
```

Из приватного ключа вычисляется Ethereum-адрес:

```text
0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266
```

Admin используется для:

- деплоя контракта;
- добавления кандидатов;
- запуска голосования;
- остановки голосования.

Важно:

```text
Приватный ключ нельзя публиковать в реальном проекте.
```

Здесь он хранится в конфиге только потому, что проект учебный и локальный.

### 29.4. VOTER_PRIVATE_KEY

Для voters ключи лежат в:

```json
"voters": {
  "voter1": "...",
  "voter2": "...",
  "voter3": "..."
}
```

Когда пользователь выбирает `voter1` во frontend, backend берет приватный ключ `voter1` и подписывает транзакцию именно этим ключом.

Поэтому в блокчейне в поле `from` будет адрес `voter1`.

### 29.5. CONTRACT_ADDRESS

`CONTRACT_ADDRESS` - адрес уже развернутого контракта.

В `backend/config.json` по умолчанию:

```json
"contractAddress": ""
```

Почему пусто:

- при первом запуске контракт еще не развернут;
- пользователь нажимает `Задеплоить контракт`;
- backend отправляет deploy-транзакцию;
- Geth включает ее в блок;
- из receipt получается адрес контракта;
- backend сохраняет адрес в памяти.

После deploy frontend получает:

```json
{
  "success": true,
  "contractAddress": "0x..."
}
```

Важно:

```text
Сейчас адрес контракта хранится в памяти backend-процесса.
```

Если backend перезапустить, он забудет адрес, если он не прописан в `config.json` или переменной окружения `CONTRACT_ADDRESS`.

Для учебной демонстрации можно просто нажать deploy заново.

## 30. Полный flow: что происходит после нажатия кнопки

Разберем полный путь на примере голосования.

### Шаг 1. Пользователь нажимает кнопку

На странице:

```text
frontend/voter.html
```

пользователь выбирает:

```text
voter1
candidateId = 1
```

и нажимает:

```text
Проголосовать
```

### Шаг 2. Frontend отправляет HTTP-запрос в backend

Файл:

```text
frontend/app.js
```

отправляет:

```http
POST http://localhost:8080/api/vote
Content-Type: application/json

{
  "voter": "voter1",
  "candidateId": 1
}
```

На этом этапе frontend еще не работает с блокчейном напрямую. Это обычный HTTP-запрос.

### Шаг 3. Backend принимает запрос

Файл:

```text
backend/handlers/voting.go
```

Функция:

```go
func (a *App) Vote(w http.ResponseWriter, r *http.Request)
```

делает:

1. читает JSON;
2. проверяет `voter`;
3. проверяет `candidateId`;
4. получает объект контракта;
5. проверяет, активно ли голосование;
6. проверяет, существует ли кандидат;
7. проверяет, голосовал ли адрес раньше.

Часть проверок продублирована в backend для удобного сообщения пользователю.

Но главная защита все равно находится в Solidity-контракте.

### Шаг 4. Backend выбирает приватный ключ voter

Если пришло:

```json
{
  "voter": "voter1"
}
```

backend берет из `config.json` ключ:

```json
"voter1": "59c699..."
```

Затем создает transactor:

```go
auth, voterAddress, err := a.voterAuth(ctx, req.Voter)
```

Transactor - это объект, который будет подписывать Ethereum-транзакцию приватным ключом `voter1`.

### Шаг 5. Backend формирует вызов функции контракта

Вызов:

```go
tx, err := instance.Vote(auth, big.NewInt(1))
```

по ABI превращается в calldata:

```text
selector функции vote + ABI-encoded candidateId
```

Это уже не JSON. Это бинарные данные Ethereum-транзакции.

### Шаг 6. go-ethereum отправляет транзакцию в Geth

`go-ethereum` подписывает транзакцию и отправляет ее в Geth через JSON-RPC.

Упрощенно backend вызывает метод:

```text
eth_sendRawTransaction
```

Тело запроса выглядит примерно так:

```json
{
  "jsonrpc": "2.0",
  "method": "eth_sendRawTransaction",
  "params": [
    "0x02f8..."
  ],
  "id": 1
}
```

`0x02f8...` - это сериализованная подписанная Ethereum-транзакция.

Backend не отправляет в Geth красивый JSON вида:

```json
{
  "function": "vote",
  "candidateId": 1
}
```

Такого в Ethereum нет. Geth получает сырую подписанную транзакцию.

### Шаг 7. Geth проверяет транзакцию

Geth проверяет:

- корректна ли подпись;
- хватает ли баланса на gas;
- правильный ли nonce;
- правильный ли chainId;
- существует ли адрес контракта;
- можно ли выполнить calldata.

Если все корректно, транзакция попадает в pending pool.

### Шаг 8. Geth включает транзакцию в блок

Так как у нас локальная сеть, блоки создаются быстро.

Geth берет транзакцию из pool, выполняет ее в EVM и добавляет в новый блок.

### Шаг 9. EVM выполняет Solidity-код

Внутри EVM выполняется функция:

```solidity
vote(1)
```

Контракт проверяет:

```solidity
require(isActive, "Voting is not active");
require(!hasVoted[msg.sender], "Already voted");
require(candidateId > 0 && candidateId <= candidatesCount, "Candidate does not exist");
```

Если все проверки прошли:

```solidity
hasVoted[msg.sender] = true;
candidates[candidateId].voteCount++;
emit VoteCast(msg.sender, candidateId);
```

Состояние блокчейна изменено.

### Шаг 10. Backend ждет receipt

Backend вызывает:

```go
bind.WaitMined(ctx, a.client, tx)
```

Receipt показывает:

- в каком блоке транзакция;
- успешна ли она;
- сколько gas потрачено;
- какие события были созданы.

Если receipt status равен success, backend возвращает:

```json
{
  "success": true,
  "message": "Vote accepted",
  "txHash": "0x..."
}
```

### Шаг 11. Frontend показывает результат

Frontend получает JSON и показывает пользователю:

```text
Vote accepted
```

### Шаг 12. На странице blockchain.html видно доказательство

На странице:

```text
http://localhost:3000/blockchain.html
```

будет видно:

```text
vote
success
from voter1 (...)
to Voting contract (...)
candidateId=1
VoteCast
```

Это показывает, что голосование действительно прошло через блокчейн.

## 31. Как backend общается с Geth

Backend общается с Geth через HTTP JSON-RPC.

Адрес:

```text
http://geth:8545
```

внутри Docker Compose.

Снаружи, с компьютера:

```text
http://localhost:8545
```

Но в Go-коде мы почти не пишем JSON-RPC вручную. За нас это делает библиотека `go-ethereum`.

Примеры операций:

| Что делает backend | Что примерно вызывает в Geth |
| --- | --- |
| Узнать последний блок | `eth_blockNumber` |
| Прочитать блок | `eth_getBlockByNumber` |
| Отправить транзакцию | `eth_sendRawTransaction` |
| Получить receipt | `eth_getTransactionReceipt` |
| Прочитать view-функцию | `eth_call` |
| Получить баланс | `eth_getBalance` |

### Read-only вызовы

Например:

```go
instance.GetCandidatesCount(...)
```

Это чтение состояния. Оно не создает транзакцию.

Через JSON-RPC это примерно:

```text
eth_call
```

Такой вызов:

- не попадает в блок;
- не меняет состояние;
- не требует gas оплаты;
- просто симулирует чтение текущего состояния.

### Write-вызовы

Например:

```go
instance.Vote(auth, candidateID)
```

Это запись состояния.

Она:

- создает транзакцию;
- подписывается приватным ключом;
- отправляется через `eth_sendRawTransaction`;
- попадает в блок;
- тратит gas;
- меняет storage контракта.

Формулировка для защиты:

> Backend общается с Geth через JSON-RPC, но напрямую JSON-запросы почти не формирует. Для этого используется библиотека `go-ethereum`, которая предоставляет высокоуровневые методы для чтения блоков, отправки транзакций, ожидания receipt и вызова функций контракта по ABI.

## 32. Что происходит при деплое контракта

Деплой начинается, когда пользователь нажимает в админке:

```text
Задеплоить контракт
```

### Шаг 1. Frontend отправляет запрос

```http
POST /api/deploy
```

### Шаг 2. Backend создает admin transactor

Файл:

```text
backend/handlers/admin.go
```

создает transactor из `adminPrivateKey`.

Именно этот аккаунт станет `msg.sender` в constructor контракта.

### Шаг 3. Backend берет bytecode

Файл:

```text
backend/contract/Voting.bin
```

содержит скомпилированный bytecode контракта.

Он встраивается в Go-бинарник через:

```go
//go:embed Voting.bin
var votingBin string
```

### Шаг 4. Backend формирует deploy-транзакцию

В `backend/contract/voting.go`:

```go
bind.DeployContract(auth, parsed, common.FromHex(bytecode), backend)
```

Deploy-транзакция отличается от обычной:

```text
у нее нет поля to
```

Потому что контракт еще не существует.

### Шаг 5. Geth выполняет constructor

Во время деплоя EVM выполняет:

```solidity
constructor() {
    admin = msg.sender;
    isActive = false;
}
```

Так как транзакцию подписал admin, в контракте сохраняется:

```text
admin = 0xf39Fd6e...
```

### Шаг 6. Появляется адрес контракта

После включения deploy-транзакции в блок Geth возвращает receipt.

В receipt есть:

```text
contractAddress
```

Backend сохраняет этот адрес в памяти и возвращает frontend-у.

### Шаг 7. Все следующие вызовы идут по этому адресу

После деплоя:

```text
addCandidate
startVoting
stopVoting
vote
getCandidate
```

отправляются уже на адрес контракта.

На странице `blockchain.html` деплой виден как:

```text
constructor / deploy Voting
```

Обычные вызовы видны как:

```text
addCandidate
startVoting
vote
```
