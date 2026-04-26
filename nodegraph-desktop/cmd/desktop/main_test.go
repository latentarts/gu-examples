package main

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestEnsureAssetsFSRequiresCoreFiles(t *testing.T) {
	okFS := fstest.MapFS{
		"index.html":   {Data: []byte("ok")},
		"main.wasm":    {Data: []byte("ok")},
		"wasm_exec.js": {Data: []byte("ok")},
	}
	if err := ensureAssetsFS(okFS, "ok"); err != nil {
		t.Fatalf("expected valid asset fs, got %v", err)
	}

	missingFS := fstest.MapFS{
		"index.html": {Data: []byte("ok")},
	}
	err := ensureAssetsFS(missingFS, "missing")
	if err == nil {
		t.Fatal("expected missing assets error")
	}
	if _, statErr := fs.Stat(missingFS, "main.wasm"); statErr == nil {
		t.Fatal("expected test fixture to omit main.wasm")
	}
}
