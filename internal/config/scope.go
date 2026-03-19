package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Scope defines the configuration scope type
type Scope string

const (
	// ScopeGlobal is the user-wide configuration (~/.claude)
	ScopeGlobal Scope = "global"
	// ScopeProject is the project-specific configuration (./.claude)
	ScopeProject Scope = "project"
	// ScopeLocal is the local configuration (./.claude/settings.local.json)
	ScopeLocal Scope = "local"
)

// ScopeManager handles global vs project scope logic
type ScopeManager struct {
	globalDir  string
	projectDir string
	cwd        string
}

// NewScopeManager creates a new ScopeManager
func NewScopeManager() (*ScopeManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	return &ScopeManager{
		globalDir:  filepath.Join(home, ".claude"),
		projectDir: filepath.Join(cwd, ".claude"),
		cwd:        cwd,
	}, nil
}

// GetSettingsPath returns the path to settings.json for a given scope and tool
func (sm *ScopeManager) GetSettingsPath(scope Scope, tool string) string {
	switch scope {
	case ScopeGlobal:
		return filepath.Join(sm.globalDir, "settings.json")
	case ScopeProject:
		return filepath.Join(sm.projectDir, "settings.json")
	case ScopeLocal:
		return filepath.Join(sm.projectDir, "settings.local.json")
	default:
		return ""
	}
}

// EnsureDir ensures the directory for a scope exists
func (sm *ScopeManager) EnsureDir(scope Scope) error {
	var dir string
	switch scope {
	case ScopeGlobal:
		dir = sm.globalDir
	case ScopeProject, ScopeLocal:
		dir = sm.projectDir
	default:
		return fmt.Errorf("invalid scope: %s", scope)
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}
	return nil
}

// IsProjectScopeAvailable checks if we're in a git repository
func (sm *ScopeManager) IsProjectScopeAvailable() bool {
	// Check for .git directory
	gitDir := filepath.Join(sm.cwd, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}

	// Also check if .claude directory exists (project already initialized)
	if _, err := os.Stat(sm.projectDir); err == nil {
		return true
	}

	return false
}

// GetCurrentScope returns the best scope for the current context
// If in a git repo, suggests project scope, otherwise global
func (sm *ScopeManager) GetCurrentScope() Scope {
	if sm.IsProjectScopeAvailable() {
		return ScopeProject
	}
	return ScopeGlobal
}

// LoadSettings loads Claude settings for a given scope
func (sm *ScopeManager) LoadSettings(scope Scope) (*ClaudeSettings, error) {
	path := sm.GetSettingsPath(scope, "claude-code")

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &ClaudeSettings{
			Env: make(map[string]string),
		}, nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}

	// Parse JSON
	var settings ClaudeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse settings: %w", err)
	}

	// Ensure Env map is initialized
	if settings.Env == nil {
		settings.Env = make(map[string]string)
	}

	return &settings, nil
}

// SaveSettings saves Claude settings for a given scope
// Preserves existing settings by merging with what's already in the file
func (sm *ScopeManager) SaveSettings(scope Scope, settings *ClaudeSettings) error {
	// Ensure directory exists
	if err := sm.EnsureDir(scope); err != nil {
		return err
	}

	path := sm.GetSettingsPath(scope, "claude-code")

	// Load existing settings to preserve other fields
	existingData := make(map[string]interface{})
	if data, err := os.ReadFile(path); err == nil {
		// File exists, parse it
		if err := json.Unmarshal(data, &existingData); err != nil {
			// If parsing fails, start fresh
			existingData = make(map[string]interface{})
		}
	}

	// Update with new env settings
	if settings.Env != nil {
		existingData["env"] = settings.Env
	}

	// Update other fields if they're set
	if settings.APIKeyHelper != "" {
		existingData["apiKeyHelper"] = settings.APIKeyHelper
	} else {
		// Remove apiKeyHelper if it was previously set but now cleared
		delete(existingData, "apiKeyHelper")
	}

	if settings.AWSAuthRefresh != "" {
		existingData["awsAuthRefresh"] = settings.AWSAuthRefresh
	}
	if settings.AWSCredentialExport != "" {
		existingData["awsCredentialExport"] = settings.AWSCredentialExport
	}
	if settings.ModelOverrides != nil {
		existingData["modelOverrides"] = settings.ModelOverrides
	}
	if settings.AvailableModels != nil {
		existingData["availableModels"] = settings.AvailableModels
	}

	// Marshal to JSON with proper indentation
	data, err := json.MarshalIndent(existingData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write settings: %w", err)
	}

	return nil
}

// RemoveEnvVar removes an environment variable from settings
func (sm *ScopeManager) RemoveEnvVar(scope Scope, key string) error {
	settings, err := sm.LoadSettings(scope)
	if err != nil {
		return err
	}

	delete(settings.Env, key)

	return sm.SaveSettings(scope, settings)
}

// HasSettings checks if settings exist for a scope
func (sm *ScopeManager) HasSettings(scope Scope) bool {
	path := sm.GetSettingsPath(scope, "claude-code")
	_, err := os.Stat(path)
	return err == nil
}

// GetGlobalDir returns the global configuration directory
func (sm *ScopeManager) GetGlobalDir() string {
	return sm.globalDir
}

// GetProjectDir returns the project configuration directory
func (sm *ScopeManager) GetProjectDir() string {
	return sm.projectDir
}

// DetectConflicts detects conflicts between global and project settings
func (sm *ScopeManager) DetectConflicts() ([]Conflict, error) {
	var conflicts []Conflict

	// Load both scopes
	globalSettings, err := sm.LoadSettings(ScopeGlobal)
	if err != nil {
		return nil, err
	}

	if !sm.HasSettings(ScopeProject) {
		// No project settings, check for potential conflicts with global
		if apiKey, ok := globalSettings.Env["ANTHROPIC_API_KEY"]; ok && apiKey != "" {
			// Check if there's a project-level provider configured
			conflicts = append(conflicts, Conflict{
				Type:         "api_key_conflict",
				Description:  "ANTHROPIC_API_KEY found in global settings",
				Severity:     "warning",
				SuggestedFix: "Remove ANTHROPIC_API_KEY from global settings when using other providers",
				Scope:        ScopeGlobal,
			})
		}
		return conflicts, nil
	}

	projectSettings, err := sm.LoadSettings(ScopeProject)
	if err != nil {
		return nil, err
	}

	// Check for overlapping env vars
	for key, globalVal := range globalSettings.Env {
		if globalVal == "" {
			continue
		}
		if projectVal, ok := projectSettings.Env[key]; ok {
			if projectVal != "" && projectVal != globalVal {
				conflicts = append(conflicts, Conflict{
					Type:         "scope_mismatch",
					Description:  fmt.Sprintf("%s differs between global and project scope", key),
					Severity:     "info",
					SuggestedFix: "Project settings will override global",
					Scope:        ScopeProject,
				})
			}
		}
	}

	return conflicts, nil
}

// Conflict represents a configuration conflict
type Conflict struct {
	Type         string
	Description  string
	Severity     string
	SuggestedFix string
	Scope        Scope
}

// IsValidScope checks if a string is a valid scope
func IsValidScope(s string) bool {
	switch Scope(s) {
	case ScopeGlobal, ScopeProject, ScopeLocal:
		return true
	default:
		return false
	}
}

// String returns the string representation of a scope
func (s Scope) String() string {
	return string(s)
}
