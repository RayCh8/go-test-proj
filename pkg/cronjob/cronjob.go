package cronjob

import (
	"context"
	"time"

	"github.com/AmazingTalker/go-rpc-kit/logkit"
)

func Execute(ctx context.Context) error {

	// here is a demo

	jobs := make([]int, 360)

	for _, i := range jobs {
		logkit.Infof(ctx, "Job %d executed", i)
		time.Sleep(time.Second)
	}

	return nil
}
