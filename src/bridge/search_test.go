package bridge

import "testing"

func TestSearchSmartMatchSupportsTokenizedQueries(t *testing.T) {
	if !searchSmartMatch("src/features/workbench/WorkbenchApp.tsx", "workbench app", false) {
		t.Fatal("expected tokenized path query to match")
	}
	if !searchSmartMatch("WorkbenchBrowserContent", "wbc", false) {
		t.Fatal("expected subsequence query to match")
	}
	if searchSmartMatch("main.go", "settings modal", false) {
		t.Fatal("unexpected unrelated match")
	}
}

func TestMatchArchiveBrowseQuickSearchMatchesDirectoryTokens(t *testing.T) {
	file := ArchiveBrowseFile{
		Name:         "WorkbenchApp.tsx",
		Directory:    "src/features/workbench",
		RelativePath: "src/features/workbench/WorkbenchApp.tsx",
		Extension:    ".tsx",
	}
	ok, matchKind := matchArchiveBrowseQuickSearch(file, "features app")
	if !ok {
		t.Fatal("expected archive browse quick search to match multi-token query")
	}
	if matchKind != "name" && matchKind != "directory" && matchKind != "path" {
		t.Fatalf("unexpected match kind: %s", matchKind)
	}
}
