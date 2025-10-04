package yaml_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v4"
)

func setupSeedCorpus(f *testing.F) {
	root := filepath.Join("yts", "testdata", "data-2022-01-17")
	if err := filepath.WalkDir(root, func(p string, e fs.DirEntry, err error) error {
		if err != nil {
			f.Fatalf("could not read test suite at %q: %s", root, err)
		}
		if e.IsDir() || filepath.Ext(p) != ".yaml" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatalf("could not read test case %q: %s", p, err)
		}
		f.Add(b)
		return nil
	}); err != nil {
		f.Fatalf("could not read test suite: %q", root)
	}
}

func FuzzMarshalUnmarshal(f *testing.F) {
	setupSeedCorpus(f)
	f.Fuzz(func(t *testing.T, in []byte) {
		var v any
		if err := yaml.Unmarshal(in, &v); err != nil {
			return
		}
		if _, err := yaml.Marshal(&v); err != nil {
			t.Fatalf("could not marshal unmarshaled tree: %q: %s", in, err)
		}
	})
}
