package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Catalogs map[string]map[string]string

func LoadCatalogs(dir string) (Catalogs, error) {
	langs := []string{"en", "el"}
	out := Catalogs{}
	for _, lang := range langs {
		b, err := ioutil.ReadFile(filepath.Join(dir, lang+".json"))
		if err != nil {
			return nil, err
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, err
		}
		out[lang] = m
	}
	baseKeys := sortedKeys(out["en"])
	for _, lang := range langs[1:] {
		if got := strings.Join(sortedKeys(out[lang]), ","); got != strings.Join(baseKeys, ",") {
			return nil, fmt.Errorf("translation keys mismatch for %s", lang)
		}
	}
	return out, nil
}

func ResolveLanguage(explicit string, cookie string, acceptLanguage string) string {
	for _, choice := range []string{explicit, cookie} {
		if lang := supportedLanguage(choice); lang != "" {
			return lang
		}
	}
	selected, bestQuality := "en", 0.0
	for _, preference := range strings.Split(acceptLanguage, ",") {
		parts := strings.Split(preference, ";")
		lang := supportedLanguage(parts[0])
		if lang == "" {
			continue
		}
		quality := 1.0
		for _, parameter := range parts[1:] {
			pair := strings.SplitN(strings.TrimSpace(parameter), "=", 2)
			if len(pair) == 2 && strings.EqualFold(pair[0], "q") {
				value, err := strconv.ParseFloat(pair[1], 64)
				if err != nil || value < 0 || value > 1 {
					quality = 0
					break
				}
				quality = value
			}
		}
		if quality > bestQuality {
			selected, bestQuality = lang, quality
		}
	}
	return selected
}

func supportedLanguage(value string) string {
	base := strings.SplitN(strings.ToLower(strings.TrimSpace(value)), "-", 2)[0]
	if base == "en" || base == "el" {
		return base
	}
	return ""
}

func (c Catalogs) T(lang string, key string, data map[string]string) string {
	table, ok := c[lang]
	if !ok {
		table = c["en"]
	}
	value, ok := table[key]
	if !ok {
		panic("missing translation key: " + key)
	}
	for k, v := range data {
		value = strings.ReplaceAll(value, "{{"+k+"}}", v)
	}
	return value
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
