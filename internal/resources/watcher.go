package resources

import (
	"log"
	"time"

	"github.com/fsnotify/fsnotify"
)

func watchDir(directory string, callback func()) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = watcher.Add(directory)
	if err != nil {
		return err
	}

	reload := make(chan struct{})
	go scheduleReload(reload, callback)
	go handleWatcher(watcher, reload)
	return nil
}

func handleWatcher(watcher *fsnotify.Watcher, reload chan<- struct{}) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write | fsnotify.Remove | fsnotify.Create) {
				reload <- struct{}{}
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("resource watcher error: %v\n", err)
		}
	}
}

func scheduleReload(reload <-chan struct{}, callback func()) {
	var timer *time.Timer = nil
	var c <-chan time.Time = nil
	duration := time.Millisecond * 500
	for {
		select {
		case <-reload:
			if timer != nil {
				timer.Reset(duration)
			} else {
				timer = time.NewTimer(duration)
				c = timer.C
			}

		case <-c:
			c = nil
			timer = nil
			callback()
		}
	}
}
