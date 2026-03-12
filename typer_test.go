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
		typer := &Typer{}
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
		typer := &Typer{}
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
		typer := &Typer{}
		typer.execCommand(TypeCommand{Kind: CommandCompositionUpdate, Text: "temp"})
		typer.execCommand(TypeCommand{Kind: CommandClear})

		want := []string{
			"text:temp",
			"backspace:4",
		}
		if !reflect.DeepEqual(*ops, want) {
			t.Fatalf("ops mismatch\n got: %#v\nwant: %#v", *ops, want)
		}
	})
}

func TestTyperPreservesCommandOrder(t *testing.T) {
	withFakeTyperIO(t, func(ops *[]string) {
		typer := &Typer{}
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
