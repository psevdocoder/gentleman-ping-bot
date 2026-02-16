package cron

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

type Manager struct {
	cron    *cron.Cron
	mu      sync.Mutex
	entries map[string]cron.EntryID
}

func NewCronManager() *Manager {
	return &Manager{
		cron:    cron.New(),
		mu:      sync.Mutex{},
		entries: make(map[string]cron.EntryID),
	}
}

// AddTask добавляет задачу по cron-выражению
func (m *Manager) AddTask(ctx context.Context, spec string, task Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.entries[task.Name()]; exists {
		return fmt.Errorf("task %s already exists", task.Name())
	}

	id, err := m.cron.AddFunc(spec, func() {
		if err := task.Work(ctx); err != nil {
			log.Printf("task %s failed: %v", task.Name(), err)
			return
		}

		log.Printf("task %s completed successfully", task.Name())
	})

	if err != nil {
		return err
	}

	m.entries[task.Name()] = id

	log.Printf("task %s added", task.Name())
	return nil
}

// RemoveTask удаляет задачу по имени
func (m *Manager) RemoveTask(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	id, exists := m.entries[name]
	if !exists {
		return fmt.Errorf("task %s not found", name)
	}

	m.cron.Remove(id)
	delete(m.entries, name)

	log.Printf("task %s removed", name)

	return nil
}

// Start запускает cron
func (m *Manager) Start() {
	m.cron.Start()
}

// Stop корректно останавливает cron
func (m *Manager) Stop(ctx context.Context) error {
	stopCtx := m.cron.Stop()

	select {
	case <-stopCtx.Done():
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
