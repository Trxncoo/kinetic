package kcache_test

import (
	"fmt"
	"time"

	"github.com/Trxncoo/kinetic/pkg/kcache"
)

func ExampleCache() {
	sessions := kcache.NewCache[string, string]()

	sessions.Set("session-1", "alice", time.Minute)

	if user, ok := sessions.Get("session-1"); ok {
		fmt.Println("logged in as", user)
	}

	sessions.Delete("session-1")

	if _, ok := sessions.Get("session-1"); !ok {
		fmt.Println("session-1 is gone")
	}

	// Output:
	// logged in as alice
	// session-1 is gone
}

type userProfileDTO struct {
	Name string
	Plan string
}

func ExampleCache_dto() {
	profiles := kcache.NewCache[string, userProfileDTO]()

	profiles.Set("user-1", userProfileDTO{Name: "Alice", Plan: "pro"}, 5*time.Minute)

	if p, ok := profiles.Get("user-1"); ok {
		fmt.Printf("%s is on the %s plan\n", p.Name, p.Plan)
	}

	// Output:
	// Alice is on the pro plan
}

func ExampleRegistry() {
	reg := kcache.NewRegistry()

	// Wired once at startup, per named cache.
	kcache.Register(reg, "sessions", kcache.NewCache[string, string]())
	kcache.Register(reg, "profiles", kcache.NewCache[string, userProfileDTO]())

	// Elsewhere, code that only knows the name gets the right cache.
	sessions := kcache.From[string, string](reg, "sessions")
	sessions.Set("session-1", "alice", time.Minute)

	if user, ok := sessions.Get("session-1"); ok {
		fmt.Println("logged in as", user)
	}

	// Output:
	// logged in as alice
}
