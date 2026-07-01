package cli

import (
	"reflect"
	"testing"
)

func TestParseTagExpr(t *testing.T) {
	add, remove := parseTagExpr("+cli,+go,-old")
	if !reflect.DeepEqual(add, []string{"cli", "go"}) {
		t.Fatalf("expected add [cli go], got %v", add)
	}
	if !reflect.DeepEqual(remove, []string{"old"}) {
		t.Fatalf("expected remove [old], got %v", remove)
	}
}

func TestParseTagExprEmpty(t *testing.T) {
	add, remove := parseTagExpr("")
	if len(add) != 0 || len(remove) != 0 {
		t.Fatalf("expected empty add and remove, got %v / %v", add, remove)
	}
}

func TestApplyTags(t *testing.T) {
	current := []string{"go", "old", "web"}
	add := []string{"cli", "web"}
	remove := []string{"old"}
	result := applyTags(current, add, remove)
	expected := []string{"go", "web", "cli"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestApplyTagsEmptyCurrent(t *testing.T) {
	add := []string{"cli", "go"}
	remove := []string{}
	result := applyTags(nil, add, remove)
	expected := []string{"cli", "go"}
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}
