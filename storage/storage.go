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
