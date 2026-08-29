package tui

import "testing"

func TestLooksLikeTermReply(t *testing.T) {
	yes := []string{
		"rgb:2121/2121/2121",
		"RGB:1e1e/1e1e/1e1e",
		"11;rgb:2121/2121/2121",
		"]11;rgb:0000/0000/0000",
		"/2121",
		"/2121/2121",
		"2121/2121/2121",
		"1a1a/1b1b/2c2c",
	}
	for _, s := range yes {
		if !looksLikeTermReply(s) {
			t.Errorf("want leak %q", s)
		}
	}
	no := []string{
		"",
		"/",
		"/nvim",
		"/alacritty",
		"r",
		"rg",
		"2024",
		"config",
	}
	for _, s := range no {
		if looksLikeTermReply(s) {
			t.Errorf("want real input %q", s)
		}
	}
}

func TestDropBuffered(t *testing.T) {
	if !dropBuffered("rgb") || !dropBuffered("rgb:") {
		t.Fatal("rgb prefix should drop")
	}
	if dropBuffered("r") || dropBuffered("/") || dropBuffered("/n") {
		t.Fatal("single command keys should not drop")
	}
}

func TestIngestDropsColorBurst(t *testing.T) {
	m := New()
	m.scanning = false
	m.mode = modeBrowse

	for _, r := range "rgb:2121/2121/2121" {
		var cmd any
		m, cmd = m.ingestBrowseKey(fakeKey(string(r)))
		_ = cmd
		if m.mode == modeFilter {
			t.Fatalf("search opened on burst, buf=%q filter=%q", m.junkBuf, m.filter)
		}
	}
	if m.mode != modeBrowse {
		t.Fatalf("mode=%d", m.mode)
	}
}

func TestIngestSlashThenLetterOpensSearch(t *testing.T) {
	m := New()
	m.scanning = false
	m.mode = modeBrowse

	m, _ = m.ingestBrowseKey(fakeKey("/"))
	if m.mode == modeFilter {
		t.Fatal("search should wait for settle")
	}
	m, _ = m.ingestBrowseKey(fakeKey("n"))
	if m.mode != modeFilter {
		t.Fatalf("want search after /n, mode=%d buf=%q", m.mode, m.junkBuf)
	}
}

func TestIngestSlashHexBurstDoesNotSearch(t *testing.T) {
	m := New()
	m.scanning = false
	m.mode = modeBrowse
	for _, r := range "/2121" {
		m, _ = m.ingestBrowseKey(fakeKey(string(r)))
	}
	if m.mode == modeFilter {
		t.Fatal(" /2121 should be dropped as a color fragment")
	}
}
