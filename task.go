package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const taskInstanceEnvVar = "SALMODULE_TASK_INSTANCE"

// config is the task instance SAL serializes for a JsonLdHarvester node.
type config struct {
	IncludeSitemaps []string
	ExcludeSitemaps []string
}

// loadConfig reads the task instance from the environment variable sal
// invokes the container with. An unset or empty variable is a harvester
// with no filtering: every sitemap in the catalog is harvested.
func loadConfig() (config, error) {
	raw := os.Getenv(taskInstanceEnvVar)
	if raw == "" {
		return config{}, nil
	}

	var instance map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &instance); err != nil {
		return config{}, fmt.Errorf("parsing %s: %w", taskInstanceEnvVar, err)
	}

	var cfg config
	var err error
	if raw, ok := instance["includeSitemaps"]; ok {
		cfg.IncludeSitemaps, err = parseStringList(raw)
		if err != nil {
			return config{}, fmt.Errorf("parsing includeSitemaps: %w", err)
		}
	}
	if raw, ok := instance["excludeSitemaps"]; ok {
		cfg.ExcludeSitemaps, err = parseStringList(raw)
		if err != nil {
			return config{}, fmt.Errorf("parsing excludeSitemaps: %w", err)
		}
	}
	return cfg, nil
}

// parseStringList accepts the shapes sal uses to serialize a repeatable
// datatype property: a bare string, a {"@value": ...} literal, or an array
// of either.
func parseStringList(raw json.RawMessage) ([]string, error) {
	var asArray []json.RawMessage
	if err := json.Unmarshal(raw, &asArray); err == nil {
		values := make([]string, 0, len(asArray))
		for _, item := range asArray {
			v, err := parseStringLiteral(item)
			if err != nil {
				return nil, err
			}
			values = append(values, v)
		}
		return values, nil
	}

	v, err := parseStringLiteral(raw)
	if err != nil {
		return nil, err
	}
	return []string{v}, nil
}

func parseStringLiteral(raw json.RawMessage) (string, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s, nil
	}

	var literal struct {
		Value string `json:"@value"`
	}
	if err := json.Unmarshal(raw, &literal); err == nil && literal.Value != "" {
		return literal.Value, nil
	}

	return "", fmt.Errorf("expected a string or {\"@value\": ...} literal, got %s", string(raw))
}

func printErrorNode(w io.Writer, err error) {
	node := map[string]any{
		"@type":        "salmodule:Error",
		"rdfs:comment": err.Error(),
	}
	enc, mErr := json.Marshal(node)
	if mErr != nil {
		return
	}
	fmt.Fprintln(w, string(enc))
}
