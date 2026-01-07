package mailer

import (
	"context"
	"sync"
)

type Mock struct {
	mu   sync.Mutex
	Sent []Email
	Err  error
}

func (m *Mock) Send(ctx context.Context, e Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Sent = append(m.Sent, e)
	return m.Err
}
