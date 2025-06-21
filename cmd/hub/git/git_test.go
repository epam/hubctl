// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testRemoteURL = "https://github.com/test/repo.git"
	testRef       = "main"
	testTag       = "v1.0.0"
)

func TestCloneErrMsgFormat(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		ref       string
		dir       string
		err       error
		expected  string
	}{
		{
			name:      "formats error message correctly",
			remoteURL: testRemoteURL,
			ref:       testRef,
			dir:       "/tmp/test",
			err:       fmt.Errorf("connection failed"),
			expected:  "unable to clone git repo https://github.com/test/repo.git at `main` into `/tmp/test`: connection failed",
		},
		{
			name:      "handles empty ref",
			remoteURL: testRemoteURL,
			ref:       "",
			dir:       "/tmp/test",
			err:       fmt.Errorf("auth failed"),
			expected:  "unable to clone git repo https://github.com/test/repo.git at `` into `/tmp/test`: auth failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cloneErrMsgFormat(tt.remoteURL, tt.ref, tt.dir, tt.err)
			assert.Equal(t, tt.expected, result.Error())
		})
	}
}

func TestPullErrMsgFormat(t *testing.T) {
	tests := []struct {
		name     string
		dir      string
		err      error
		expected string
	}{
		{
			name:     "formats pull error message correctly",
			dir:      "/tmp/test",
			err:      fmt.Errorf("not a git repository"),
			expected: "unable to pull git repo in `/tmp/test` directory: not a git repository",
		},
		{
			name:     "handles empty directory",
			dir:      "",
			err:      fmt.Errorf("directory not found"),
			expected: "unable to pull git repo in `` directory: directory not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pullErrMsgFormat(tt.dir, tt.err)
			assert.Equal(t, tt.expected, result.Error())
		})
	}
}

func TestClone(t *testing.T) {
	t.Run("clone with invalid remote URL", func(t *testing.T) {
		tempDir := t.TempDir()

		err := Clone("invalid-url", testRef, tempDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to clone git repo")
		assert.Contains(t, err.Error(), "invalid-url")
	})

	t.Run("clone with non-existent directory parent", func(t *testing.T) {
		invalidDir := "/non/existent/path/repo"

		err := Clone(testRemoteURL, testRef, invalidDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to clone git repo")
	})
}

func TestPull(t *testing.T) {
	t.Run("pull from non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		err := Pull(testRef, tempDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to pull git repo")
		assert.Contains(t, err.Error(), "repository does not exist")
	})

	t.Run("pull from non-existent directory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"

		err := Pull(testRef, nonExistentDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to pull git repo")
	})
}

func TestFindRemoteReference(t *testing.T) {
	tests := []struct {
		name        string
		remoteURL   string
		targetRef   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "invalid remote URL",
			remoteURL:   "invalid-url",
			targetRef:   testRef,
			expectError: true,
			errorMsg:    "unable to get ref list",
		},
		{
			name:        "empty target ref",
			remoteURL:   testRemoteURL,
			targetRef:   "",
			expectError: true,
			errorMsg:    "unable to get ref list", // Will fail at remote list stage
		},
		{
			name:        "non-existent ref",
			remoteURL:   testRemoteURL,
			targetRef:   "non-existent-branch",
			expectError: true,
			errorMsg:    "unable to get ref list", // Will fail at remote list stage for invalid URL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ref, err := findRemoteReference(tt.remoteURL, tt.targetRef)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ref)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ref)
			}
		})
	}
}

func TestCloneIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("clone to existing non-empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a file in the directory
		testFile := filepath.Join(tempDir, "existing-file.txt")
		err := os.WriteFile(testFile, []byte("test content"), 0644)
		require.NoError(t, err)

		err = Clone(testRemoteURL, testRef, tempDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to clone git repo")
	})
}

func TestPullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("pull with empty target ref", func(t *testing.T) {
		tempDir := t.TempDir()

		// Initialize a git repository
		repo, err := git.PlainInit(tempDir, false)
		require.NoError(t, err)

		// Add a remote
		_, err = repo.CreateRemote(&config.RemoteConfig{
			Name: remoteName,
			URLs: []string{testRemoteURL},
		})
		require.NoError(t, err)

		// Create initial commit to have a HEAD
		worktree, err := repo.Worktree()
		require.NoError(t, err)

		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test"), 0644)
		require.NoError(t, err)

		_, err = worktree.Add("test.txt")
		require.NoError(t, err)

		_, err = worktree.Commit("initial commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
			},
		})
		require.NoError(t, err)

		err = Pull("", tempDir)

		// This will fail because we can't actually pull from the remote,
		// but it tests the code path for empty targetRef
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkCloneErrMsgFormat(b *testing.B) {
	err := fmt.Errorf("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cloneErrMsgFormat(testRemoteURL, testRef, "/tmp/test", err)
	}
}

func BenchmarkPullErrMsgFormat(b *testing.B) {
	err := fmt.Errorf("test error")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pullErrMsgFormat("/tmp/test", err)
	}
}

// Table-driven test for reference prefixes
func TestRefPrefixes(t *testing.T) {
	tests := []struct {
		name     string
		ref      string
		expected []string
	}{
		{
			name: "verify ref prefixes",
			ref:  "test",
			expected: []string{
				"refs/heads/test",
				"refs/tags/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var results []string
			for _, format := range refFindOrder {
				results = append(results, fmt.Sprintf(format, tt.ref))
			}

			assert.Equal(t, tt.expected, results)
		})
	}
}

// Test constants
func TestConstants(t *testing.T) {
	assert.Equal(t, "origin", remoteName)
	assert.Equal(t, "refs/", refPrefix)
	assert.Equal(t, "refs/heads/", refHeadPrefix)
	assert.Equal(t, "refs/tags/", refTagPrefix)

	expectedPrefixes := []string{"refs/heads/", "refs/tags/"}
	assert.Equal(t, expectedPrefixes, refPrefixes)

	expectedFindOrder := []string{"refs/heads/%s", "refs/tags/%s"}
	assert.Equal(t, expectedFindOrder, refFindOrder)
}

// Helper function tests (if util package functions were to be tested)
func TestHelperFunctions(t *testing.T) {
	// Note: These tests assume the util.ContainsPrefix function exists
	// You may need to adjust based on the actual implementation

	t.Run("ref prefix validation", func(t *testing.T) {
		testCases := []struct {
			name     string
			ref      string
			expected bool
		}{
			{
				name:     "valid branch ref",
				ref:      "refs/heads/main",
				expected: true,
			},
			{
				name:     "valid tag ref",
				ref:      "refs/tags/v1.0.0",
				expected: true,
			},
			{
				name:     "invalid ref",
				ref:      "invalid/ref",
				expected: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// This would test the util.ContainsPrefix function
				// Adjust based on actual implementation
				hasValidPrefix := false
				for _, prefix := range refPrefixes {
					if len(tc.ref) >= len(prefix) && tc.ref[:len(prefix)] == prefix {
						hasValidPrefix = true
						break
					}
				}
				assert.Equal(t, tc.expected, hasValidPrefix)
			})
		}
	})
}
