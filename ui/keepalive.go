package ui

import (
	"time"
)

// Keepalive manages the keepalive goroutine.
type Keepalive struct {
	running bool
	stopCh  chan struct{}
}

// NewKeepalive creates a new keepalive manager.
func NewKeepalive() *Keepalive {
	return &Keepalive{}
}

// Start begins the keepalive loop.
func (k *Keepalive) Start(interval time.Duration, doCheck func()) {
	if k.running {
		return
	}
	k.running = true
	k.stopCh = make(chan struct{})
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-k.stopCh:
				return
			case <-ticker.C:
				doCheck()
			}
		}
	}()
}

// Stop terminates the keepalive loop.
func (k *Keepalive) Stop() {
	if !k.running {
		return
	}
	k.running = false
	close(k.stopCh)
}

// IsRunning returns whether the keepalive is active.
func (k *Keepalive) IsRunning() bool {
	return k.running
}

// Restart stops and restarts with a new interval.
func (k *Keepalive) Restart(interval time.Duration, doCheck func()) {
	k.Stop()
	k.Start(interval, doCheck)
}

// DelayedFunc runs fn after the given delay.
// The caller must ensure UI updates are synchronized via mainWindow.Synchronize.
func DelayedFunc(delay time.Duration, fn func()) {
	go func() {
		time.Sleep(delay)
		fn()
	}()
}
