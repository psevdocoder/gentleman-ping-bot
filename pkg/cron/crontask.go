package cron

import "context"

type Task interface {
	Work(ctx context.Context) error
	Name() string
}
