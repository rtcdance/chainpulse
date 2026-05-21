package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"), "Database URL")
	migrationsPath := flag.String("path", "migrations", "Path to migration files")
	flag.Parse()

	if *dbURL == "" {
		log.Fatal("DATABASE_URL is required (set env var or use -db flag)")
	}

	args := flag.Args()
	if len(args) == 0 {
		args = []string{"up"}
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", *migrationsPath),
		*dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close() //nolint:errcheck // deferred close

	command := args[0]

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		if err == migrate.ErrNoChange {
			fmt.Println("No new migrations to apply.")
		} else {
			fmt.Println("Migrations applied successfully.")
		}

	case "down":
		if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Rollback applied successfully.")

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		fmt.Printf("Current version: %d (dirty: %v)\n", version, dirty)

	case "force":
		if len(args) < 2 {
			log.Fatal("Usage: migrate force <version>")
		}
		var version int
		if _, err := fmt.Sscanf(args[1], "%d", &version); err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		fmt.Printf("Forced version to %d.\n", version)

	default:
		log.Fatalf("Unknown command: %s. Use: up, down, version, force", command)
	}
}
