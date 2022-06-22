package utils

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type ProgressBar struct {
	total          int
	current        int
	lastUpdateTime time.Time
	startTime      time.Time
}

func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		startTime: time.Now(),
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

	if p.current >= p.total {
		fmt.Fprintf(os.Stderr, "\r%-60s| 100%%  %-5s\n", strings.Repeat("#", 60), time.Since(p.startTime).Truncate(time.Second))
	} else {
		fmt.Fprintf(os.Stderr, "\r%-60s| %.1f%% %-5s", strings.Repeat("#", int(float64(p.current)/float64(p.total)*60)), float64(p.current)/float64(p.total)*100, time.Since(p.startTime).Truncate(time.Second))
	}
}
