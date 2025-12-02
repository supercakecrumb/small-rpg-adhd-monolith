package i18n

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Translator loads YAML locale files and provides lookup with fallback.
type Translator struct {
	locales     map[string]map[string]string
	defaultLang string
}

// NewTranslator loads all *.yaml locale files from dir.
// Each file should be named like en.yaml, ru.yaml and contain flat key/value pairs.
func NewTranslator(dir string, defaultLang string) (*Translator, error) {
	t := &Translator{
		locales:     make(map[string]map[string]string),
		defaultLang: defaultLang,
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".yaml") {
			return nil
		}
		lang := strings.TrimSuffix(d.Name(), ".yaml")
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read locale %s: %w", path, readErr)
		}
		kv := make(map[string]string)
		if unmarshalErr := yaml.Unmarshal(data, &kv); unmarshalErr != nil {
			return fmt.Errorf("parse locale %s: %w", path, unmarshalErr)
		}
		t.locales[lang] = kv
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Ensure default exists
	if _, ok := t.locales[defaultLang]; !ok {
		t.locales[defaultLang] = make(map[string]string)
	}

	return t, nil
}

// NewFallback creates a translator with no locales and a given default language.
func NewFallback(defaultLang string) *Translator {
	return &Translator{
		locales:     map[string]map[string]string{defaultLang: {}},
		defaultLang: defaultLang,
	}
}

// T returns translation for key with fallback to default and then the key itself.
func (t *Translator) T(lang, key string) string {
	if lang != "" {
		if val, ok := t.locales[lang][key]; ok {
			return val
		}
	}
	if val, ok := t.locales[t.defaultLang][key]; ok {
		return val
	}
	return key
}

// Available returns loaded language codes.
func (t *Translator) Available() []string {
	keys := make([]string, 0, len(t.locales))
	for k := range t.locales {
		keys = append(keys, k)
	}
	return keys
}
