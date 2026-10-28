package bootstrap

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestStartModulesOnlyStartsEnabledModules(t *testing.T) {
	var events []string
	modules := []moduleSpec{
		{name: "disabled", enabled: false, start: func() (context.CancelFunc, error) {
			t.Fatal("disabled module started")
			return nil, nil
		}},
		{name: "active", enabled: true, start: func() (context.CancelFunc, error) {
			events = append(events, "start")
			return func() { events = append(events, "stop") }, nil
		}},
	}
	stops, err := startModules(modules)
	if err != nil {
		t.Fatalf("start modules: %v", err)
	}
	if !reflect.DeepEqual(events, []string{"start"}) {
		t.Fatalf("start events = %v", events)
	}
	stopModules(stops)
	if !reflect.DeepEqual(events, []string{"start", "stop"}) {
		t.Fatalf("shutdown events = %v", events)
	}
}

func TestStartModulesUnwindsInReverseOrder(t *testing.T) {
	var events []string
	boom := errors.New("boom")
	start := func(name string) moduleSpec {
		return moduleSpec{name: name, enabled: true, start: func() (context.CancelFunc, error) {
			events = append(events, "start:"+name)
			return func() { events = append(events, "stop:"+name) }, nil
		}}
	}
	modules := []moduleSpec{start("first"), start("second"), {
		name: "broken", enabled: true,
		start: func() (context.CancelFunc, error) { return nil, boom },
	}}
	stops, err := startModules(modules)
	if !errors.Is(err, boom) || stops != nil {
		t.Fatalf("start result = %v, %v", stops, err)
	}
	want := []string{"start:first", "start:second", "stop:second", "stop:first"}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestStartModulesRejectsDuplicateAndIncompleteModules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		modules []moduleSpec
		message string
	}{
		{name: "duplicate", modules: []moduleSpec{{name: "a"}, {name: "a"}}, message: "duplicate"},
		{name: "unnamed", modules: []moduleSpec{{name: " "}}, message: "invalid"},
		{name: "missing start", modules: []moduleSpec{{name: "a", enabled: true}}, message: "start function"},
		{name: "missing stop", modules: []moduleSpec{{name: "a", enabled: true, start: func() (context.CancelFunc, error) {
			return nil, nil
		}}}, message: "shutdown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := startModules(tc.modules)
			if err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error = %v, want %q", err, tc.message)
			}
		})
	}
}
