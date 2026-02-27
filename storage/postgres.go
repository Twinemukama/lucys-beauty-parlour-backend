package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"lucys-beauty-parlour-backend/models"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// normalizeImagePaths ensures all image paths have a leading slash for URL compatibility
func normalizeImagePaths(paths []string) []string {
	for i, p := range paths {
		if p != "" && !strings.HasPrefix(p, "/") {
			paths[i] = "/" + p
		}
	}
	return paths
}

func (s *PostgresStore) CreateAppointment(a *models.Appointment) *models.Appointment {
	const q = `
		INSERT INTO appointments (
			customer_name, customer_email, customer_phone, staff_name,
			appointment_date, appointment_time, service_id, service_description,
			currency, price_cents, notes, status
		)
		VALUES ($1,$2,$3,$4,$5::date,$6::time,$7,$8,$9,$10,$11,$12)
		RETURNING id;
	`
	if err := s.db.QueryRow(q,
		a.CustomerName,
		a.CustomerEmail,
		a.CustomerPhone,
		a.StaffName,
		a.Date,
		a.Time,
		a.ServiceID,
		a.ServiceDescription,
		a.Currency,
		a.PriceCents,
		a.Notes,
		a.Status,
	).Scan(&a.ID); err != nil {
		return nil
	}
	return a
}

func (s *PostgresStore) GetAllAppointments() []*models.Appointment {
	rows, err := s.db.Query(`
		SELECT id, customer_name, customer_email, customer_phone, staff_name,
			TO_CHAR(appointment_date, 'YYYY-MM-DD') AS date_str,
			TO_CHAR(appointment_time, 'HH24:MI') AS time_str,
			service_id, service_description, currency, price_cents, notes, status
		FROM appointments
		ORDER BY id DESC
	`)
	if err != nil {
		return []*models.Appointment{}
	}
	defer rows.Close()

	out := make([]*models.Appointment, 0)
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			continue
		}
		out = append(out, a)
	}
	return out
}

func (s *PostgresStore) GetAppointment(id int64) (*models.Appointment, error) {
	const q = `
		SELECT id, customer_name, customer_email, customer_phone, staff_name,
			TO_CHAR(appointment_date, 'YYYY-MM-DD') AS date_str,
			TO_CHAR(appointment_time, 'HH24:MI') AS time_str,
			service_id, service_description, currency, price_cents, notes, status
		FROM appointments
		WHERE id = $1
	`
	row := s.db.QueryRow(q, id)
	a := &models.Appointment{}
	if err := row.Scan(
		&a.ID,
		&a.CustomerName,
		&a.CustomerEmail,
		&a.CustomerPhone,
		&a.StaffName,
		&a.Date,
		&a.Time,
		&a.ServiceID,
		&a.ServiceDescription,
		&a.Currency,
		&a.PriceCents,
		&a.Notes,
		&a.Status,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("not found")
		}
		return nil, err
	}
	return a, nil
}

func (s *PostgresStore) UpdateAppointment(id int64, upd *models.Appointment) (*models.Appointment, error) {
	const q = `
		UPDATE appointments SET
			customer_name = $1,
			customer_email = $2,
			customer_phone = $3,
			staff_name = $4,
			appointment_date = $5::date,
			appointment_time = $6::time,
			service_id = $7,
			service_description = $8,
			currency = $9,
			price_cents = $10,
			notes = $11,
			status = $12
		WHERE id = $13
	`
	res, err := s.db.Exec(q,
		upd.CustomerName,
		upd.CustomerEmail,
		upd.CustomerPhone,
		upd.StaffName,
		upd.Date,
		upd.Time,
		upd.ServiceID,
		upd.ServiceDescription,
		upd.Currency,
		upd.PriceCents,
		upd.Notes,
		upd.Status,
		id,
	)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("not found")
	}
	upd.ID = id
	return upd, nil
}

func (s *PostgresStore) DeleteAppointment(id int64) error {
	res, err := s.db.Exec(`DELETE FROM appointments WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *PostgresStore) IsAppointmentSlotAvailable(date string) bool {
	var cnt int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM appointments WHERE appointment_date = $1::date AND status = 'confirmed'`, date).Scan(&cnt)
	if err != nil {
		return false
	}
	return cnt < 15
}

func (s *PostgresStore) CancelAppointment(id int64) (*models.Appointment, error) {
	res, err := s.db.Exec(`UPDATE appointments SET status = 'cancelled' WHERE id = $1`, id)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("not found")
	}
	return s.GetAppointment(id)
}

func (s *PostgresStore) GetAppointmentsWithPagination(offset, limit int) ([]*models.Appointment, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	var total int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM appointments`).Scan(&total); err != nil {
		return []*models.Appointment{}, 0
	}

	rows, err := s.db.Query(`
		SELECT id, customer_name, customer_email, customer_phone, staff_name,
			TO_CHAR(appointment_date, 'YYYY-MM-DD') AS date_str,
			TO_CHAR(appointment_time, 'HH24:MI') AS time_str,
			service_id, service_description, currency, price_cents, notes, status
		FROM appointments
		ORDER BY id DESC
		OFFSET $1 LIMIT $2
	`, offset, limit)
	if err != nil {
		return []*models.Appointment{}, total
	}
	defer rows.Close()

	out := make([]*models.Appointment, 0)
	for rows.Next() {
		a, err := scanAppointment(rows)
		if err != nil {
			continue
		}
		out = append(out, a)
	}
	return out, total
}

func (s *PostgresStore) CreateServiceItem(it *models.ServiceItem) *models.ServiceItem {
	descJSON, _ := json.Marshal(it.Descriptions)

	if it.ID > 0 {
		err := s.db.QueryRow(`
			INSERT INTO service_items (id, service, name, descriptions, rating)
			VALUES ($1, $2, $3, $4::jsonb, $5)
			ON CONFLICT (id) DO UPDATE SET
				service = EXCLUDED.service,
				name = EXCLUDED.name,
				descriptions = EXCLUDED.descriptions,
				rating = EXCLUDED.rating
			RETURNING id
		`, it.ID, it.Service, it.Name, string(descJSON), it.Rating).Scan(&it.ID)
		if err != nil {
			return nil
		}
		return it
	}

	err := s.db.QueryRow(`
		WITH next_id AS (
			SELECT COALESCE(MAX(id), 0) + 1 AS id FROM service_items
		)
		INSERT INTO service_items (id, service, name, descriptions, rating)
		SELECT id, $1, $2, $3::jsonb, $4 FROM next_id
		RETURNING id
	`, it.Service, it.Name, string(descJSON), it.Rating).Scan(&it.ID)
	if err != nil {
		return nil
	}
	return it
}

func (s *PostgresStore) UpdateServiceItem(id int64, upd *models.ServiceItem) (*models.ServiceItem, error) {
	descJSON, err := json.Marshal(upd.Descriptions)
	if err != nil {
		return nil, err
	}

	res, err := s.db.Exec(`
		UPDATE service_items
		SET service = $1, name = $2, descriptions = $3::jsonb, rating = $4
		WHERE id = $5
	`, upd.Service, upd.Name, string(descJSON), upd.Rating, id)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("not found")
	}
	upd.ID = id
	return upd, nil
}

func (s *PostgresStore) DeleteServiceItem(id int64) error {
	res, err := s.db.Exec(`DELETE FROM service_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *PostgresStore) GetServiceItem(id int64) (*models.ServiceItem, error) {
	var descriptionsRaw []byte
	it := &models.ServiceItem{}
	err := s.db.QueryRow(`
		SELECT id, service, name, descriptions, rating
		FROM service_items
		WHERE id = $1
	`, id).Scan(&it.ID, &it.Service, &it.Name, &descriptionsRaw, &it.Rating)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(descriptionsRaw, &it.Descriptions); err != nil {
		it.Descriptions = []string{}
	}
	return it, nil
}

func (s *PostgresStore) ListServiceItems(category string, minRating float64, q string, offset, limit int) ([]*models.ServiceItem, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	where := []string{"1=1"}
	args := make([]any, 0)
	argN := 1

	if category != "" {
		where = append(where, fmt.Sprintf("service = $%d", argN))
		args = append(args, category)
		argN++
	}
	if minRating > 0 {
		where = append(where, fmt.Sprintf("rating >= $%d", argN))
		args = append(args, minRating)
		argN++
	}
	if strings.TrimSpace(q) != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(q)) + "%"
		where = append(where, fmt.Sprintf("(LOWER(name) LIKE $%d OR LOWER(descriptions::text) LIKE $%d)", argN, argN))
		args = append(args, pattern)
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM service_items WHERE %s", whereSQL)
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return []*models.ServiceItem{}, 0
	}

	listArgs := append(args, offset, limit)
	listQuery := fmt.Sprintf(`
		SELECT id, service, name, descriptions, rating
		FROM service_items
		WHERE %s
		ORDER BY id ASC
		OFFSET $%d LIMIT $%d
	`, whereSQL, argN, argN+1)

	rows, err := s.db.Query(listQuery, listArgs...)
	if err != nil {
		return []*models.ServiceItem{}, total
	}
	defer rows.Close()

	out := make([]*models.ServiceItem, 0)
	for rows.Next() {
		var descRaw []byte
		it := &models.ServiceItem{}
		if err := rows.Scan(&it.ID, &it.Service, &it.Name, &descRaw, &it.Rating); err != nil {
			continue
		}
		if err := json.Unmarshal(descRaw, &it.Descriptions); err != nil {
			it.Descriptions = []string{}
		}
		out = append(out, it)
	}
	return out, total
}

func (s *PostgresStore) CreateMenuItem(it *models.MenuItem) *models.MenuItem {
	err := s.db.QueryRow(`
		INSERT INTO menu_items (category, name, currency, price_cents, duration_minutes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, it.Category, it.Name, it.Currency, it.PriceCents, it.DurationMinutes).Scan(&it.ID)
	if err != nil {
		return nil
	}
	return it
}

func (s *PostgresStore) GetMenuItem(id int64) (*models.MenuItem, error) {
	it := &models.MenuItem{}
	err := s.db.QueryRow(`
		SELECT id, category, name, currency, price_cents, duration_minutes
		FROM menu_items
		WHERE id = $1
	`, id).Scan(&it.ID, &it.Category, &it.Name, &it.Currency, &it.PriceCents, &it.DurationMinutes)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	return it, nil
}

func (s *PostgresStore) UpdateMenuItem(id int64, upd *models.MenuItem) (*models.MenuItem, error) {
	res, err := s.db.Exec(`
		UPDATE menu_items
		SET category = $1, name = $2, currency = $3, price_cents = $4, duration_minutes = $5
		WHERE id = $6
	`, upd.Category, upd.Name, upd.Currency, upd.PriceCents, upd.DurationMinutes, id)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, errors.New("not found")
	}
	upd.ID = id
	return upd, nil
}

func (s *PostgresStore) DeleteMenuItem(id int64) error {
	res, err := s.db.Exec(`DELETE FROM menu_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *PostgresStore) ListMenuItems(category string, q string, offset, limit int) ([]*models.MenuItem, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	where := []string{"1=1"}
	args := make([]any, 0)
	argN := 1

	if strings.TrimSpace(category) != "" {
		where = append(where, fmt.Sprintf("category = $%d", argN))
		args = append(args, strings.TrimSpace(category))
		argN++
	}
	if strings.TrimSpace(q) != "" {
		where = append(where, fmt.Sprintf("LOWER(name) LIKE $%d", argN))
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(q))+"%")
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	var total int
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM menu_items WHERE %s", whereSQL)
	if err := s.db.QueryRow(countQ, args...).Scan(&total); err != nil {
		return []*models.MenuItem{}, 0
	}

	listArgs := append(args, offset, limit)
	listQ := fmt.Sprintf(`
		SELECT id, category, name, currency, price_cents, duration_minutes
		FROM menu_items
		WHERE %s
		ORDER BY id ASC
		OFFSET $%d LIMIT $%d
	`, whereSQL, argN, argN+1)

	rows, err := s.db.Query(listQ, listArgs...)
	if err != nil {
		return []*models.MenuItem{}, total
	}
	defer rows.Close()

	items := make([]*models.MenuItem, 0)
	for rows.Next() {
		it := &models.MenuItem{}
		if err := rows.Scan(&it.ID, &it.Category, &it.Name, &it.Currency, &it.PriceCents, &it.DurationMinutes); err != nil {
			continue
		}
		items = append(items, it)
	}
	return items, total
}

// Portfolio Item Methods
func (s *PostgresStore) CreatePortfolioItem(it *models.PortfolioItem) *models.PortfolioItem {
	imgJSON, _ := json.Marshal(it.Images)

	err := s.db.QueryRow(`
		INSERT INTO portfolio_items (category, style, images, description)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, it.Category, it.Style, string(imgJSON), it.Description).Scan(&it.ID, &it.CreatedAt)
	if err != nil {
		// Log the error for debugging
		fmt.Printf("CreatePortfolioItem error: %v\n", err)
		return nil
	}
	return it
}

func (s *PostgresStore) UpdatePortfolioItem(id int64, upd *models.PortfolioItem) (*models.PortfolioItem, error) {
	imgJSON, err := json.Marshal(upd.Images)
	if err != nil {
		return nil, err
	}

	var createdAt string
	err = s.db.QueryRow(`
		UPDATE portfolio_items
		SET category = $1, style = $2, images = $3::jsonb, description = $4
		WHERE id = $5
		RETURNING TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
	`, upd.Category, upd.Style, string(imgJSON), upd.Description, id).Scan(&createdAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	upd.ID = id
	upd.CreatedAt = createdAt
	return upd, nil
}

func (s *PostgresStore) DeletePortfolioItem(id int64) error {
	res, err := s.db.Exec(`DELETE FROM portfolio_items WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("not found")
	}
	return nil
}

func (s *PostgresStore) GetPortfolioItem(id int64) (*models.PortfolioItem, error) {
	var imagesRaw []byte
	it := &models.PortfolioItem{}
	err := s.db.QueryRow(`
		SELECT id, category, style, images, description, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM portfolio_items
		WHERE id = $1
	`, id).Scan(&it.ID, &it.Category, &it.Style, &imagesRaw, &it.Description, &it.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, errors.New("not found")
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(imagesRaw, &it.Images); err != nil {
		it.Images = []string{}
	}
	it.Images = normalizeImagePaths(it.Images)
	return it, nil
}

func (s *PostgresStore) ListPortfolioItems(category string, q string, offset, limit int) ([]*models.PortfolioItem, int) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	where := []string{"1=1"}
	args := make([]any, 0)
	argN := 1

	if category != "" {
		where = append(where, fmt.Sprintf("category = $%d", argN))
		args = append(args, category)
		argN++
	}
	if strings.TrimSpace(q) != "" {
		pattern := "%" + strings.ToLower(strings.TrimSpace(q)) + "%"
		where = append(where, fmt.Sprintf("(LOWER(style) LIKE $%d OR LOWER(description) LIKE $%d)", argN, argN))
		args = append(args, pattern)
		argN++
	}

	whereSQL := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM portfolio_items WHERE %s", whereSQL)
	var total int
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return []*models.PortfolioItem{}, 0
	}

	listArgs := append(args, offset, limit)
	listQuery := fmt.Sprintf(`
		SELECT id, category, style, images, description, TO_CHAR(created_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM portfolio_items
		WHERE %s
		ORDER BY created_at DESC
		OFFSET $%d LIMIT $%d
	`, whereSQL, argN, argN+1)

	rows, err := s.db.Query(listQuery, listArgs...)
	if err != nil {
		return []*models.PortfolioItem{}, total
	}
	defer rows.Close()

	out := make([]*models.PortfolioItem, 0)
	for rows.Next() {
		var imgRaw []byte
		it := &models.PortfolioItem{}
		if err := rows.Scan(&it.ID, &it.Category, &it.Style, &imgRaw, &it.Description, &it.CreatedAt); err != nil {
			continue
		}
		if err := json.Unmarshal(imgRaw, &it.Images); err != nil {
			it.Images = []string{}
		}
		it.Images = normalizeImagePaths(it.Images)
		out = append(out, it)
	}
	return out, total
}

func scanAppointment(scanner interface {
	Scan(dest ...any) error
}) (*models.Appointment, error) {
	a := &models.Appointment{}
	if err := scanner.Scan(
		&a.ID,
		&a.CustomerName,
		&a.CustomerEmail,
		&a.CustomerPhone,
		&a.StaffName,
		&a.Date,
		&a.Time,
		&a.ServiceID,
		&a.ServiceDescription,
		&a.Currency,
		&a.PriceCents,
		&a.Notes,
		&a.Status,
	); err != nil {
		return nil, err
	}
	return a, nil
}
