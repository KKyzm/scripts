package cliptool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Command struct {
	Name  string `json:"name" yaml:"name"`
	Shell string `json:"shell" yaml:"shell"`
}

func LoadCommands() ([]Command, string, error) {
	for _, path := range configSearchPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, "", fmt.Errorf("read config %q: %w", path, err)
		}

		commands, err := parseCommands(path, data)
		if err != nil {
			return defaultCommands(), fmt.Sprintf("config warning: %s", err), nil
		}
		if len(commands) == 0 {
			return defaultCommands(), fmt.Sprintf("config warning: %s is empty, using built-ins", path), nil
		}
		return commands, "", nil
	}

	return defaultCommands(), "", nil
}

func configSearchPaths() []string {
	home := mustGetHomeDir()
	if home == "" {
		return nil
	}

	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	base := filepath.Join(configHome, "clip-tool")
	return []string{
		filepath.Join(base, "commands.yaml"),
		filepath.Join(base, "commands.yml"),
		filepath.Join(base, "commands.json"),
	}
}

func parseCommands(path string, data []byte) ([]Command, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, nil
	}

	ext := strings.ToLower(filepath.Ext(path))
	var commands []Command
	var err error
	if ext == ".json" {
		commands, err = parseCommandsJSON(data)
	} else {
		commands, err = parseCommandsYAML(data)
	}
	if err != nil {
		return nil, err
	}

	return normalizeCommands(commands), nil
}

func parseCommandsJSON(data []byte) ([]Command, error) {
	var direct map[string]string
	if err := json.Unmarshal(data, &direct); err == nil {
		return mapCommands(direct), nil
	}

	var nested struct {
		Commands map[string]string `json:"commands"`
		Items    []Command         `json:"items"`
	}
	if err := json.Unmarshal(data, &nested); err != nil {
		return nil, fmt.Errorf("parse JSON config: %w", err)
	}
	if len(nested.Commands) > 0 {
		return mapCommands(nested.Commands), nil
	}
	return nested.Items, nil
}

func parseCommandsYAML(data []byte) ([]Command, error) {
	var direct map[string]string
	if err := yaml.Unmarshal(data, &direct); err == nil && len(direct) > 0 {
		return mapCommands(direct), nil
	}

	var nested struct {
		Commands map[string]string `yaml:"commands"`
		Items    []Command         `yaml:"items"`
	}
	if err := yaml.Unmarshal(data, &nested); err != nil {
		return nil, fmt.Errorf("parse YAML config: %w", err)
	}
	if len(nested.Commands) > 0 {
		return mapCommands(nested.Commands), nil
	}
	return nested.Items, nil
}

func mapCommands(raw map[string]string) []Command {
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	commands := make([]Command, 0, len(keys))
	for _, key := range keys {
		commands = append(commands, Command{Name: key, Shell: raw[key]})
	}
	return commands
}

func normalizeCommands(commands []Command) []Command {
	result := make([]Command, 0, len(commands))
	for _, command := range commands {
		name := strings.TrimSpace(command.Name)
		shell := strings.TrimSpace(command.Shell)
		if name == "" || shell == "" {
			continue
		}
		result = append(result, Command{Name: name, Shell: shell})
	}
	return result
}

func defaultCommands() []Command {
	return []Command{
		{Name: "identity", Shell: "cat"},
		{Name: "collapse-blank-lines", Shell: "awk 'BEGIN{blank=0} /^$/ {if (!blank) print; blank=1; next} {blank=0; print}'"},
		{Name: "lower-case", Shell: "tr '[:upper:]' '[:lower:]'"},
		{Name: "remove-blank-lines", Shell: "awk 'NF'"},
		{Name: "remove-duplicate-lines", Shell: "awk '!seen[$0]++'"},
		{Name: "reverse-lines", Shell: "awk '{lines[NR]=$0} END{for(i=NR;i>=1;i--) print lines[i]}'"},
		{Name: "trim-trailing-space", Shell: "sed 's/[[:space:]]*$//'"},
		{Name: "trim-leading-space", Shell: "sed 's/^[[:space:]]*//'"},
		{Name: "trim-space", Shell: "sed 's/^[[:space:]]*//;s/[[:space:]]*$//'"},
		{Name: "sort-lines", Shell: "sort"},
		{Name: "sort-lines-ignore-case", Shell: "sort -f"},
		{Name: "number-lines", Shell: "awk '{print NR \"\\t\" $0}'"},
		{Name: "unique-lines", Shell: "awk '!seen[$0]++'"},
		{Name: "upper-case", Shell: "tr '[:lower:]' '[:upper:]'"},
	}
}
