package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type seedService struct {
	ID           int64
	Service      string
	Name         string
	Descriptions []string
	Rating       float64
}

var defaultServices = []seedService{
	{ID: 1, Service: "hair", Name: "Knotless Braids", Descriptions: []string{"Small", "Medium", "Large"}, Rating: 0},
	{ID: 2, Service: "hair", Name: "Wig Install", Descriptions: []string{"Closure", "Frontal"}, Rating: 0},
	{ID: 3, Service: "makeup", Name: "Soft Glam", Descriptions: []string{"Day", "Evening"}, Rating: 0},
	{ID: 4, Service: "makeup", Name: "Bridal Makeup", Descriptions: []string{"Bride", "Bridesmaid"}, Rating: 0},
	{ID: 5, Service: "nails", Name: "Gel Manicure", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 6, Service: "nails", Name: "Acrylic Full Set", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 7, Service: "hair", Name: "Senegalese Twists", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 8, Service: "hair", Name: "Soft Locs", Descriptions: []string{"Shoulder Length", "Mid-back", "Waist Length"}, Rating: 0},
	{ID: 9, Service: "hair", Name: "Butterfly Locs", Descriptions: []string{"Shoulder Length", "Mid-back", "Waist Length"}, Rating: 0},
	{ID: 10, Service: "hair", Name: "French Curls", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 11, Service: "hair", Name: "Cornrows (All Back)", Descriptions: []string{"4 Lines", "6 Lines", "8+ Lines"}, Rating: 0},
	{ID: 12, Service: "hair", Name: "Stitch Cornrows", Descriptions: []string{"4 Lines", "6 Lines", "8+ Lines"}, Rating: 0},
	{ID: 13, Service: "hair", Name: "Fulani Cornrows", Descriptions: []string{"Classic", "With Beads"}, Rating: 0},
	{ID: 14, Service: "hair", Name: "Passion Twists", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 15, Service: "hair", Name: "Kinky Twists", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 16, Service: "hair", Name: "Hermaid Braids", Descriptions: []string{"Small", "Medium", "Large"}, Rating: 0},
	{ID: 17, Service: "hair", Name: "Italy Curls", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 18, Service: "hair", Name: "Jayda Wayda", Descriptions: []string{"Short", "Medium", "Long"}, Rating: 0},
	{ID: 19, Service: "hair", Name: "Gypsy Locs", Descriptions: []string{"Shoulder Length", "Mid-back", "Waist Length"}, Rating: 0},
	{ID: 20, Service: "hair", Name: "Sew-ins", Descriptions: []string{"Leave-out", "Closure", "Frontal"}, Rating: 0},
	{ID: 21, Service: "hair", Name: "Fulani Passion Twists", Descriptions: []string{"Short", "Midback", "Long", "Reversed", "Bouncy"}, Rating: 0},
	{ID: 22, Service: "makeup", Name: "Eyebrow Trimming", Descriptions: []string{}, Rating: 0},
}

func Migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS admins (
			id BIGSERIAL PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS service_items (
			id BIGINT PRIMARY KEY,
			service TEXT NOT NULL,
			name TEXT NOT NULL,
			descriptions JSONB NOT NULL DEFAULT '[]'::jsonb,
			rating DOUBLE PRECISION NOT NULL DEFAULT 0
		);`,
		`CREATE TABLE IF NOT EXISTS portfolio_items (
			id BIGSERIAL PRIMARY KEY,
			category TEXT NOT NULL,
			style TEXT NOT NULL,
			images JSONB NOT NULL DEFAULT '[]'::jsonb,
			description TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`CREATE TABLE IF NOT EXISTS menu_items (
			id BIGSERIAL PRIMARY KEY,
			category TEXT NOT NULL,
			name TEXT NOT NULL,
			currency TEXT,
			price_cents BIGINT NOT NULL,
			duration_minutes INT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS appointments (
			id BIGSERIAL PRIMARY KEY,
			customer_name TEXT NOT NULL,
			customer_email TEXT NOT NULL,
			customer_phone TEXT NOT NULL,
			staff_name TEXT,
			appointment_date DATE NOT NULL,
			appointment_time TIME NOT NULL,
			service_id BIGINT NOT NULL REFERENCES service_items(id) ON DELETE RESTRICT,
			service_description TEXT NOT NULL,
			currency TEXT,
			price_cents BIGINT NOT NULL DEFAULT 0,
			notes TEXT,
			status TEXT NOT NULL DEFAULT 'pending'
		);`,
		`CREATE INDEX IF NOT EXISTS idx_appointments_date_status ON appointments(appointment_date, status);`,
		`CREATE INDEX IF NOT EXISTS idx_service_items_service ON service_items(service);`,
		`CREATE INDEX IF NOT EXISTS idx_portfolio_items_category ON portfolio_items(category);`,
		`CREATE INDEX IF NOT EXISTS idx_menu_items_category ON menu_items(category);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}

	// Migrate portfolio_items schema if it exists with old structure
	// Check if 'title' column exists and rename it to 'style'
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'portfolio_items' 
			AND column_name = 'title'
		);
	`).Scan(&columnExists)

	if err == nil && columnExists {
		// Rename title to style
		if _, err := db.Exec(`ALTER TABLE portfolio_items RENAME COLUMN title TO style;`); err != nil {
			return err
		}
		// Make description NOT NULL if it isn't already
		if _, err := db.Exec(`ALTER TABLE portfolio_items ALTER COLUMN description SET NOT NULL;`); err != nil {
			// Ignore error if it's already NOT NULL
		}
	}

	return nil
}

func Seed(db *sql.DB) error {
	if err := seedAdmin(db); err != nil {
		return err
	}
	if err := seedServices(db); err != nil {
		return err
	}
	return nil
}

func seedAdmin(db *sql.DB) error {
	email := strings.TrimSpace(os.Getenv("ADMIN_EMAIL"))
	password := os.Getenv("ADMIN_PASSWORD")
	if email == "" || password == "" {
		return fmt.Errorf("ADMIN_EMAIL and ADMIN_PASSWORD are required for admin seeding")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		INSERT INTO admins (email, password_hash)
		VALUES ($1, $2)
		ON CONFLICT (email)
		DO UPDATE SET password_hash = EXCLUDED.password_hash, updated_at = NOW();
	`, email, string(hash))
	return err
}

func seedServices(db *sql.DB) error {
	for _, s := range defaultServices {
		descriptionsJSON, err := json.Marshal(s.Descriptions)
		if err != nil {
			return err
		}

		_, err = db.Exec(`
			INSERT INTO service_items (id, service, name, descriptions, rating)
			VALUES ($1, $2, $3, $4::jsonb, $5)
			ON CONFLICT (id)
			DO UPDATE SET
				service = EXCLUDED.service,
				name = EXCLUDED.name,
				descriptions = EXCLUDED.descriptions,
				rating = EXCLUDED.rating;
		`, s.ID, s.Service, s.Name, string(descriptionsJSON), s.Rating)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidateAdminCredentials(db *sql.DB, email, password string) (bool, error) {
	var hash string
	err := db.QueryRow(`SELECT password_hash FROM admins WHERE email = $1`, strings.TrimSpace(email)).Scan(&hash)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false, nil
	}
	return true, nil
}

func AdminExists(db *sql.DB, email string) (bool, error) {
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM admins WHERE email = $1)`, strings.TrimSpace(email)).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

func UpdateAdminPassword(db *sql.DB, email, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := db.Exec(`UPDATE admins SET password_hash = $1, updated_at = NOW() WHERE email = $2`, string(hash), strings.TrimSpace(email))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}
