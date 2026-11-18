package main

import (
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"unitystarter/locales"
)

func TestDownloadsCollapsePatchVersionsByMajorLine(t *testing.T) {
	a := fixture()
	a.page = pageDownloads
	a.jobs = map[string]*downloadJob{}
	a.downloadsReady = true
	a.downloads = []downloadEntry{
		{Engine: kindUnity, Version: "7000.0.0a16", Size: 1, Date: time.Now()},
		{Engine: kindUnity, Version: "7000.0.0a12", Size: 1},
		{Engine: kindUnity, Version: "6000.3.28f1", Size: 1},
		{Engine: kindTuanjie, Version: "2022.3.62t17", Size: 1},
	}
	tt := ui.NewTester(a.view, 1180, 760)
	if !tt.HasText("7000.0") || !tt.HasText("7000.0.0a16") {
		t.Fatalf("latest patch per major line is missing: %q", tt.Texts())
	}
	if tt.HasText("7000.0.0a12") {
		t.Fatalf("older patch is visible before expanding: %q", tt.Texts())
	}
	if err := tt.Click(locales.DownloadsShowAll("2")); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("7000.0.0a12") {
		t.Fatalf("expanded major line did not reveal the older patch: %q", tt.Texts())
	}
	if err := tt.Click(locales.DownloadsShowLatest()); err != nil {
		t.Fatal(err)
	}
	if tt.HasText("7000.0.0a12") {
		t.Fatalf("collapsed major line still shows the older patch: %q", tt.Texts())
	}
}

func TestWidePagesFillTheWindow(t *testing.T) {
	a := fixture()
	a.page = pageStore
	tt := ui.NewTester(a.view, 1600, 900)
	button, ok := tt.Find(locales.OpenExternal(locales.LinkAssetStoreUnity()))
	if !ok || button.X < 1300 || button.X+button.W > 1600 {
		t.Fatalf("store content did not adapt to a wide window: %+v, found=%v", button, ok)
	}
}
