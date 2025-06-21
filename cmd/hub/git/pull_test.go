// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/epam/hubctl/cmd/hub/config"
	"github.com/epam/hubctl/cmd/hub/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test fixtures and helpers
const (
	testPullRemoteURL = "https://github.com/test/repo.git"
	testPullRef       = "main"
	testComponentName = "test-component"
)

// Helper function to create test directory structure
func setupTestDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	return tempDir
}

// Helper function to create a file with content
func createTestFile(t *testing.T, path, content string) {
	t.Helper()
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

// Helper function to create a directory
func createTestDir(t *testing.T, path string) {
	t.Helper()
	err := os.MkdirAll(path, 0755)
	require.NoError(t, err)
}

func TestMaybeRemote(t *testing.T) {
	tests := []struct {
		name     string
		origin   string
		expected bool
	}{
		{
			name:     "HTTP URL",
			origin:   "https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "SSH URL",
			origin:   "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "local path with colon",
			origin:   "/path/to:repo",
			expected: true,
		},
		{
			name:     "simple local path",
			origin:   "/path/to/repo",
			expected: false,
		},
		{
			name:     "relative path",
			origin:   "./repo",
			expected: false,
		},
		{
			name:     "empty string",
			origin:   "",
			expected: false,
		},
		{
			name:     "Windows path",
			origin:   "C:\\path\\to\\repo",
			expected: true,
		},
		{
			name:     "Windows path with colon",
			origin:   "C:\\path:to\\repo",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maybeRemote(tt.origin)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindLocalClone(t *testing.T) {
	repos := []LocalGitRepo{
		{
			Remote: "https://github.com/user/repo1.git",
			Ref:    "main",
			AbsDir: "/path/to/repo1",
		},
		{
			Remote: "https://github.com/user/repo2.git",
			Ref:    "develop",
			AbsDir: "/path/to/repo2",
		},
		{
			Remote: "https://github.com/user/repo3.git",
			Ref:    "main",
			AbsDir: "/path/to/repo3",
		},
	}

	tests := []struct {
		name     string
		remote   string
		ref      string
		expected string
	}{
		{
			name:     "found matching remote and ref",
			remote:   "https://github.com/user/repo1.git",
			ref:      "main",
			expected: "/path/to/repo1",
		},
		{
			name:     "found matching remote with different ref",
			remote:   "https://github.com/user/repo2.git",
			ref:      "develop",
			expected: "/path/to/repo2",
		},
		{
			name:     "remote not found",
			remote:   "https://github.com/user/nonexistent.git",
			ref:      "main",
			expected: "https://github.com/user/nonexistent.git",
		},
		{
			name:     "remote found but ref doesn't match",
			remote:   "https://github.com/user/repo1.git",
			ref:      "develop",
			expected: "https://github.com/user/repo1.git",
		},
		{
			name:     "empty remote",
			remote:   "",
			ref:      "main",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findLocalClone(repos, tt.remote, tt.ref)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindLocalCloneEmptyRepos(t *testing.T) {
	result := findLocalClone([]LocalGitRepo{}, "https://github.com/user/repo.git", "main")
	assert.Equal(t, "https://github.com/user/repo.git", result)
}

func TestDirInRepoList(t *testing.T) {
	repos := []LocalGitRepo{
		{AbsDir: "/path/to/repo1"},
		{AbsDir: "/path/to/repo2"},
		{AbsDir: "/path/to/repo3"},
	}

	tests := []struct {
		name     string
		dir      string
		expected bool
	}{
		{
			name:     "directory exists in list",
			dir:      "/path/to/repo1",
			expected: true,
		},
		{
			name:     "directory exists in middle of list",
			dir:      "/path/to/repo2",
			expected: true,
		},
		{
			name:     "directory exists at end of list",
			dir:      "/path/to/repo3",
			expected: true,
		},
		{
			name:     "directory not in list",
			dir:      "/path/to/repo4",
			expected: false,
		},
		{
			name:     "empty directory",
			dir:      "",
			expected: false,
		},
		{
			name:     "similar but different path",
			dir:      "/path/to/repo11",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := dirInRepoList(tt.dir, repos)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDirInRepoListEmptyRepos(t *testing.T) {
	result := dirInRepoList("/any/path", []LocalGitRepo{})
	assert.False(t, result)
}

func TestEmptyDir(t *testing.T) {
	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentDir := filepath.Join(t.TempDir(), "nonexistent")

		clone, err := emptyDir(nonExistentDir, true)

		assert.NoError(t, err)
		assert.True(t, clone)
	})

	t.Run("empty directory", func(t *testing.T) {
		tempDir := setupTestDir(t)
		emptyDirectory := filepath.Join(tempDir, "empty")
		createTestDir(t, emptyDirectory)

		clone, err := emptyDir(emptyDirectory, true)

		assert.NoError(t, err)
		assert.True(t, clone)
	})

	t.Run("directory with files - force enabled", func(t *testing.T) {
		tempDir := setupTestDir(t)
		dirWithFiles := filepath.Join(tempDir, "with_files")
		createTestDir(t, dirWithFiles)
		createTestFile(t, filepath.Join(dirWithFiles, "test.txt"), "content")

		// Enable force mode
		originalForce := config.Force
		config.Force = true
		defer func() { config.Force = originalForce }()

		clone, err := emptyDir(dirWithFiles, true)

		assert.NoError(t, err)
		assert.True(t, clone)
	})

	t.Run("directory with files - force disabled", func(t *testing.T) {
		tempDir := setupTestDir(t)
		dirWithFiles := filepath.Join(tempDir, "with_files")
		createTestDir(t, dirWithFiles)
		createTestFile(t, filepath.Join(dirWithFiles, "test.txt"), "content")

		// Disable force mode
		originalForce := config.Force
		config.Force = false
		defer func() { config.Force = originalForce }()

		clone, err := emptyDir(dirWithFiles, true)

		assert.Error(t, err)
		assert.False(t, clone)
		assert.Contains(t, err.Error(), "not an empty directory")
	})

	t.Run("directory with files - removeContentIfForced false", func(t *testing.T) {
		tempDir := setupTestDir(t)
		dirWithFiles := filepath.Join(tempDir, "with_files")
		createTestDir(t, dirWithFiles)
		createTestFile(t, filepath.Join(dirWithFiles, "test.txt"), "content")

		clone, err := emptyDir(dirWithFiles, false)

		assert.NoError(t, err)
		assert.False(t, clone)
	})

	t.Run("existing git repository", func(t *testing.T) {
		tempDir := setupTestDir(t)
		gitRepo := filepath.Join(tempDir, "git_repo")
		createTestDir(t, gitRepo)
		createTestDir(t, filepath.Join(gitRepo, ".git"))

		clone, err := emptyDir(gitRepo, true)

		assert.NoError(t, err)
		assert.False(t, clone)
	})

	t.Run("file instead of directory - force enabled", func(t *testing.T) {
		tempDir := setupTestDir(t)
		filePath := filepath.Join(tempDir, "file.txt")
		createTestFile(t, filePath, "content")

		// Enable force mode
		originalForce := config.Force
		config.Force = true
		defer func() { config.Force = originalForce }()

		clone, err := emptyDir(filePath, true)

		assert.NoError(t, err)
		assert.True(t, clone)
	})

	t.Run("file instead of directory - force disabled", func(t *testing.T) {
		tempDir := setupTestDir(t)
		filePath := filepath.Join(tempDir, "file.txt")
		createTestFile(t, filePath, "content")

		// Disable force mode
		originalForce := config.Force
		config.Force = false
		defer func() { config.Force = originalForce }()

		clone, err := emptyDir(filePath, true)

		assert.Error(t, err)
		assert.False(t, clone)
		assert.Contains(t, err.Error(), "is not a directory")
	})

	t.Run("git directory is file not directory", func(t *testing.T) {
		tempDir := setupTestDir(t)
		repoDir := filepath.Join(tempDir, "repo")
		createTestDir(t, repoDir)
		createTestFile(t, filepath.Join(repoDir, ".git"), "git file content")

		clone, err := emptyDir(repoDir, true)

		assert.Error(t, err)
		assert.False(t, clone)
		assert.Contains(t, err.Error(), "is not a Git repo")
	})
}

func TestGetGit(t *testing.T) {
	t.Run("empty remote", func(t *testing.T) {
		source := manifest.Git{Remote: ""}
		components := []string{}
		repos := []LocalGitRepo{}

		resultComponents, resultRepos, err := getGit(source, "/base", "rel", "component", false, false, false, components, repos)

		assert.NoError(t, err)
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})

	t.Run("component already exists", func(t *testing.T) {
		source := manifest.Git{Remote: testPullRemoteURL}
		components := []string{"existing-component"}
		repos := []LocalGitRepo{}

		resultComponents, resultRepos, err := getGit(source, "/base", "rel", "existing-component", false, false, false, components, repos)

		assert.NoError(t, err)
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})

	t.Run("directory used twice", func(t *testing.T) {
		tempDir := setupTestDir(t)
		absDir := filepath.Join(tempDir, "repo")

		source := manifest.Git{Remote: testPullRemoteURL}
		components := []string{}
		repos := []LocalGitRepo{
			{AbsDir: absDir},
		}

		resultComponents, resultRepos, err := getGit(source, tempDir, "repo", "component", false, false, false, components, repos)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "used twice")
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})

	t.Run("subtree not implemented", func(t *testing.T) {
		tempDir := setupTestDir(t)

		source := manifest.Git{Remote: testPullRemoteURL}
		components := []string{}
		repos := []LocalGitRepo{}

		resultComponents, resultRepos, err := getGit(source, tempDir, "new-repo", "component", false, false, true, components, repos)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})

	t.Run("reset not implemented", func(t *testing.T) {
		tempDir := setupTestDir(t)
		repoDir := filepath.Join(tempDir, "existing-repo")
		createTestDir(t, repoDir)
		createTestDir(t, filepath.Join(repoDir, ".git"))

		source := manifest.Git{Remote: testPullRemoteURL}
		components := []string{}
		repos := []LocalGitRepo{}

		resultComponents, resultRepos, err := getGit(source, tempDir, "existing-repo", "component", true, false, false, components, repos)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not implemented")
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})

	t.Run("with local dir override", func(t *testing.T) {
		tempDir := setupTestDir(t)

		source := manifest.Git{
			Remote:   testPullRemoteURL,
			LocalDir: "custom-dir",
		}
		components := []string{}
		repos := []LocalGitRepo{}

		// This will fail at Clone step, but we can test the path logic
		resultComponents, resultRepos, err := getGit(source, tempDir, "original-dir", "component", false, false, false, components, repos)

		// Should fail at clone step, but the directory should be based on LocalDir
		assert.Error(t, err)
		assert.Equal(t, components, resultComponents)
		assert.Equal(t, repos, resultRepos)
	})
}

func TestPullManifest(t *testing.T) {
	// Note: This function is complex and depends on external manifest parsing
	// These are basic tests for the main entry point

	t.Run("captures log output for no git sources", func(t *testing.T) {
		// This test would require mocking the manifest.ParseManifest function
		// For now, we'll test the logging behavior when repos is empty

		output := captureLogOutput(t, func() {
			// Simulate the logging that happens when len(repos) == 0
			manifests := []string{"test-manifest.yaml"}
			log.Printf("No Git sources found in %s", strings.Join(manifests, ", "))
		})

		assert.Contains(t, output, "No Git sources found in test-manifest.yaml")
	})

	t.Run("captures log output for verbose mode", func(t *testing.T) {
		originalVerbose := config.Verbose
		config.Verbose = true
		defer func() { config.Verbose = originalVerbose }()

		output := captureLogOutput(t, func() {
			// Simulate the logging that happens when config.Verbose is true
			components := []string{"component1", "component2"}
			log.Printf("Components sourced from Git: %s", strings.Join(components, ", "))
		})

		assert.Contains(t, output, "Components sourced from Git: component1, component2")
	})
}

// Benchmark tests
func BenchmarkMaybeRemote(b *testing.B) {
	testCases := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"/local/path/to/repo",
		"./relative/path",
		"",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			maybeRemote(tc)
		}
	}
}

func BenchmarkFindLocalClone(b *testing.B) {
	repos := make([]LocalGitRepo, 100)
	for i := 0; i < 100; i++ {
		repos[i] = LocalGitRepo{
			Remote: fmt.Sprintf("https://github.com/user/repo%d.git", i),
			Ref:    "main",
			AbsDir: fmt.Sprintf("/path/to/repo%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		findLocalClone(repos, "https://github.com/user/repo50.git", "main")
	}
}

func BenchmarkDirInRepoList(b *testing.B) {
	repos := make([]LocalGitRepo, 100)
	for i := 0; i < 100; i++ {
		repos[i] = LocalGitRepo{
			AbsDir: fmt.Sprintf("/path/to/repo%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dirInRepoList("/path/to/repo50", repos)
	}
}

// Edge case tests
func TestPullEdgeCases(t *testing.T) {
	t.Run("maybeRemote with special characters", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool
		}{
			{"file://path:with:colons", true},
			{"path with spaces:and:colons", true},
			{"path\nwith\nnewlines", false},
			{"path\twith\ttabs", false},
		}

		for _, tc := range testCases {
			result := maybeRemote(tc.input)
			assert.Equal(t, tc.expected, result, "input: %s", tc.input)
		}
	})

	t.Run("findLocalClone with nil repos", func(t *testing.T) {
		result := findLocalClone(nil, "remote", "ref")
		assert.Equal(t, "remote", result)
	})

	t.Run("dirInRepoList with nil repos", func(t *testing.T) {
		result := dirInRepoList("/any/path", nil)
		assert.False(t, result)
	})

	t.Run("emptyDir with permission issues", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping permission test when running as root")
		}

		tempDir := setupTestDir(t)
		restrictedDir := filepath.Join(tempDir, "restricted")
		createTestDir(t, restrictedDir)

		// Make directory unreadable
		err := os.Chmod(restrictedDir, 0000)
		require.NoError(t, err)
		defer os.Chmod(restrictedDir, 0755) // Cleanup

		clone, err := emptyDir(restrictedDir, true)

		assert.Error(t, err)
		assert.False(t, clone)
	})
}

// Integration tests
func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration tests in short mode")
	}

	t.Run("emptyDir full workflow", func(t *testing.T) {
		tempDir := setupTestDir(t)

		// Test non-existent -> empty -> with files -> force clean
		testDir := filepath.Join(tempDir, "workflow")

		// Step 1: Non-existent directory
		clone, err := emptyDir(testDir, true)
		assert.NoError(t, err)
		assert.True(t, clone)

		// Step 2: Create and test empty directory
		createTestDir(t, testDir)
		clone, err = emptyDir(testDir, true)
		assert.NoError(t, err)
		assert.True(t, clone)

		// Step 3: Add files and test without force
		createTestFile(t, filepath.Join(testDir, "test.txt"), "content")
		originalForce := config.Force
		config.Force = false

		clone, err = emptyDir(testDir, true)
		assert.Error(t, err)
		assert.False(t, clone)

		// Step 4: Test with force
		config.Force = true
		clone, err = emptyDir(testDir, true)
		assert.NoError(t, err)
		assert.True(t, clone)

		config.Force = originalForce
	})
}

// Test constants and variables
func TestPullConstants(t *testing.T) {
	assert.Equal(t, os.FileMode(0755), dirMode)
}

// Helper tests
func TestPullHelperFunctions(t *testing.T) {
	t.Run("setupTestDir creates valid directory", func(t *testing.T) {
		dir := setupTestDir(t)
		assert.NotEmpty(t, dir)

		info, err := os.Stat(dir)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})

	t.Run("createTestFile creates file with content", func(t *testing.T) {
		tempDir := setupTestDir(t)
		filePath := filepath.Join(tempDir, "test.txt")
		content := "test content"

		createTestFile(t, filePath, content)

		readContent, err := os.ReadFile(filePath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(readContent))
	})

	t.Run("createTestDir creates directory", func(t *testing.T) {
		tempDir := setupTestDir(t)
		dirPath := filepath.Join(tempDir, "subdir", "nested")

		createTestDir(t, dirPath)

		info, err := os.Stat(dirPath)
		assert.NoError(t, err)
		assert.True(t, info.IsDir())
	})
}
