package main

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/keysym.h>
#include <X11/extensions/XTest.h>
#include <X11/XKBlib.h>

static Display *dpy = NULL;
static int spare_keycode = 0;

static int x11_init(void) {
	dpy = XOpenDisplay(NULL);
	if (!dpy) return -1;

	// Find a spare keycode for temporary remapping of unusual characters
	int min_kc, max_kc;
	XDisplayKeycodes(dpy, &min_kc, &max_kc);
	for (int kc = max_kc; kc >= min_kc; kc--) {
		int syms_per_kc;
		KeySym *syms = XGetKeyboardMapping(dpy, kc, 1, &syms_per_kc);
		int empty = 1;
		for (int i = 0; i < syms_per_kc; i++) {
			if (syms[i] != NoSymbol) { empty = 0; break; }
		}
		XFree(syms);
		if (empty) { spare_keycode = kc; break; }
	}

	return 0;
}

static void x11_close(void) {
	if (dpy) { XCloseDisplay(dpy); dpy = NULL; }
}

// Type a keysym that already has a keycode in the current keymap.
// Returns 1 if it handled it, 0 if it needs temporary remapping.
static int type_keysym_fast(KeySym ks) {
	KeyCode kc = XKeysymToKeycode(dpy, ks);
	if (kc == 0) return 0;

	// Check if shift is needed
	int need_shift = 0;
	KeySym ks0 = XkbKeycodeToKeysym(dpy, kc, 0, 0);
	KeySym ks1 = XkbKeycodeToKeysym(dpy, kc, 0, 1);
	if (ks1 == ks && ks0 != ks) need_shift = 1;

	if (need_shift) {
		KeyCode shift = XKeysymToKeycode(dpy, XK_Shift_L);
		if (shift) XTestFakeKeyEvent(dpy, shift, True, 0);
		XTestFakeKeyEvent(dpy, kc, True, 0);
		XTestFakeKeyEvent(dpy, kc, False, 0);
		if (shift) XTestFakeKeyEvent(dpy, shift, False, 0);
	} else {
		XTestFakeKeyEvent(dpy, kc, True, 0);
		XTestFakeKeyEvent(dpy, kc, False, 0);
	}
	return 1;
}

// Type a keysym by temporarily remapping a spare keycode.
// Requires sync calls so it's slower — only used for chars not in the keymap.
static void type_keysym_remap(KeySym ks) {
	if (spare_keycode == 0) return;

	XChangeKeyboardMapping(dpy, spare_keycode, 1, &ks, 1);
	XSync(dpy, False);

	XTestFakeKeyEvent(dpy, spare_keycode, True, 0);
	XTestFakeKeyEvent(dpy, spare_keycode, False, 0);
	XSync(dpy, False);

	KeySym nosym = NoSymbol;
	XChangeKeyboardMapping(dpy, spare_keycode, 1, &nosym, 1);
	XSync(dpy, False);
}

// Type an array of keysyms. Normal chars are batched; unusual ones
// force a flush and use temporary remapping.
static void x11_type_keysyms(unsigned long *keysyms, int count) {
	if (!dpy || count == 0) return;
	int has_pending = 0;

	for (int i = 0; i < count; i++) {
		if (type_keysym_fast((KeySym)keysyms[i])) {
			has_pending = 1;
		} else {
			// Flush any pending fast events before doing a slow remap
			if (has_pending) { XFlush(dpy); has_pending = 0; }
			type_keysym_remap((KeySym)keysyms[i]);
		}
	}

	if (has_pending) XFlush(dpy);
}

static void x11_send_backspaces(int count) {
	if (!dpy || count <= 0) return;
	KeyCode kc = XKeysymToKeycode(dpy, XK_BackSpace);
	if (kc == 0) return;
	for (int i = 0; i < count; i++) {
		XTestFakeKeyEvent(dpy, kc, True, 0);
		XTestFakeKeyEvent(dpy, kc, False, 0);
	}
	XFlush(dpy);
}

static void x11_send_keysym(unsigned long ks) {
	if (!dpy) return;
	if (!type_keysym_fast((KeySym)ks)) {
		type_keysym_remap((KeySym)ks);
		return;
	}
	XFlush(dpy);
}
*/
import "C"

import (
	"fmt"
	"log"
	"unicode/utf8"
	"unsafe"
)

var (
	typeTextFn       = typeText
	sendKeyFn        = sendKey
	sendBackspacesFn = sendBackspaces
)

type Typer struct {
	cmdCh       chan TypeCommand
	compCurrent string
}

type TypeCommand struct {
	Kind TypeCommandKind
	Text string
	Key  string
}

type TypeCommandKind int

const (
	CommandText TypeCommandKind = iota
	CommandKey
	CommandCompositionUpdate
	CommandCompositionCommit
	CommandClear
)

func NewTyper() *Typer {
	t := &Typer{
		cmdCh: make(chan TypeCommand, 64),
	}
	go t.loop()
	return t
}

func InitX11() error {
	if C.x11_init() != 0 {
		return fmt.Errorf("failed to open X11 display (is DISPLAY set?)")
	}
	return nil
}

func CloseX11() {
	C.x11_close()
}

// SendText queues committed text to type.
func (t *Typer) SendText(text string) {
	t.cmdCh <- TypeCommand{Kind: CommandText, Text: text}
}

// SendKey queues a special key press.
func (t *Typer) SendKey(key string) {
	t.cmdCh <- TypeCommand{Kind: CommandKey, Key: key}
}

// SetComposition updates the in-progress composition target.
func (t *Typer) SetComposition(s string) {
	t.cmdCh <- TypeCommand{Kind: CommandCompositionUpdate, Text: s}
}

// CommitComposition reconciles the current preview with the final committed text.
func (t *Typer) CommitComposition(text string) {
	t.cmdCh <- TypeCommand{Kind: CommandCompositionCommit, Text: text}
}

// Clear resets all state.
func (t *Typer) Clear() {
	t.cmdCh <- TypeCommand{Kind: CommandClear}
}

func (t *Typer) loop() {
	for {
		cmd := <-t.cmdCh
		t.execCommand(cmd)
	}
}

func (t *Typer) replaceComposition(text string) {
	if text == t.compCurrent {
		return
	}

	deleteCount := utf8.RuneCountInString(t.compCurrent)
	if deleteCount > 0 {
		sendBackspacesFn(deleteCount)
	}

	if len(text) > 0 {
		typeTextFn(text)
	}

	t.compCurrent = text
}

func (t *Typer) execCommand(cmd TypeCommand) {
	switch cmd.Kind {
	case CommandText:
		typeTextFn(cmd.Text)
	case CommandKey:
		sendKeyFn(cmd.Key)
	case CommandCompositionUpdate:
		t.replaceComposition(cmd.Text)
	case CommandCompositionCommit:
		t.replaceComposition(cmd.Text)
		t.compCurrent = ""
	case CommandClear:
		t.replaceComposition("")
		t.compCurrent = ""
	}
}

func runeToKeysym(r rune) C.ulong {
	switch r {
	case '\n', '\r':
		return C.XK_Return
	case '\t':
		return C.XK_Tab
	}
	// Latin-1 range: keysym == codepoint
	if r >= 0x20 && r <= 0xFF {
		return C.ulong(r)
	}
	// Unicode: keysym = 0x01000000 | codepoint
	if r > 0xFF {
		return C.ulong(0x01000000 | r)
	}
	return C.ulong(r)
}

func typeText(text string) {
	runes := []rune(text)
	if len(runes) == 0 {
		return
	}
	keysyms := make([]C.ulong, len(runes))
	for i, r := range runes {
		keysyms[i] = runeToKeysym(r)
	}
	C.x11_type_keysyms((*C.ulong)(unsafe.Pointer(&keysyms[0])), C.int(len(keysyms)))
}

func sendBackspaces(count int) {
	if count <= 0 {
		return
	}
	C.x11_send_backspaces(C.int(count))
}

var keyMap = map[string]C.ulong{
	"Backspace":  C.XK_BackSpace,
	"Enter":      C.XK_Return,
	"Tab":        C.XK_Tab,
	"Escape":     C.XK_Escape,
	"ArrowLeft":  C.XK_Left,
	"ArrowRight": C.XK_Right,
	"ArrowUp":    C.XK_Up,
	"ArrowDown":  C.XK_Down,
	"Delete":     C.XK_Delete,
	"Home":       C.XK_Home,
	"End":        C.XK_End,
}

func sendKey(key string) {
	ks, ok := keyMap[key]
	if !ok {
		log.Printf("unknown key: %s", key)
		return
	}
	C.x11_send_keysym(ks)
}
