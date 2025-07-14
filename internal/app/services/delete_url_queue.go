package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TPizik/url-shortener/internal/app/models"
	"go.uber.org/zap"
)

type urlStorage interface {
	DoDeleteURLTasks(ctx context.Context, tasks []models.DeleteURLsTask) error
}

type DeleteURLQueue struct {
	ch         chan *models.DeleteURLsTask
	urlStorage urlStorage
	tasks      []models.DeleteURLsTask
	logger     *zap.SugaredLogger
}

func NewDeleteURLQueue(urlStorage urlStorage, maxWorker int) *DeleteURLQueue {
	return &DeleteURLQueue{
		urlStorage: urlStorage,
		ch:         make(chan *models.DeleteURLsTask, maxWorker),
		tasks:      make([]models.DeleteURLsTask, 0, 500),
		logger:     InitLogger(),
	}
}

func (q *DeleteURLQueue) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case task := <-q.ch:
			q.tasks = append(q.tasks, *task)
		case <-ctx.Done():
			if err := q.doDeleteTasks(); err != nil {
				q.logger.Errorln(err.Error())
			}
		case <-ticker.C:
			if err := q.doDeleteTasks(); err != nil {
				q.logger.Errorln(err.Error())
			}
		}
	}
}

func (q *DeleteURLQueue) Push(task *models.DeleteURLsTask) {
	q.ch <- task
}

func (q *DeleteURLQueue) doDeleteTasks() error {
	if len(q.tasks) == 0 {
		return nil
	}

	if err := q.urlStorage.DoDeleteURLTasks(context.Background(), q.tasks); err != nil {
		return err
	}

	Sugar.Infoln(fmt.Sprintf("Successfully did %d delete url tasks", len(q.tasks)))
	q.tasks = q.tasks[0:]
	return nil
}
