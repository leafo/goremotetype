package main

import (
	"fmt"
	"reflect"
	"testing"
)

func withFakeTyperIO(t *testing.T, fn func(*[]string)) {
	t.Helper()

	origTypeText := typeTextFn
	origSendKey := sendKeyFn
	origSendBackspaces := sendBackspacesFn

	ops := []string{}
	typeTextFn = func(text string) {
		ops = append(ops, fmt.Sprintf("text:%s", text))
	}
	sendKeyFn = func(key string) {
		ops = append(ops, fmt.Sprintf("key:%s", key))
	}
	sendBackspacesFn = func(count int) {
		ops = append(ops, fmt.Sprintf("backspace:%d", count))
	}

	t.Cleanup(func() {
		typeTextFn = origTypeText
		sendKeyFn = origSendKey
		sendBackspacesFn = origSendBackspaces
	})

	fn(&ops)
}

func TestTyperCommitCompositionReplacesPreview(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "How is this"})
		typer.execCommand(TypeCommand{Kind: CommandCompositionCommit, Text: "How is this?"})

		want := []string{
			"text:How is this",
			"backspace:11",
			"text:How is this?",
		}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
	})
}

func TestTyperCommitCompositionCancelRemovesPreview(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "draft"})
		typer.execCommand(TypeCommand{Kind: CommandCompositionCommit, Text: ""})

		want := []string{
			"text:draft",
			"backspace:5",
		}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
	})
}

func TestTyperClearRemovesActivePreview(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "temp"})
		typer.execCommand(TypeCommand{Kind: CommandClear})

		want := []string{"text:temp"}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
		if typer.compositionActive {
			t.Fatalf("expected composition to be inactive after clear")
		}
		if typer.compCurrent != "" {
			t.Fatalf("expected compCurrent to be empty after clear, got %q", typer.compCurrent)
		}
	})
}

func TestTyperPreservesCommandOrder(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "How is this"})
		typer.execCommand(TypeCommand{Kind: CommandCompositionCommit, Text: "How is this?"})
		typer.execCommand(TypeCommand{Kind: CommandText, Text: " Fine."})

		want := []string{
			"text:How is this",
			"backspace:11",
			"text:How is this?",
			"text: Fine.",
		}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
	})
}

func TestTyperCommitCompositionClearsActiveStateOnNoOpCommit(t *testing.T) {
	withFakeTyperIO(t, func(_ *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "steady"})
		if !typer.compositionActive {
			t.Fatalf("expected composition to be active after update")
		}

		typer.execCommand(TypeCommand{Kind: CommandCompositionCommit, Text: "steady"})
		if typer.compositionActive {
			t.Fatalf("expected composition to be inactive after no-op commit")
		}
		if typer.compCurrent != "" {
			t.Fatalf("expected compCurrent to be cleared after commit, got %q", typer.compCurrent)
		}
	})
}

func TestTyperDisabledDropsTypingCommands(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandSetEnabled, Enabled: false})
		typer.execCommand(TypeCommand{Kind: CommandText, Text: "ignored"})
		typer.execCommand(TypeCommand{Kind: CommandKey, Key: "Enter"})
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "draft"})
		typer.execCommand(TypeCommand{Kind: CommandCompositionCommit, Text: "draft"})

		if len(*ops) != 0 {
			t.Fatalf("expected no io while disabled, got %#v", *ops)
		}
		if typer.compositionActive {
			t.Fatalf("expected composition to remain inactive while disabled")
		}
		if typer.compCurrent != "" {
			t.Fatalf("expected compCurrent to remain empty while disabled, got %q", typer.compCurrent)
		}
	})
}

func TestTyperReenableAllowsTypingAgain(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{enabled: true}
		typer.execCommand(TypeCommand{Kind: CommandSetEnabled, Enabled: false})
		typer.execCommand(TypeCommand{Kind: CommandSetEnabled, Enabled: true})
		typer.execCommand(TypeCommand{Kind: CommandText, Text: "works"})

		want := []string{"text:works"}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
	})
}
