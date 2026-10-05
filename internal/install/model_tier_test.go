package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	hero "github.com/hero-engine/hero"
)

func TestRenderAgentModelTier(t *testing.T) {
	src := "---\nname: a\nmodel_tier: %s\nmode: subagent\n---\nbody\n"
	cases := []struct {
		tier   string
		claude bool
		want   string
	}{
		{"light", true, "---\nname: a\nmodel: haiku\nmode: subagent\n---\nbody\n"},
		{"standard", true, "---\nname: a\nmodel: sonnet\nmode: subagent\n---\nbody\n"},
		{"deep", true, "---\nname: a\nmode: subagent\n---\nbody\n"},
		{"light", false, "---\nname: a\nmode: subagent\n---\nbody\n"},
		{"standard", false, "---\nname: a\nmode: subagent\n---\nbody\n"},
	}
	for _, c := range cases {
		got, err := renderAgentModelTier([]byte(strings.Replace(src, "%s", c.tier, 1)), c.claude)
		if err != nil {
			t.Fatalf("%s claude=%v: %v", c.tier, c.claude, err)
		}
		if string(got) != c.want {
			t.Errorf("%s claude=%v:\ngot  %q\nwant %q", c.tier, c.claude, got, c.want)
		}
	}
}

func TestRenderAgentModelTier_Unknown(t *testing.T) {
	_, err := renderAgentModelTier([]byte("---\nname: a\nmodel_tier: huge\n---\n"), true)
	if err == nil || !strings.Contains(err.Error(), `"huge"`) {
		t.Fatalf("want error naming the value, got %v", err)
	}
}

func TestInstallFlat_UnknownTierNamesFile(t *testing.T) {
	contentFS := fstest.MapFS{"agents/bad.md": {Data: []byte("---\nname: bad\nmodel_tier: huge\n---\n")}}
	opts := Options{ContentFS: contentFS, Domain: "engineering", Target: TargetClaude}
	err := installFlat(opts, &Result{}, "agents", t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "agents/bad.md") || !strings.Contains(err.Error(), `"huge"`) {
		t.Fatalf("want error naming agents/bad.md and \"huge\", got %v", err)
	}
}

func TestRenderAgentModelTier_Unchanged(t *testing.T) {
	for _, src := range []string{
		"---\nname: a\n---\nmodel_tier: light\n",
		"no frontmatter\nmodel_tier: light\n",
		"---\nname: a\n---\nbody\n",
	} {
		got, err := renderAgentModelTier([]byte(src), true)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != src {
			t.Errorf("changed %q to %q", src, got)
		}
	}
}

func TestInstallFlat_ModelTierPerTarget(t *testing.T) {
	contentFS := fstest.MapFS{
		"agents/fast.md":  {Data: []byte("---\nname: fast\nmodel_tier: light\n---\nbody\n")},
		"agents/plain.md": {Data: []byte("---\nname: plain\n---\nbody\n")},
	}
	for _, target := range []Target{TargetClaude, TargetOpenCode, TargetCursor, TargetGrok, TargetGeneric} {
		dir := t.TempDir()
		opts := Options{ContentFS: contentFS, Domain: "engineering", Target: target}
		if err := installFlat(opts, &Result{}, "agents", dir); err != nil {
			t.Fatalf("%s: %v", target, err)
		}
		fast, _ := os.ReadFile(filepath.Join(dir, "fast.md"))
		if strings.Contains(string(fast), "model_tier") {
			t.Errorf("%s: model_tier leaked: %q", target, fast)
		}
		if hasModel := strings.Contains(string(fast), "model: haiku"); hasModel != (target == TargetClaude) {
			t.Errorf("%s: model: haiku present=%v: %q", target, hasModel, fast)
		}
		plain, _ := os.ReadFile(filepath.Join(dir, "plain.md"))
		if string(plain) != "---\nname: plain\n---\nbody\n" {
			t.Errorf("%s: untagged agent changed: %q", target, plain)
		}
	}
}

func TestShippedAgents_ModelTiers(t *testing.T) {
	want := map[string]string{
		"issue-tracker":          "haiku",
		"session-primer":         "haiku",
		"comment-scrubber":       "sonnet",
		"deadcode-scrubber":      "sonnet",
		"dedup-scrubber":         "sonnet",
		"defensive-scrubber":     "sonnet",
		"dependency-scrubber":    "sonnet",
		"legacy-scrubber":        "sonnet",
		"type-scrubber":          "sonnet",
		"documentation-engineer": "sonnet",
		"functional-qa-engineer": "sonnet",
		"dependency-analyst":     "sonnet",
	}
	for _, domain := range hero.AvailableDomains() {
		domainFS, err := hero.DomainFS(domain)
		if err != nil {
			t.Fatalf("DomainFS(%s): %v", domain, err)
		}
		dir := t.TempDir()
		opts := Options{ContentFS: hero.OverlayFS(domainFS, hero.CoreFS()), Domain: domain, Target: TargetClaude}
		// Fails on any shipped template with an unknown model_tier.
		if err := installFlat(opts, &Result{}, "agents", dir); err != nil {
			t.Fatalf("%s: %v", domain, err)
		}
		if domain != "engineering" {
			continue
		}
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			name := strings.TrimSuffix(e.Name(), ".md")
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if _, tagged := want[name]; !tagged && strings.Contains(string(data), "\nmodel: ") {
				t.Errorf("%s: untagged agent got a model line", name)
			}
		}
		for name, model := range want {
			data, err := os.ReadFile(filepath.Join(dir, name+".md"))
			if err != nil {
				t.Fatalf("%s: %v", name, err)
			}
			if !strings.Contains(string(data), "\nmodel: "+model+"\n") {
				t.Errorf("%s: want model: %s", name, model)
			}
		}
	}
}
