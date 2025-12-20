package plugins

import (
	apix "github.com/Infra-Forge/infra-apix"
)

// AutoTagPlugin automatically adds tags to routes based on path patterns.
// This plugin demonstrates the OnRouteRegister hook.
//
// Example usage:
//
//	plugin := &plugins.AutoTagPlugin{
//		Rules: map[string]string{
//			"/api/users":   "users",
//			"/api/posts":   "posts",
//			"/api/admin":   "admin",
//		},
//	}
//	apix.RegisterPlugin(plugin)
type AutoTagPlugin struct {
	apix.BasePlugin
	// Rules maps path prefixes to tags
	Rules map[string]string
}

// NewAutoTagPlugin creates a new AutoTagPlugin with the given rules.
func NewAutoTagPlugin(rules map[string]string) *AutoTagPlugin {
	return &AutoTagPlugin{
		BasePlugin: apix.BasePlugin{PluginName: "auto-tag"},
		Rules:      rules,
	}
}

// OnRouteRegister adds tags to routes based on path prefix matching.
func (p *AutoTagPlugin) OnRouteRegister(ref *apix.RouteRef) error {
	for prefix, tag := range p.Rules {
		if len(ref.Path) >= len(prefix) && ref.Path[:len(prefix)] == prefix {
			// Check if tag already exists
			exists := false
			for _, existingTag := range ref.Tags {
				if existingTag == tag {
					exists = true
					break
				}
			}
			if !exists {
				ref.Tags = append(ref.Tags, tag)
			}
		}
	}
	return nil
}

