# Package Reference — Marketplace Bahan Bangunan

Referensi lokal API & pola pemakaian setiap dependency project. **AI/developer: gunakan dokumen ini sebagai sumber kebenaran pertama sebelum mencari dokumentasi eksternal.** Jika ada API yang tidak tercakup di sini, baru cek dokumentasi resmi — lalu tambahkan temuannya ke dokumen ini.

Aturan umum: pin semua versi di go.mod; jangan upgrade major version tanpa persetujuan.

---

## 1. Fiber v3 (`github.com/gofiber/fiber/v3`) — HTTP framework

**Status: v3 sudah rilis STABIL (v3.0.0+, terbaru v3.1.x). Wajib Go 1.25+.**

Perubahan terpenting dari v2 (jangan pakai pola v2):

```go
// Handler: fiber.Ctx adalah INTERFACE, tanpa pointer
app.Get("/products/:id", func(c fiber.Ctx) error {   // BUKAN *fiber.Ctx
    return c.JSON(fiber.Map{"ok": true})
})
```

**fiber.Ctx memenuhi context.Context** — teruskan langsung ke pgx/redis/gRPC, tanpa `c.UserContext()`:

```go
rows, err := pool.Query(c, "SELECT ...")   // c langsung sebagai ctx
```

**Unified binding** (pengganti BodyParser/QueryParser/ParamsParser v2):

```go
type CreateProductReq struct {
    ID    int    `params:"id"`
    Sort  string `query:"sort"`
    Name  string `json:"name" validate:"required,min=3"`
}
var req CreateProductReq
if err := c.Bind().Body(&req); err != nil { ... }   // juga: .Query(&req), .URI(&req), .Header(&req)
```

Setup, route group, middleware:

```go
app := fiber.New(fiber.Config{AppName: "marketplace-api", BodyLimit: 2 << 20})
v1 := app.Group("/api/v1")
admin := v1.Group("/admin", mw.Auth(), mw.RequireRole("admin"))

// Middleware signature sama dengan handler:
func RequestID() fiber.Handler {
    return func(c fiber.Ctx) error {
        // c.Get(header), c.Set(header), c.Locals(key, val), c.Next()
        return c.Next()
    }
}
app.Listen(":8080", fiber.ListenConfig{DisableStartupMessage: false}) // Listen menyatu dengan config
```

Gotchas v3:
- `app.Static()` DIHAPUS → gunakan middleware `static`. `app.Mount()` dihapus → sub-app via `app.Use()`.
- Prefix middleware lebih ketat: `app.Use("/api", mw)` hanya match `/api` dan `/api/...` (tidak `/apiv1`).
- Logger middleware: field `Output` → `Stream`.
- `fiber.Ctx` TIDAK thread-safe & di-reuse antar request — jangan simpan referensi c/hasil `c.Params()` keluar dari handler; copy nilainya.
- Helper generic tersedia (mis. `fiber.Query[int](c, "page", 1)`, `fiber.Params[int](c, "id")`) menggantikan QueryInt/ParamsInt.
- GET otomatis mendaftarkan HEAD.
- Ambil claims auth dari `c.Locals("user_id")` yang di-set middleware sendiri (lihat pkg/jwt) — jangan pakai fiber contrib jwt agar kontrol penuh blacklist.

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
err := row.Scan(&u.ID, &u.Name)             // pgx.ErrNoRows jika kosong → map ke domain errs.ErrNotFound

rows, err := pool.Query(ctx, `SELECT ...`)
defer rows.Close()
users, err := pgx.CollectRows(rows, pgx.RowToStructByName[User])  // helper scan ke struct (tag db:"col")
```

Transaksi (pola wajib untuk checkout/settlement):

```go
tx, err := pool.Begin(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)                       // no-op jika sudah commit
// ... tx.Exec / tx.QueryRow ...
return tx.Commit(ctx)
```

Gotchas: placeholder `$1..$n` (bukan `?`); numeric → scan ke `pgtype.Numeric` atau int64 (kita pakai int64 rupiah); UUID scan langsung ke `uuid.UUID` (pgx v5 support native); error unik constraint: cek `pgconn.PgError` code `23505` → map ke ErrConflict.

## 3. sqlc (`github.com/sqlc-dev/sqlc`) — generator query type-safe

`sqlc.yaml` di root; query SQL di `internal/repository/postgres/queries/*.sql`; output package `sqlcgen`.

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

Anotasi query: `-- name: GetUserByEmail :one`, `:many`, `:exec`, `:execrows`. Regenerate: `make sqlc` (`sqlc generate`). Repository implementasi membungkus sqlcgen + mapping ke domain entity. Query dinamis kompleks (filter katalog) boleh hand-written pgx di luar sqlc.

## 4. go-redis v9 (`github.com/redis/go-redis/v9`)

```go
rdb := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password})
err := rdb.Ping(ctx).Err()

rdb.Set(ctx, key, val, 10*time.Minute)
val, err := rdb.Get(ctx, key).Result()       // redis.Nil jika tidak ada → JANGAN dianggap error sistem
rdb.Del(ctx, key)
ok, err := rdb.SetNX(ctx, "stock:"+id, 1, 10*time.Second).Result()  // distributed lock (FR-5.4)
rdb.Incr(ctx, key); rdb.Expire(ctx, key, ttl)                        // rate limit / brute force counter
```

Gotchas: selalu bedakan `err == redis.Nil` vs error koneksi; lock release pakai Lua compare-and-del (simpan token acak sebagai value, hapus hanya jika token cocok) agar tidak menghapus lock milik proses lain.

## 5. Asynq (`github.com/hibiken/asynq`) — background jobs (Redis-backed)

```go
// Enqueue (infra/queue/client.go)
client := asynq.NewClient(asynq.RedisClientOpt{Addr: addr})
payload, _ := json.Marshal(EmailSendPayload{...})
task := asynq.NewTask("email:send", payload)
info, err := client.Enqueue(task, asynq.MaxRetry(5), asynq.Queue("default"),
    asynq.ProcessIn(2*time.Hour))            // delay — dipakai order:expire alternatif

// Worker (cmd/worker)
srv := asynq.NewServer(asynq.RedisClientOpt{Addr: addr},
    asynq.Config{Concurrency: 10, Queues: map[string]int{"critical": 6, "default": 3, "low": 1}})
mux := asynq.NewServeMux()
mux.HandleFunc("email:send", handleEmailSend)   // func(ctx context.Context, t *asynq.Task) error
srv.Run(mux)

// Scheduler (cron) — proses sama dengan worker
sched := asynq.NewScheduler(asynq.RedisClientOpt{Addr: addr}, nil)
sched.Register("* * * * *", asynq.NewTask("order:expire", nil))       // tiap menit
sched.Register("0 8 * * *", asynq.NewTask("notification:lowstock", nil))
```

Gotchas: handler return error → retry otomatis (exponential backoff) sampai MaxRetry → masuk archived; return `asynq.SkipRetry` untuk gagal permanen. Handler WAJIB idempotent (NFR-3). Task type string konstanta di `infra/queue/tasks.go`.

## 6. zerolog (`github.com/rs/zerolog`)

```go
log := zerolog.New(os.Stdout).With().Timestamp().Str("service", "api").Logger()
log.Info().Str("request_id", rid).Int("status", 200).Dur("latency", d).Msg("request")
log.Error().Err(err).Str("order_id", id).Msg("checkout failed")
zerolog.SetGlobalLevel(zerolog.InfoLevel)    // dari config; DebugLevel di dev
```

Gotchas: jangan `Msgf` dengan data user mentah; mask field sensitif (NFR-6). Level parse: `zerolog.ParseLevel(cfg.Log.Level)`.

## 7. Viper (`github.com/spf13/viper`)

```go
v := viper.New()
v.SetConfigFile("config/config.yaml")
v.SetEnvPrefix("APP")
v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
v.AutomaticEnv()                              // APP_DB_DSN override db.dsn
_ = v.ReadInConfig()                          // yaml opsional; env cukup
var cfg Config
err := v.Unmarshal(&cfg)                      // struct tag `mapstructure:"db"`
```

Gotcha: AutomaticEnv + Unmarshal tidak selalu memetakan env yang tidak ada di yaml — daftarkan default `v.SetDefault("db.max_conns", 20)` untuk setiap key agar env binding konsisten.

## 8. golang-migrate (`github.com/golang-migrate/migrate/v4`)

File: `migrations/000001_create_roles.up.sql` + `.down.sql` (urutan 6 digit). Jalankan via CLI di Makefile:

```
migrate -path migrations -database "$(DB_DSN)" up
migrate -path migrations -database "$(DB_DSN)" down 1
migrate create -ext sql -dir migrations -seq create_products
```

Programatik (cmd/migrate) pakai `migrate.New("file://migrations", dsn)` → `.Up()`. Gotchas: `migrate.ErrNoChange` bukan error fatal; dirty state (gagal di tengah) → perbaiki manual + `force <version>`; JANGAN edit file migrasi yang sudah pernah dijalankan di branch manapun.

## 9. validator v10 (`github.com/go-playground/validator/v10`)

```go
validate := validator.New(validator.WithRequiredStructEnabled())
err := validate.Struct(req)
if errs, ok := err.(validator.ValidationErrors); ok {
    for _, fe := range errs { /* fe.Field(), fe.Tag(), fe.Param() → map ke pesan Indonesia */ }
}
// Custom rule (phone Indonesia):
validate.RegisterValidation("phone_id", func(fl validator.FieldLevel) bool {
    return regexp.MustCompile(`^(\+62|62|0)8[0-9]{7,12}$`).MatchString(fl.Field().String())
})
```

Tag yang sering dipakai: `required`, `email`, `min=8`, `max=100`, `gt=0`, `gte=1`, `oneof=sak batang m3 dus pcs lembar kg roll set`, `uuid`, `dive` (validasi slice element). Satu instance validator global (thread-safe), simpan di pkg/validator.

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
// v5: expiry dicek otomatis saat Parse; cek errors.Is(err, jwt.ErrTokenExpired) untuk pesan spesifik
claims, ok := parsed.Claims.(jwt.MapClaims)
```

Gotcha v5: API error berubah dari v4 — gunakan `errors.Is` dengan sentinel v5 (`jwt.ErrTokenExpired`, `jwt.ErrTokenMalformed`). Refresh token BUKAN JWT (opaque random 32 byte hex di Redis).

## 11. google/uuid (`github.com/google/uuid`) — UUID v7

```go
id, err := uuid.NewV7()      // time-ordered, dipakai semua PK
id := uuid.Must(uuid.NewV7())
parsed, err := uuid.Parse(str)
```

Gotcha: `uuid.New()` = v4 (random) — JANGAN dipakai untuk PK (kita standar v7 agar index B-tree efisien). DB kolom `uuid`, default dari aplikasi (bukan DB).

## 12. gRPC + protobuf (`google.golang.org/grpc`, `google.golang.org/protobuf`)

Generate (Makefile gen-proto):

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

Server (services/payment): implement interface `paymentv1.UnimplementedPaymentServiceServer`, daftarkan ke `grpc.NewServer()`, `lis, _ := net.Listen("tcp", ":50051")`. Gotchas: `grpc.NewClient` (pengganti DialContext yang deprecated); selalu timeout per call; proto field opsional pakai wrapper/optional keyword.

## 13. AWS SDK v2 S3 (`github.com/aws/aws-sdk-go-v2`) — R2/MinIO

```go
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(key, secret, "")))
client := s3.NewFromConfig(cfg, func(o *s3.Options) {
    o.BaseEndpoint = aws.String(endpoint)     // MinIO: http://localhost:9000 | R2: https://<acct>.r2.cloudflarestorage.com
    o.UsePathStyle = true                     // WAJIB untuk MinIO
    o.Region = "auto"                         // R2 pakai "auto"
})
_, err = client.PutObject(ctx, &s3.PutObjectInput{Bucket: &b, Key: &k, Body: r, ContentType: &ct})
// Presigned URL (download dokumen KYC / export CSV):
ps := s3.NewPresignClient(client)
req, err := ps.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &b, Key: &k},
    s3.WithPresignExpires(24*time.Hour))
// req.URL
```

## 14. Testing: testify, mockery, testcontainers

```go
// testify
assert.Equal(t, want, got); require.NoError(t, err)   // require menghentikan test, assert lanjut

// mockery v2 (.mockery.yaml: packages internal/domain/... all interfaces → test/mocks)
repo := mocks.NewUserRepository(t)
repo.EXPECT().FindByEmail(mock.Anything, "a@b.c").Return(nil, errs.ErrNotFound).Once()

// testcontainers (integration test repository)
pgC, err := postgres.Run(ctx, "postgres:16-alpine",
    postgres.WithDatabase("test"), postgres.WithUsername("test"), postgres.WithPassword("test"),
    testcontainers.WithWaitStrategy(wait.ForLog("ready to accept connections").WithOccurrence(2)))
dsn, _ := pgC.ConnectionString(ctx, "sslmode=disable")
// jalankan migrasi ke dsn, lalu test repository sungguhan
```

Gotcha: mockery generate ulang setiap interface domain berubah (`make mocks`); testcontainers butuh Docker socket — di CI GitHub Actions berjalan native.

---

## Kebijakan Update Dokumen Ini

Saat implementasi menemukan API yang berbeda dari tercatat di sini (versi berubah), perbaiki dokumen ini dalam commit yang sama dengan kode (`docs(packages): correct X API for vY.Z`). Dokumen ini adalah cache pengetahuan — keakuratannya tanggung jawab bersama.
