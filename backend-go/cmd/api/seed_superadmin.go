package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// SeedSuperAdmin ensures at least one super admin exists to bootstrap the system.
func SeedSuperAdmin(db *pgxpool.Pool) {
	ctx := context.Background()
	var count int

	// Check if any super admin exists
	err := db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role = 'SUPER_ADMIN'").Scan(&count)
	if err != nil {
		log.Fatalf("❌ Failed to check for existing SUPER_ADMIN: %v", err)
	}

	if count == 0 {
		log.Println("No SUPER_ADMIN found. Seeding default super admin...")

		// Hash the default password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("superadmin123"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("❌ Failed to hash seed password: %v", err)
		}

		// Insert the default super admin
		// Using an email as the login identifier since that's what the LoginRequest DTO expects
		email := "superadmin@google-hackathon.com"

		_, err = db.Exec(ctx, `
			INSERT INTO users (email, password_hash, name, role) 
			VALUES ($1, $2, $3, $4)
		`, email, string(hashedPassword), "superadmin", "SUPER_ADMIN")

		if err != nil {
			log.Fatalf("❌ Failed to insert seed SUPER_ADMIN: %v", err)
		}

		log.Printf("✅ Seeded default SUPER_ADMIN successfully (Email: %s | Pass: superadmin123)", email)
	} else {
		log.Println("✅ SUPER_ADMIN already exists. Skipping seed.")
	}
}
