# Package Reference — Building Materials Marketplace

Local reference for the API & usage patterns of each project dependency. **AI/developer: use this document as the first source of truth before searching external documentation.** If an API isn't covered here, check the official docs — then add the findings to this document.

General rule: pin all versions in go.mod; do not upgrade major versions without approval.

---

## 1. Fiber v3 (`github.com/gofiber/fiber/v3`) — HTTP framework

**Status: v3 is STABLE (v3.0.0+, latest v3.1.x). Requires Go 1.25+.**

Most important changes from v2 (do not use v2 patterns):

```go
// Handler: fiber.Ctx is an INTERFACE, no pointer
app.Get("/products/:id", func(c fiber.Ctx) error {   // NOT *fiber.Ctx
    return c.JSON(fiber.Map{"ok": true})
})
```

**fiber.Ctx satisfies context.Context** — pass it directly to pgx/redis/gRPC, no `c.UserContext()` needed:

```go
rows, err := pool.Query(c, "SELECT ...")   // c used directly as ctx
```

**Unified binding** (replaces BodyParser/QueryParser/ParamsParser from v2):

```go
type CreateProductReq struct {
    ID    int    `params:"id"`
    Sort  string `query:"sort"`
    Name  string `json:"name" validate:"required,min=3"`
}
var req CreateProductReq
if err := c.Bind().Body(&req); err != nil { ... }   // also: .Query(&req), .URI(&req), .Header(&req)
```

Setup, route group, middleware:

```go
app := fiber.New(fiber.Config{AppName: "marketplace-api", BodyLimit: 2 << 20})
v1 := app.Group("/api/v1")
admin := v1.Group("/admin", mw.Auth(), mw.RequireRole("admin"))

// Middleware signature same as handler:
func RequestID() fiber.Handler {
    return func(c fiber.Ctx) error {
        // c.Get(header), c.Set(header), c.Locals(key, val), c.Next()
        return c.Next()
    }
}
app.Listen(":8080", fiber.ListenConfig{DisableStartupMessage: false}) // Listen merged with config
```

v3 gotchas:
- `app.Static()` REMOVED → use the `static` middleware. `app.Mount()` removed → sub-app via `app.Use()`.
- Prefix middleware is stricter: `app.Use("/api", mw)` only matches `/api` and `/api/...` (not `/apiv1`).
- Logger middleware: field `Output` → `Stream`.
- `fiber.Ctx` is NOT thread-safe & is reused across requests — don't keep a reference to c or the result of `c.Params()` outside the handler; copy the value.
- Generic helpers available (e.g. `fiber.Query[int](c, "page", 1)`, `fiber.Params[int](c, "id")`) replacing QueryInt/ParamsInt.
- GET automatically registers HEAD.
- Get auth claims from `c.Locals("user_id")`, set by our own middleware (see pkg/jwt) — don't use fiber contrib jwt so we keep full control of the blacklist.

---

## 2. pgx v5 (`github.com/jackc/pgx/v5`) — PostgreSQL driver

```go
import "github.com/jackc/pgx/v5/pgxpool"

cfg, _ := pgxpool.ParseConfig(dsn)          // dsn: postgres://user:pass@host:5432/db?sslmode=disable
cfg.MaxConns = 20
pool, err := pgxpool.NewWithConfig(ctx, cfg)
defer pool.Close()
err = pool.Ping(ctx)
```

Query & scan:

```go
row := pool.QueryRow(ctx, `SELECT id, name FROM users WHERE id=$1`, id)
err := row.Scan(&u.ID, &u.Name)             // pgx.ErrNoRows if empty → map to domain errs.ErrNotFound

rows, err := pool.Query(ctx, `SELECT ...`)
defer rows.Close()
users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])  // helper to scan into struct (tag db:"col")
```

Transactions (mandatory pattern for checkout/settlement):

```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)                       // no-op if already committed
// ... tx.Exec / tx.QueryRow ...
return tx.Commit(ctx)
```

Gotchas: placeholders `$1..$n` (not `?`); numeric → scan into `pgtype.Numeric` or int64 (we use int64 rupiah); UUID scans directly into `uuid.UUID` (pgx v5 has native support); unique constraint errors: check `pgconn.PgError` code `23505` → map to ErrConflict.

## 3. sqlc (`github.com/sqlc-dev/sqlc`) — type-safe query generator

`sqlc.yaml` at root; SQL queries in `internal/repository/postgres/queries/*.sql`; output package `sqlcgen`.

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/repository/postgres/queries"
    schema: "migrations"
    gen:
      go:
        package: "sqlcgen"
        out: "internal/repository/postgres/sqlcgen"
        sql_package: "pgx/v5"
        emit_pointers_for_null_types: true
```

Query annotations: `-- name: GetUserByEmail :one`, `:many`, `:exec`, `:execrows`. Regenerate: `make sqlc` (`sqlc generate`). Repository implementations wrap sqlcgen + map to domain entities. Complex dynamic queries (catalog filters) may be hand-written pgx outside sqlc.

## 4. go-redis v9 (`github.com/redis/go-redis/v9`)

```go
rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
err := rdb.Ping(ctx).Err()

rdb.Set(ctx, key, val, 10*time.Minute)
val, err := rdb.Get(ctx, key).Result()       // redis.Nil if missing → DO NOT treat as a system error
rdb.Del(ctx, key)
ok, err := rdb.SetNX(ctx, "stock:"+id, 1, 10*time.Second).Result()  // distributed lock (FR-5.4)
rdb.Incr(ctx, key); rdb.Expire(ctx, key, ttl)                        // rate limit / brute force counter
```

Gotchas: always distinguish `err == redis.Nil` from a connection error; release locks using Lua compare-and-del (store a random token as the value, delete only if the token matches) so we don't delete another process's lock.

## 5. Asynq (`github.com/hibiken/asynq`) — background jobs (Redis-backed)

```go
// Enqueue (infra/queue/client.go)
client := asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
payload, _ := json.Marshal(EmailSendPayload{...})
task := asynq.NewTask("email:send", payload)
info, err := client.Enqueue(task, asynq.MaxRetry(5), asynq.Queue("default"),
    asynq.ProcessIn(2*time.Hour))            // delay — used as an alternative for order:expire

// Worker (cmd/worker)
srv := asynq.NewServer(asynq.RedisClientOpt{Addr: addr},
    asynq.Config{Concurrency: 10, Queues: map[string]int{"critical": 6, "default": 3, "low": 1}})
mux := asynq.NewServeMux()
mux.HandleFunc("email:send", handleEmailSend)   // func(ctx context.Context, t *asynq.Task) error
srv.Run(mux)

// Scheduler (cron) — same process as the worker
sched := asynq.NewScheduler(asynq.RedisClientOpt{Addr: addr}, nil)
sched.Register("* * * * *", asynq.NewTask("order:expire", nil))       // every minute
sched.Register("0 8 * * *", asynq.NewTask("notification:lowstock", nil))
```

Gotchas: handler returning an error → automatic retry (exponential backoff) up to MaxRetry → then archived; return `asynq.SkipRetry` for permanent failures. Handlers MUST be idempotent (NFR-3). Task type string constants live in `infra/queue/tasks.go`.

## 6. zerolog (`github.com/rs/zerolog`)

```go
log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "api").Logger()
log.Info().Str("request_id", rid).Int("status", 200).Dur("latency", d).Msg("request")
log.Error().Err(err).Str("order_id", id).Msg("checkout failed")
zerolog.SetGlobalLevel(zerolog.InfoLevel)    // from config; DebugLevel in dev
```

Gotchas: don't use `Msgf` with raw user data; mask sensitive fields (NFR-6). Level parsing: `zerolog.ParseLevel(cfg.Log.Level)`.

## 7. Viper (`github.com/spf13/viper`)

```go
v := viper.New()
v.SetConfigFile("config/config.yaml")
v.SetEnvPrefix("APP")
v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
v.AutomaticEnv()                              // APP_DB_DSN overrides db.dsn
_ = v.ReadInConfig()                          // yaml optional; env is enough
var cfg Config
err := v.Unmarshal(&cfg)                      // struct tag `mapstructure:"db"`
```

Gotcha: AutomaticEnv + Unmarshal doesn't always map env vars that aren't in the yaml — register a default `v.SetDefault("db.max_conns", 20)` for every key so env binding stays consistent.

## 8. golang-migrate (`github.com/golang-migrate/migrate/v4`)

Files: `migrations/000001_create_roles.up.sql` + `.down.sql` (6-digit sequence). Run via CLI in the Makefile:

```
migrate -path migrations -database "$(DB_DSN)" up
migrate -path migrations -database "$(DB_DSN)" down 1
migrate create -ext sql -dir migrations -seq create_products
```

Programmatic (cmd/migrate) use `migrate.New("file://migrations", dsn)` → `.Up()`. Gotchas: `migrate.ErrNoChange` is not a fatal error; dirty state (failed mid-run) → fix manually + `force <version>`; DO NOT edit a migration file that has already run on any branch.

## 9. validator v10 (`github.com/go-playground/validator/v10`)

```go
validate := validator.New(validator.WithRequiredStructEnabled())
err := validate.Struct(req)
if errs, ok := err.(validator.ValidationErrors); ok {
    for _, fe := range errs { /* fe.Field(), fe.Tag(), fe.Param() → map to Indonesian message */ }
}
// Custom rule (Indonesian phone number):
validate.RegisterValidation("phone_id", func(fl validator.FieldLevel) bool {
    return regexp.MustCompile(`^(\+62|62|0)8[0-9]{7,12}$`).MatchString(fl.Field().String())
})
```

Commonly used tags: `required`, `email`, `min=8`, `max=100`, `gt=0`, `gte=1`, `oneof=sak batang m3 dus pcs lembar kg roll set`, `uuid`, `dive` (validate slice elements). One global validator instance (thread-safe), stored in pkg/validator.

## 10. golang-jwt v5 (`github.com/golang-jwt/jwt/v5`)

```go
claims := jwt.MapClaims{
    "sub": userID, "role": role, "jti": jti,
    "exp": time.Now().Add(15 * time.Minute).Unix(), "iat": time.Now().Unix(),
}
token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
signed, err := token.SignedString([]byte(secret))

parsed, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
    if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok { return nil, jwt.ErrSignatureInvalid }
    return []byte(secret), nil
})
// v5: expiry checked automatically during Parse; check errors.Is(err, jwt.ErrTokenExpired) for a specific message
claims, ok := parsed.Claims.(jwt.MapClaims)
```

v5 gotcha: error API changed from v4 — use `errors.Is` with v5 sentinels (`jwt.ErrTokenExpired`, `jwt.ErrTokenMalformed`). Refresh token is NOT a JWT (opaque random 32-byte hex in Redis).

## 11. google/uuid (`github.com/google/uuid`) — UUID v7

```go
id, err := uuid.NewV7()      // time-ordered, used for all PKs
id := uuid.Must(uuid.NewV7())
parsed, err := uuid.Parse(str)
```

Gotcha: `uuid.New()` = v4 (random) — DO NOT use for PKs (our standard is v7 for efficient B-tree indexing). DB column type `uuid`, default generated by the application (not the DB).

## 12. gRPC + protobuf (`google.golang.org/grpc`, `google.golang.org/protobuf`) — not used yet

**Not part of the current architecture.** Payment (Xendit), notification (Mailjet), and shipping (RajaOngkir) are internal adapter packages (`internal/infra/xendit`, `internal/infra/mailjet`, `internal/infra/rajaongkir`) implementing the `domain/service` interfaces, called in-process — see CLAUDE.md §5.9. Keep this section only as a reference for the day a real need to extract a standalone service shows up (separate team, independent scaling, different deploy cadence). Do not build this until then.

Generate (if/when extracted):

```
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/payment/v1/payment.proto
```

Client (infra/grpcclient):

```go
conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
client := paymentv1.NewPaymentServiceClient(conn)
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()
res, err := client.CreateInvoice(ctx, &paymentv1.CreateInvoiceRequest{...})
st, _ := status.FromError(err)               // st.Code(): codes.Unavailable → ErrPaymentUnavailable
```

Server (services/payment): implement the `paymentv1.UnimplementedPaymentServiceServer` interface, register it with `grpc.NewServer()`, `lis, _ := net.Listen("tcp", ":50051")`. Gotchas: `grpc.NewClient` (replaces the deprecated DialContext); always set a timeout per call; optional proto fields use the wrapper/optional keyword.

## 13. AWS SDK v2 S3 (`github.com/aws/aws-sdk-go-v2`) — R2/MinIO

```go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")))
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String(endpoint)     // MinIO: http://localhost:9000 | R2: https://<acct>.r2.cloudflarestorage.com
    o.UsePathStyle = true                     // REQUIRED for MinIO
    o.Region = "auto"                         // R2 uses "auto"
})
_, err = client.PutObject(ctx, &s3.PutObjectInput{Bucket: &b, Key: &k, Body: r, ContentType: &ct})
// Presigned URL (KYC document download / CSV export):
ps := s3.NewPresignClient(client)
req, err := ps.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &b, Key: &k},
    s3.WithPresignExpires(24*time.Hour))
// req.URL
```

## 14. Testing: testify, mockery, testcontainers

```go
// testify
assert.Equal(t, want, got); require.NoError(t, err)   // require stops the test, assert continues

// mockery v2 (.mockery.yaml: packages internal/domain/... all interfaces → test/mocks)
repo := mocks.NewUserRepository(t)
repo.EXPECT().FindByEmail(mock.Anything, "a@b.c").Return(nil, errs.ErrNotFound).Once()

// testcontainers (repository integration tests)
pgC, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("test"), postgres.WithUsername("test"), postgres.WithPassword("test"),
    testcontainers.WithWaitStrategy(wait.ForLog("ready to accept connections").WithOccurrence(2)))
dsn, _ := pgC.ConnectionString(ctx, "sslmode=disable")
// run migrations against dsn, then test against the real repository
```

Gotcha: regenerate mockery whenever a domain interface changes (`make mocks`); testcontainers needs the Docker socket — runs natively in CI on GitHub Actions.

---

## Policy for Updating This Document

When implementation reveals an API that differs from what's recorded here (version change), fix this document in the same commit as the code (`docs(packages): correct X API for vY.Z`). This document is a knowledge cache — its accuracy is a shared responsibility.
