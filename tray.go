package main

import (
	"sync"

	"github.com/getlantern/systray"
)

type trayState struct {
	enabled   bool
	composing bool
}

type Tray struct {
	mu              sync.Mutex
	currentState    trayState
	quitRequested   chan struct{}
	toggleRequested chan bool
	stateCh         chan trayState
	exited          chan struct{}
	quitOnce        sync.Once
}

func StartTray(title, tooltip string) *Tray {
	t := &Tray{
		currentState:    trayState{enabled: true},
		quitRequested:   make(chan struct{}, 1),
		toggleRequested: make(chan bool, 1),
		stateCh:         make(chan trayState, 4),
		exited:          make(chan struct{}),
	}

	go systray.Run(func() {
		current := trayState{enabled: true}
		systray.SetIcon(trayIconBlue)
		systray.SetTitle(title)
		systray.SetTooltip(tooltip)

		status := systray.AddMenuItem(tooltip, "Current server status")
		status.Disable()
		enableItem := systray.AddMenuItemCheckbox("Typing Enabled", "Toggle whether goremotetype can type", true)
		systray.AddSeparator()

		quitItem := systray.AddMenuItem("Quit", "Quit goremotetype")
		go func() {
			<-quitItem.ClickedCh
			select {
			case t.quitRequested <- struct{}{}:
			default:
			}
		}()

		go func() {
			for {
				select {
				case <-enableItem.ClickedCh:
					next := !current.enabled
					select {
					case t.toggleRequested <- next:
					default:
					}
				case state, ok := <-t.stateCh:
					if !ok {
						return
					}
					current = state
					if !state.enabled {
						systray.SetIcon(trayIconGray)
						systray.SetTooltip("Typing disabled")
						status.SetTitle("Typing disabled")
						enableItem.Uncheck()
					} else if state.composing {
						systray.SetIcon(trayIconGreen)
						systray.SetTooltip("Composing")
						status.SetTitle("Composing")
						enableItem.Check()
					} else {
						systray.SetIcon(trayIconBlue)
						systray.SetTooltip(tooltip)
						status.SetTitle(tooltip)
						enableItem.Check()
					}
				}
			}
		}()
	}, func() {
		close(t.exited)
	})

	return t
}

func (t *Tray) QuitRequested() <-chan struct{} {
	if t == nil {
		return nil
	}
	return t.quitRequested
}

func (t *Tray) ToggleRequested() <-chan bool {
	if t == nil {
		return nil
	}
	return t.toggleRequested
}

func (t *Tray) Close() {
	if t == nil {
		return
	}

	t.quitOnce.Do(func() {
		close(t.stateCh)
		systray.Quit()
		<-t.exited
	})
}

func (t *Tray) SetComposing(composing bool) {
	t.setState(func(state *trayState) {
		state.composing = composing
	})
}

func (t *Tray) SetEnabled(enabled bool) {
	t.setState(func(state *trayState) {
		state.enabled = enabled
	})
}

func (t *Tray) setState(update func(*trayState)) {
	if t == nil {
		return
	}

	t.mu.Lock()
	state := t.currentState
	update(&state)
	t.currentState = state
	t.mu.Unlock()

	select {
	case t.stateCh <- state:
	default:
		select {
		case <-t.stateCh:
		default:
		}
		t.stateCh <- state
	}
}
