package worker

import (
	"log"
	"swift-seat/internal/repository"
	"time"

	"github.com/go-co-op/gocron"
)

type Cronjob struct{
	repo *repository.PostgresDB
}

func NewCronjobWorker(repo *repository.PostgresDB) *Cronjob {
	return &Cronjob{
		repo: repo,
	}
}

func (c *Cronjob) StartWorkers() {
    s := gocron.NewScheduler(time.Local)

    // تنظیم اجرا در ساعت 02:00 بامداد هر روز
    s.Every(1).Day().At("02:00").Do(func() {
        count, err := c.repo.DeactivateExpiredEvents()
        if err != nil {
            log.Printf("Worker Error: %v", err)
            return
        }
        log.Printf("Worker finished: %d events deactivated", count)
    })

    s.StartAsync()
}