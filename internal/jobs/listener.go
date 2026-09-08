package jobs

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type Listener struct {
	url      string
	mutex    sync.Mutex
	watchers map[int]chan string
	nextId   int
}

func NewListener(databaseURL string) *Listener {
	return &Listener{url: databaseURL, watchers: map[int]chan string{}}
}

func (l *Listener) Watch() (<-chan string, func()) {
	updates := make(chan string, 16)
	l.mutex.Lock()
	id := l.nextId
	l.nextId++
	l.watchers[id] = updates
	l.mutex.Unlock()
	return updates, func() {
		l.mutex.Lock()
		delete(l.watchers, id)
		l.mutex.Unlock()
		close(updates)
	}
}

func (l *Listener) Watchers() int {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	return len(l.watchers)
}

func (l *Listener) Start(ctx context.Context) {
	go func() {
		attempt := 0
		for ctx.Err() == nil {
			if err := l.listen(ctx); err != nil && ctx.Err() == nil {
				attempt++
				if attempt <= 3 {
					log.Printf("job announcements interrupted: %v", err)
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}

func (l *Listener) listen(ctx context.Context) error {
	connection, err := pgx.Connect(ctx, l.url)
	if err != nil {
		return err
	}
	defer connection.Close(context.Background())
	if _, err := connection.Exec(ctx, "LISTEN "+Channel); err != nil {
		return err
	}
	for {
		notification, err := connection.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		l.deliver(notification.Payload)
	}
}

func (l *Listener) deliver(payload string) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	for _, updates := range l.watchers {
		select {
		case updates <- payload:
		default:
		}
	}
}
