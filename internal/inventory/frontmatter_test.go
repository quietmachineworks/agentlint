package inventory

import "testing"

func TestParseFrontmatterFoldsContinuationLines(t *testing.T) {
	text := "---\nname: math\ndescription:\n  \"One sentence split\n  across lines.\"\n---\nbody\n"
	fm := ParseFrontmatter(text)
	if !fm.Present {
		t.Fatal("frontmatter not detected")
	}
	description, ok := fm.Get("description")
	if !ok || description == "" {
		t.Fatalf("folded description read as empty: %q", description)
	}
	if name, _ := fm.Get("name"); name != "math" {
		t.Fatalf("name is %q", name)
	}
}

func TestParseFrontmatterAbsent(t *testing.T) {
	if ParseFrontmatter("# Just a body\n").Present {
		t.Fatal("reported frontmatter where there is none")
	}
}

func TestSplitList(t *testing.T) {
	got := SplitList(`Bash, "Read", 'Write'`)
	want := []string{"Bash", "Read", "Write"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
