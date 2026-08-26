<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="assets/logo-dark-mode.png">
    <source media="(prefers-color-scheme: light)" srcset="assets/logo-light-mode.png">
    <img alt="kinetic" src="assets/logo-light-mode.png" width="480">
  </picture>
</p>

<p align="center">
  <a href="https://github.com/Trxncoo/kinetic/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/Trxncoo/kinetic/actions/workflows/ci.yml/badge.svg"></a>
  <a href="https://pkg.go.dev/github.com/Trxncoo/kinetic"><img alt="Go Reference" src="https://pkg.go.dev/badge/github.com/Trxncoo/kinetic.svg"></a>
  <a href="go.mod"><img alt="Go Version" src="https://img.shields.io/github/go-mod/go-version/Trxncoo/kinetic"></a>
  <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
</p>

<p align="center">
  <strong>Small, composable Go packages — built for performance and DX.</strong>
</p>

---

## Packages

- [`kevent`](pkg/kevent) — a generic in-memory event bus (`Bus[T]`,
  `Registry`).
- [`kcache`](pkg/kcache) — a generic sharded in-memory cache (`Cache[K,
  V]`, `Registry`).
- [`kcore`](pkg/kcore) — the generic concurrency-safe registry
  (`Registry[Key]`) that `kevent.Registry` and `kcache.Registry` are both
  built on.
- [`kretry`](pkg/kretry) — retry with composable backoff and jitter
  (`Backoff`, `Do`/`DoValue`).

## Install

```sh
go get github.com/Trxncoo/kinetic
```

## Showcase

### kevent

#### Bus

```go
orders := kevent.NewBus[OrderPlaced]()

unsubscribe := orders.Subscribe(func(ctx context.Context, e OrderPlaced) error {
	fmt.Println("emailing receipt for", e.ID)
	return nil
})

orders.Publish(ctx, OrderPlaced{ID: "o1", Total: 42})
unsubscribe()
```

#### Registry

```go
reg := kevent.NewRegistry()

// Wired once at startup, per event type.
kevent.Register(reg, kevent.NewBus[OrderPlaced]())
kevent.Register(reg, kevent.NewBus[UserSignedUp]())

// Elsewhere, code that only knows the event type gets the right bus.
orders := kevent.From[OrderPlaced](reg)
orders.Subscribe(func(ctx context.Context, e OrderPlaced) error {
	fmt.Println("emailing receipt for", e.ID)
	return nil
})
orders.Publish(ctx, OrderPlaced{ID: "o4", Total: 7})
```

See [`pkg/kevent`](pkg/kevent) for the full package docs and runnable
examples.

### kcache

#### Cache

```go
profiles := kcache.NewCache[string, UserProfileDTO]()

profiles.Set("user-1", UserProfileDTO{Name: "Alice", Plan: "pro"}, 5*time.Minute)

if p, ok := profiles.Get("user-1"); ok {
	fmt.Printf("%s is on the %s plan\n", p.Name, p.Plan)
}
```

#### Registry

```go
reg := kcache.NewRegistry()

// Wired once at startup, per named cache.
kcache.Register(reg, "sessions", kcache.NewCache[string, string]())
kcache.Register(reg, "profiles", kcache.NewCache[string, UserProfileDTO]())

// Elsewhere, code that only knows the name gets the right cache.
sessions := kcache.From[string, string](reg, "sessions")
sessions.Set("session-1", "alice", time.Minute)
```

See [`pkg/kcache`](pkg/kcache) for the full package docs and runnable
examples.

### kcore

```go
r := kcore.NewRegistry[string]()

kcore.Register(r, "greeting", "hello")

if v, err := kcore.From[string, string](r, "greeting"); err == nil {
	fmt.Println(v)
}
```

See [`pkg/kcore`](pkg/kcore) for the full package docs and a
runnable example.

### kretry

```go
backoff := kretry.NewExponential(200 * time.Millisecond).
	WithMaxRetries(5).
	WithCappedDuration(30 * time.Second).
	WithFullJitter()

body, err := kretry.DoValue(ctx, backoff, func(ctx context.Context) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, kretry.RetryableError(err) // network error: worth retrying
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, kretry.RetryableError(fmt.Errorf("server error: %d", resp.StatusCode))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("client error: %d", resp.StatusCode) // permanent, stops immediately
	}

	return io.ReadAll(resp.Body)
})
```

See [`pkg/kretry`](pkg/kretry) for the full package docs and runnable
examples.

## License

[MIT](LICENSE)
