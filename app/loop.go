// Package app is
package app

import (
	"sync"
	"time"

	"github.com/gdamore/tcell"
)

var refreshScreen func() error
var refreshIfView func(v viewMode) error
var widgets *widgetRegistry

//RenderLoop runs dry
// nolint: gocyclo
func RenderLoop(dry *Dry) {

	//use to signal rendering
	renderChan := make(chan struct{})

	var closingLock sync.RWMutex
	refreshScreen = func() error {
		closingLock.RLock()
		defer closingLock.RUnlock()

		renderChan <- struct{}{}
		return nil
	}

	refreshIfView = func(v viewMode) error {
		if v == dry.viewMode() {
			return refreshScreen()
		}
		return nil
	}

	dryOutputChan := dry.OuputChannel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		for range renderChan {
			if dry.isPaused() {
				continue
			}
			dry.gscreen().Clear()
			render(dry)
		}
	}()

	refreshScreen()

	go func() {
		statusBar := widgets.MessageBar
		for dryMessage := range dryOutputChan {
			statusBar.Message(dryMessage, 10*time.Second)
			statusBar.Render()
		}
	}()

	handler := viewsToHandlers[dry.viewMode()]
	//main loop that handles termui events
loop:
	for {
		event := dry.screen.Poll()
		if dry.isPaused() {
			continue
		}
		switch ev := event.(type) {
		case *tcell.EventInterrupt:
			break loop
		case *tcell.EventKey:
			//Ctrl+C breaks the loop (and exits dry) no matter what
			if ev.Key() == tcell.KeyCtrlC || ev.Rune() == 'Q' {
				break loop
			}
			handler.handle(ev, func(eh eventHandler) {
				handler = eh
			})

		case *tcell.EventResize:
			//Reload dry ui elements
			//TODO widgets.reload()
		}
	}

	//make the global refreshScreen func a noop before closing
	closingLock.Lock()
	refreshScreen = func() error {
		return nil
	}
	closingLock.Unlock()

	//Close the channel used to notify the rendering goroutine
	close(renderChan)
	//Wait for the rendering goroutine to exit
	wg.Wait()
}
