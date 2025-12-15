package storage

import (
	"errors"
	"strings"
	"sync"

	"lucys-beauty-parlour-backend/models"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	appts map[int64]*models.Appointment
	next  int64
	// Services blog
	services    map[int64]*models.ServiceItem
	nextService int64
}

type RefreshStore struct {
	mu     sync.RWMutex
	tokens map[string]bool
}

func NewRefreshStore() *RefreshStore {
	return &RefreshStore{
		tokens: make(map[string]bool),
	}
}

func (s *RefreshStore) Save(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[token] = true
}

func (s *RefreshStore) Exists(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.tokens[token]
}

func (s *RefreshStore) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, token)
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		appts:       make(map[int64]*models.Appointment),
		next:        1,
		services:    make(map[int64]*models.ServiceItem),
		nextService: 1,
	}
}

func (s *InMemoryStore) CreateAppointment(a *models.Appointment) *models.Appointment {
	s.mu.Lock()
	defer s.mu.Unlock()
	a.ID = s.next
	s.next++
	s.appts[a.ID] = a
	return a
}

func (s *InMemoryStore) GetAllAppointments() []*models.Appointment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Appointment, 0, len(s.appts))
	for _, v := range s.appts {
		out = append(out, v)
	}
	return out
}

func (s *InMemoryStore) GetAppointment(id int64) (*models.Appointment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if a, ok := s.appts[id]; ok {
		return a, nil
	}
	return nil, errors.New("not found")
}

func (s *InMemoryStore) UpdateAppointment(id int64, upd *models.Appointment) (*models.Appointment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.appts[id]; !ok {
		return nil, errors.New("not found")
	}
	upd.ID = id
	s.appts[id] = upd
	return upd, nil
}

func (s *InMemoryStore) DeleteAppointment(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.appts[id]; !ok {
		return errors.New("not found")
	}
	delete(s.appts, id)
	return nil
}

func (s *InMemoryStore) CountAppointmentsByDateAndStatus(date string, status string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	for _, a := range s.appts {
		if a.Date == date && a.Status == status {
			count++
		}
	}
	return count
}

func (s *InMemoryStore) IsAppointmentSlotAvailable(date string) bool {
	confirmedCount := s.CountAppointmentsByDateAndStatus(date, "confirmed")
	return confirmedCount < 15
}

func (s *InMemoryStore) CancelAppointment(id int64) (*models.Appointment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a, ok := s.appts[id]; ok {
		a.Status = "cancelled"
		return a, nil
	}
	return nil, errors.New("not found")
}

// GetAppointmentsWithPagination returns paginated appointments with total count
func (s *InMemoryStore) GetAppointmentsWithPagination(offset, limit int) ([]*models.Appointment, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get total count
	totalCount := len(s.appts)

	// Validate pagination parameters
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10 // default limit
	}
	if limit > 100 {
		limit = 100 // max limit
	}

	// Collect all appointments
	all := make([]*models.Appointment, 0, len(s.appts))
	for _, v := range s.appts {
		all = append(all, v)
	}

	// Calculate end index
	start := offset
	end := offset + limit
	if start > len(all) {
		start = len(all)
	}
	if end > len(all) {
		end = len(all)
	}

	// Return paginated slice
	return all[start:end], totalCount
}

// --- Services (Blog) Operations ---

func (s *InMemoryStore) CreateServiceItem(it *models.ServiceItem) *models.ServiceItem {
	s.mu.Lock()
	defer s.mu.Unlock()
	it.ID = s.nextService
	s.nextService++
	s.services[it.ID] = it
	return it
}

func (s *InMemoryStore) UpdateServiceItem(id int64, upd *models.ServiceItem) (*models.ServiceItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[id]; !ok {
		return nil, errors.New("not found")
	}
	upd.ID = id
	s.services[id] = upd
	return upd, nil
}

func (s *InMemoryStore) DeleteServiceItem(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[id]; !ok {
		return errors.New("not found")
	}
	delete(s.services, id)
	return nil
}

func (s *InMemoryStore) GetServiceItem(id int64) (*models.ServiceItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if v, ok := s.services[id]; ok {
		return v, nil
	}
	return nil, errors.New("not found")
}

// ListServiceItems returns filtered + paginated items and total count
func (s *InMemoryStore) ListServiceItems(category string, minRating float64, q string, offset, limit int) ([]*models.ServiceItem, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// collect filtered
	filtered := make([]*models.ServiceItem, 0, len(s.services))
	for _, v := range s.services {
		if category != "" && v.Service != category {
			continue
		}
		if minRating > 0 && v.Rating < minRating {
			continue
		}
		if q != "" {
			if !matchesQuery(v, q) {
				continue
			}
		}
		filtered = append(filtered, v)
	}

	total := len(filtered)
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	start := offset
	end := offset + limit
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

func matchesQuery(v *models.ServiceItem, q string) bool {
	ql := strings.ToLower(q)
	if strings.Contains(strings.ToLower(v.Name), ql) {
		return true
	}
	for _, d := range v.Descriptions {
		if strings.Contains(strings.ToLower(d), ql) {
			return true
		}
	}
	return false
}
