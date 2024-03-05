package handlers

import (
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"testing"
)

type testScenarioAddUpdate struct {
	meta          v1.ObjectMeta
	data          map[string]string
	binaryData    map[string][]byte
	expectedFiles map[string]string
}

func TestGenericHandlerImpl_OnAdd(tt *testing.T) {
	scenarios := map[string]testScenarioAddUpdate{
		"base": {
			meta: v1.ObjectMeta{},
			data: map[string]string{
				"some-key": "value",
			},
			binaryData: map[string][]byte{
				"some-key-2": []byte("ZGVtbw=="),
			},
			expectedFiles: map[string]string{
				"some-key":   "value",
				"some-key-2": "demo",
			},
		},
		"ovewritten": {
			meta: v1.ObjectMeta{},
			data: map[string]string{
				"some-key":                 "value",
				"some-key-not-overwritten": "important-value",
			},
			binaryData: map[string][]byte{
				"some-key":       []byte("ZGVtbw=="), //demo
				"some-key-2":     []byte("ZGVtbw=="), //demo
				"some-key-3.txt": []byte("ZGVtbw=="), //demo
			},
			expectedFiles: map[string]string{
				"some-key":                 "demo",
				"some-key-2":               "demo",
				"some-key-3.txt":           "demo",
				"some-key-not-overwritten": "important-value",
			},
		},
	}
	handler := NewGenericHandlerImpl(
		"/tmp",
		func() {

		},
		"0755",
		"some-annotation",
		false,
	)

	for s, scenario := range scenarios {
		tt.Run(s, func(t *testing.T) {
			dest, err := os.MkdirTemp("", "OnAdd")
			assert.Nil(t, err, "failed to create temp dir")
			t.Logf("created dir: %s", dest)
			defer os.RemoveAll(dest)
			if scenario.meta.Annotations == nil {
				scenario.meta.Annotations = map[string]string{}
			}
			scenario.meta.Annotations["some-annotation"] = dest
			handler.OnAdd(scenario.meta, scenario.data, scenario.binaryData, false)
			files, err := os.ReadDir(dest)

			assert.Equal(t, len(scenario.expectedFiles), len(files), "found unexpected number of files")

			for path, expectedContent := range scenario.expectedFiles {
				assert.FileExists(t, filepath.Join(dest, path), "expected file was not created")
				actualContent, err := os.ReadFile(filepath.Join(dest, path))
				assert.Nil(t, err, "unable to read materialized file")
				assert.Equalf(t, expectedContent, string(actualContent), "not expected content for: %s", path)
			}
		})
	}
}
