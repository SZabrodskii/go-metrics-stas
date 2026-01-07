# Профилирование и бенчмарки

## Запуск бенчмарков

```bash
# Все бенчмарки
go test -bench=. -benchmem ./...

# Бенчмарки storage
go test -bench=. -benchmem ./internal/repository/

# Бенчмарки handlers
go test -bench=. -benchmem ./internal/handler/

# Бенчмарки client (agent)
go test -bench=. -benchmem ./internal/agent/

# Бенчмарки gzip middleware
go test -bench=. -benchmem ./internal/middleware/

# Бенчмарки sync.Pool
go test -bench=. -benchmem ./internal/pool/
```

## Профилирование памяти с pprof

Сервер имеет встроенные pprof endpoints:

```bash
# Запуск сервера
go build -o metrics-server ./cmd/server && ./metrics-server

# Получение heap профиля
curl http://localhost:8080/debug/pprof/heap > profiles/heap.pprof

# Получение профиля аллокаций
curl http://localhost:8080/debug/pprof/allocs > profiles/allocs.pprof

# Анализ профиля
go tool pprof -top profiles/heap.pprof
go tool pprof -list=SendMetric profiles/heap.pprof
go tool pprof -web profiles/heap.pprof
```

## Нагрузочное тестирование

```bash
# Установка hey
go install github.com/rakyll/hey@latest

# Запуск нагрузочного теста
./scripts/load_test.sh
```

## Сравнение профилей (до и после оптимизации)

```bash
go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

**Результаты оптимизации (Iter17):**

```
Type: inuse_space
Showing nodes accounting for -2.50MB, 37.47% of 6.68MB total
Dropped 45 nodes (cum <= 0.03MB)
      flat  flat%   sum%        cum   cum%
   -1MB 15.39% 15.39%       -1MB 15.39%  bufio.NewWriterSize (inline)
 -0.50MB  7.68% 23.07%    -0.50MB  7.68%  runtime.mcall
 -0.50MB  7.67% 30.74%    -0.50MB  7.67%  runtime.malg
 -0.50MB  7.67% 38.41%    -0.50MB  7.67%  runtime.newproc1
```

Отрицательные значения показывают уменьшение использования памяти после оптимизации.

**Основные оптимизации:**
- Использование `sync.Pool` для переиспользования `bytes.Buffer` и `gzip.Writer`
- Переиспользование `gzip.Reader` в middleware декомпрессии
- Преаллокация слайсов с известной ёмкостью
