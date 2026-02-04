package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	_ "github.com/lib/pq"

	"github.com/issuesight/issuesight/internal/concepts"
	"github.com/issuesight/issuesight/internal/config"
	"github.com/issuesight/issuesight/internal/platform/db/ent"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load failed: %v", err)
	}

	db, err := ent.Open("postgres", cfg.PostgresURL)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := db.Schema.Create(ctx); err != nil {
		log.Fatalf("db schema migrate failed: %v", err)
	}

	if err := concepts.AutoSeed(ctx, db, slog.Default(), concepts.SeedOptions{
		ForceBackfill: cfg.ConceptsForceBackfill,
	}); err != nil {
		log.Fatalf("concept seed failed: %v", err)
	}

	log.Println("concept seed complete")
}
