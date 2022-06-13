package utils

import (
	"fmt"
	"os"
	"time"
)

type ProgressBar struct {
	total          int
	current        int
	lastUpdateTime time.Time
	startTime      time.Time
	messageFunc    func(total, current int, elapsed time.Duration) string
}

func NewProgressBar(messageFunc func(total, current int, elapsed time.Duration) string) *ProgressBar {
	return &ProgressBar{
		startTime:   time.Now(),
		messageFunc: messageFunc,
	}
}

func (p *ProgressBar) Update(current, total int) {
	p.current = current
	p.total = total
}

func (p *ProgressBar) Print() {
	if p.total == 0 {
		return
	}
	now := time.Now()
	if p.current < p.total &&
		now.Sub(p.lastUpdateTime) < time.Second/10 {
		return
	}
	p.lastUpdateTime = now

	message := p.messageFunc(p.total, p.current, now.Sub(p.startTime))

	if p.current >= p.total {
		fmt.Fprintf(os.Stderr, "\r%s 100%% %s    \n", message, time.Since(p.startTime).Truncate(time.Second/10))
	} else {
		fmt.Fprintf(os.Stderr, "\r%s %.1f%% %s", message, float64(p.current)/float64(p.total)*100, time.Since(p.startTime).Truncate(time.Second/10))
	}
}
