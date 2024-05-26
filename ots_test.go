package gots_test

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pgaskin/gots"
)

func init() {
	gots.Compile()
}

func TestOTS(t *testing.T) {
	if _, err := os.Stat(filepath.Join("ots", "tests", "fonts")); errors.Is(err, fs.ErrNotExist) {
		t.Skip("ots test fonts directory doesn't exist (is the git submodule initialized?), skipping tests")
		return
	}
	for _, kind := range []string{"bad", "fuzzing", "good"} {
		t.Run(strings.ToUpper(kind[0:1])+kind[1:], func(t *testing.T) {
			dir := filepath.Join("ots", "tests", "fonts", kind)
			fis, err := os.ReadDir(dir)
			if err != nil {
				panic(err)
			}
			for _, fi := range fis {
				t.Run(fi.Name(), func(t *testing.T) {
					fn := filepath.Join(dir, fi.Name())
					input, err := os.ReadFile(fn)
					if err != nil {
						panic(err)
					}
					extF := filepath.Ext(fi.Name())
					extI := gots.Extension(input)
					if kind == "good" && extI != extF && !(extI == ".ttc" && extF == ".ttf") {
						t.Errorf("%s: file extension is %s, we detected %s", fn, extF, extI)
					}
					output, err := gots.Process(input, gots.WithMessages(func(level gots.MessageLevel, msg string) {
						t.Logf("%s: ots: %s: %s", fn, level, msg)
					}))
					if kind != "fuzzing" {
						if success := (kind == "good"); (err == nil) != success {
							t.Errorf("%s: expected success=%t, got error: %v", fn, success, err)
						}
						if err == nil && gots.Extension(output) == "" {
							t.Errorf("%s: failed to get extension of processed font", fn)
						}
					}
					if err == nil {
						switch fi.Name() {
						case "4765a8901e377d1e767f67e1cc768ae3c9207bd1.ttc":
						case "7043d3c69c50da8eba1a0ad627b9f6de70e832e5.ttf":
						case "8330c9816493e1adccc0500b414455b85088d7d1.ttf":
						case "b927e6af295696a2307641eb9679d0832dd7c22d.ttf":
						case "f5ff6aaa96256b0e2c1abfdebf592c0987a1637a.ttf":
						case "3ee1ab163f0029bdd8f90b79f2c0e798bc26957b.ttf":
						default:
							output1, err := gots.Process(output)
							if err != nil {
								t.Errorf("%s: re-process processed output: unexpected error: %v", fn, err)
							}
							if !bytes.Equal(output, output1) {
								t.Errorf("%s: re-processed output doesn't match", fn)
							}
						}
					}
				})
			}
		})
	}
}
