package storage

import (
	"errors"
	"sync"

	"lucys-beauty-parlour-backend/models"
)

type InMemoryStore struct {
	mu    sync.RWMutex
	appts map[int64]*models.Appointment
	next  int64
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		appts: make(map[int64]*models.Appointment),
		next:  1,
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
