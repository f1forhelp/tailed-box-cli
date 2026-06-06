package ui

import "strings"

const fallbackGroup = "System"

func actionGroup(action Action) string {
	group, _, ok := strings.Cut(action.Title, ":")
	if !ok {
		return fallbackGroup
	}
	group = strings.TrimSpace(group)
	if group == "" {
		return fallbackGroup
	}
	return group
}

func actionName(action Action) string {
	_, name, ok := strings.Cut(action.Title, ":")
	if !ok {
		return strings.TrimSpace(action.Title)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return strings.TrimSpace(action.Title)
	}
	return name
}

func actionGroups(actions []Action) []string {
	seen := make(map[string]bool)
	var groups []string
	for _, action := range actions {
		group := actionGroup(action)
		if seen[group] {
			continue
		}
		seen[group] = true
		groups = append(groups, group)
	}
	return groups
}

func actionIndexesForGroup(actions []Action, group string) []int {
	var indexes []int
	for i, action := range actions {
		if actionGroup(action) == group {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func firstActionIndexForGroup(actions []Action, group string) int {
	for i, action := range actions {
		if actionGroup(action) == group {
			return i
		}
	}
	return 0
}

func selectedActionIndex(actions []Action, cursor int) int {
	if len(actions) == 0 {
		return 0
	}
	if cursor < 0 {
		return 0
	}
	if cursor >= len(actions) {
		return len(actions) - 1
	}
	return cursor
}

func selectedGroup(actions []Action, cursor int) string {
	if len(actions) == 0 {
		return fallbackGroup
	}
	return actionGroup(actions[selectedActionIndex(actions, cursor)])
}
