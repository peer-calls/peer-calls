package logger

import (
	"strings"
)

type wildcardNode struct {
	// Level is the logging level of this subsystem.
	level Level

	// Name is the latest part of the namespace, after the last colon.
	name string

	children map[string]*wildcardNode
}

func newWildcardNode(config ConfigMap) Config {
	if config == nil {
		return nil
	}

	root := &wildcardNode{}

	for k, v := range config {
		root.add(k, v)
	}

	return root
}

var _ Config = &wildcardNode{}

func (n *wildcardNode) add(namespace string, level Level) {
	if namespace == "" {
		n.level = level

		return
	}

	names := strings.Split(namespace, ":")

	parent := n

	for _, name := range names {
		child, ok := parent.children[name]
		if !ok {
			child = &wildcardNode{
				level: LevelUnknown,
				name:  name,
			}

			if parent.children == nil {
				parent.children = map[string]*wildcardNode{
					name: child,
				}
			} else {
				parent.children[name] = child
			}
		}

		parent = child
	}

	parent.level = level
}

func (n *wildcardNode) levelForNamespace(names []string) (Level, bool) {
	if len(names) == 0 {
		node := n

		if node.level == LevelUnknown {
			// Final check for wildcard on the right side.
			if child, ok := n.children["**"]; ok {
				node = child
			}
		}

		return node.level, node.level != LevelUnknown
	}

	parent := n

	name := names[0]

	if child, ok := parent.children[name]; ok {
		if level, ok := child.levelForNamespace(names[1:]); ok {
			return level, true
		}
	}

	// Handle special case for double-wildcard. This matches any number of
	// namespace sections in between.
	if n.name == "**" {
		for i := 0; i < len(names); i++ {
			if child, ok := parent.children[names[i]]; ok {
				if level, ok := child.levelForNamespace(names[i+1:]); ok {
					return level, true
				}
			}
		}

		// Special case when ** is at the end.
		if n.level != LevelUnknown {
			return n.level, true
		}
	}

	if child, ok := parent.children["*"]; ok {
		if level, ok := child.levelForNamespace(names[1:]); ok {
			return level, true
		}
	}

	if child, ok := parent.children["**"]; ok {
		// Send all names to take into account for the scenario where there is no prefix matched by **.
		if level, ok := child.levelForNamespace(names); ok {
			return level, true
		}
	}

	return LevelDisabled, false
}

// LevelForNamespace implements Config.
func (n *wildcardNode) LevelForNamespace(namespace string) Level {
	if namespace == "" {
		return n.level
	}

	split := strings.Split(namespace, ":")

	if level, ok := n.levelForNamespace(split); ok {
		return level
	}

	return n.level
}
