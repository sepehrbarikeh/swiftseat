package worker

import (
	"log"
	"time"

	"swift-seat/internal/repository"
)

type CleanupWorker struct {
	repo     *repository.PostgresDB
	interval time.Duration
	stopChan chan struct{}
}

func NewCleanupWorker(repo *repository.PostgresDB, interval time.Duration) *CleanupWorker {
	return &CleanupWorker{
		repo:     repo,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (w *CleanupWorker) Start() {
	ticker := time.NewTicker(w.interval)

	go func() {
		log.Printf("[Worker] Cleanup worker started with interval %v", w.interval)
		for {
			select {
			case <-ticker.C:
				rowsAffected, err := w.repo.CleanupExpiredSeats()
				if err != nil {
					log.Printf("[Worker Error] cleanup error %v", err)
				} else if rowsAffected > 0 {
					log.Printf("[Worker] %d expired seats cleaned up.", rowsAffected)
				}
			case <-w.stopChan:
				ticker.Stop()
				log.Println("[Worker] Cleanup worker stopped.")
				return
			}
		}
	}()
}


func (w *CleanupWorker) Stop() {
	close(w.stopChan)
}