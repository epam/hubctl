// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	goGit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test constants
const (
	testAuthorName  = "Test Author"
	testAuthorEmail = "test@example.com"
	testCommitMsg   = "test commit"
	testFileName    = "test.txt"
	testFileContent = "test content"
)

// setupTestRepo creates a test git repository with initial commit
func setupTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	// Initialize git repository
	repo, err := goGit.PlainInit(tempDir, false)
	require.NoError(t, err, "failed to initialize git repository")

	// Create a test file
	testFile := filepath.Join(tempDir, testFileName)
	err = os.WriteFile(testFile, []byte(testFileContent), 0644)
	require.NoError(t, err, "failed to create test file")

	// Add and commit the file
	worktree, err := repo.Worktree()
	require.NoError(t, err, "failed to get worktree")

	_, err = worktree.Add(testFileName)
	require.NoError(t, err, "failed to add file to git")

	_, err = worktree.Commit(testCommitMsg, &goGit.CommitOptions{
		Author: &object.Signature{
			Name:  testAuthorName,
			Email: testAuthorEmail,
			When:  time.Now(),
		},
	})
	require.NoError(t, err, "failed to commit file")

	return tempDir
}

// setupEmptyTestRepo creates an empty git repository without commits
func setupEmptyTestRepo(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()

	_, err := goGit.PlainInit(tempDir, false)
	require.NoError(t, err, "failed to initialize empty git repository")

	return tempDir
}

func TestHeadInfo(t *testing.T) {
	t.Run("successful head info retrieval", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		name, rev, err := HeadInfo(repoDir)

		assert.NoError(t, err)
		assert.NotEmpty(t, name, "name should not be empty")
		assert.NotEmpty(t, rev, "revision should not be empty")
		assert.True(t, strings.HasPrefix(name, "refs/heads/"), "name should be a branch reference")
		assert.Len(t, rev, 40, "revision should be a 40-character SHA-1 hash")
		assert.NotEqual(t, "(unknown)", name, "name should not be unknown")
		assert.NotEqual(t, "(unknown)", rev, "revision should not be unknown")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"

		name, rev, err := HeadInfo(nonExistentDir)

		assert.Error(t, err)
		assert.Empty(t, name)
		assert.Empty(t, rev)
		assert.Contains(t, err.Error(), "is not valid git repository")
		assert.Contains(t, err.Error(), nonExistentDir)
	})

	t.Run("non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		name, rev, err := HeadInfo(tempDir)

		assert.Error(t, err)
		assert.Empty(t, name)
		assert.Empty(t, rev)
		assert.Contains(t, err.Error(), "is not valid git repository")
		assert.Contains(t, err.Error(), tempDir)
	})

	t.Run("empty git repository", func(t *testing.T) {
		repoDir := setupEmptyTestRepo(t)

		name, rev, err := HeadInfo(repoDir)

		assert.Error(t, err)
		assert.Empty(t, name)
		assert.Empty(t, rev)
		assert.Contains(t, err.Error(), "unable to get git repository")
		assert.Contains(t, err.Error(), "HEAD")
	})

	t.Run("head info with whitespace handling", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		name, rev, err := HeadInfo(repoDir)

		assert.NoError(t, err)
		// Verify that strings.Trim worked correctly (no \r\n characters)
		assert.False(t, strings.Contains(name, "\r"))
		assert.False(t, strings.Contains(name, "\n"))
		assert.False(t, strings.Contains(rev, "\r"))
		assert.False(t, strings.Contains(rev, "\n"))
	})
}

func TestStatus(t *testing.T) {
	t.Run("clean repository status", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.True(t, isClean, "repository should be clean")
	})

	t.Run("dirty repository status", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Modify existing file to make repository dirty
		testFile := filepath.Join(repoDir, testFileName)
		err := os.WriteFile(testFile, []byte("modified content"), 0644)
		require.NoError(t, err, "failed to modify test file")

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.False(t, isClean, "repository should be dirty")
	})

	t.Run("repository with untracked files", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Add untracked file
		untrackedFile := filepath.Join(repoDir, "untracked.txt")
		err := os.WriteFile(untrackedFile, []byte("untracked content"), 0644)
		require.NoError(t, err, "failed to create untracked file")

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.False(t, isClean, "repository with untracked files should not be clean")
	})

	t.Run("non-existent directory", func(t *testing.T) {
		nonExistentDir := "/non/existent/directory"

		isClean, err := Status(nonExistentDir)

		assert.Error(t, err)
		assert.False(t, isClean)
		assert.Contains(t, err.Error(), "is not valid git repository")
		assert.Contains(t, err.Error(), nonExistentDir)
	})

	t.Run("non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		isClean, err := Status(tempDir)

		assert.Error(t, err)
		assert.False(t, isClean)
		assert.Contains(t, err.Error(), "is not valid git repository")
		assert.Contains(t, err.Error(), tempDir)
	})

	t.Run("empty git repository", func(t *testing.T) {
		repoDir := setupEmptyTestRepo(t)

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.True(t, isClean, "empty repository should be clean")
	})

	t.Run("status logging output", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Capture log output
		var logBuffer bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&logBuffer)
		defer log.SetOutput(originalOutput)

		// Make repository dirty
		testFile := filepath.Join(repoDir, testFileName)
		err := os.WriteFile(testFile, []byte("modified for logging test"), 0644)
		require.NoError(t, err, "failed to modify test file")

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.False(t, isClean)

		// Verify log output
		logOutput := logBuffer.String()
		assert.Contains(t, logOutput, "Git repository status:")
	})

	t.Run("clean repository no logging", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Capture log output
		var logBuffer bytes.Buffer
		originalOutput := log.Writer()
		log.SetOutput(&logBuffer)
		defer log.SetOutput(originalOutput)

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.True(t, isClean)

		// Verify no log output for clean repository
		logOutput := logBuffer.String()
		assert.Empty(t, logOutput, "clean repository should not produce log output")
	})
}

// Integration tests
func TestHeadInfoIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("multiple commits head info", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Get initial head info
		initialName, initialRev, err := HeadInfo(repoDir)
		require.NoError(t, err)

		// Create another commit
		repo, err := goGit.PlainOpen(repoDir)
		require.NoError(t, err)

		worktree, err := repo.Worktree()
		require.NoError(t, err)

		// Add another file
		secondFile := filepath.Join(repoDir, "second.txt")
		err = os.WriteFile(secondFile, []byte("second file"), 0644)
		require.NoError(t, err)

		_, err = worktree.Add("second.txt")
		require.NoError(t, err)

		_, err = worktree.Commit("second commit", &goGit.CommitOptions{
			Author: &object.Signature{
				Name:  testAuthorName,
				Email: testAuthorEmail,
				When:  time.Now(),
			},
		})
		require.NoError(t, err)

		// Get new head info
		newName, newRev, err := HeadInfo(repoDir)
		require.NoError(t, err)

		// Branch name should be the same, but revision should be different
		assert.Equal(t, initialName, newName, "branch name should remain the same")
		assert.NotEqual(t, initialRev, newRev, "revision should be different after new commit")
	})
}

func TestStatusIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("staged changes status", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Add a new file and stage it
		repo, err := goGit.PlainOpen(repoDir)
		require.NoError(t, err)

		worktree, err := repo.Worktree()
		require.NoError(t, err)

		stagedFile := filepath.Join(repoDir, "staged.txt")
		err = os.WriteFile(stagedFile, []byte("staged content"), 0644)
		require.NoError(t, err)

		_, err = worktree.Add("staged.txt")
		require.NoError(t, err)

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.False(t, isClean, "repository with staged changes should not be clean")
	})

	t.Run("deleted file status", func(t *testing.T) {
		repoDir := setupTestRepo(t)

		// Delete the committed file
		testFile := filepath.Join(repoDir, testFileName)
		err := os.Remove(testFile)
		require.NoError(t, err)

		isClean, err := Status(repoDir)

		assert.NoError(t, err)
		assert.False(t, isClean, "repository with deleted files should not be clean")
	})
}

// Benchmark tests
func BenchmarkHeadInfo(b *testing.B) {
	// Setup repository once for all benchmark iterations
	tempDir := b.TempDir()
	repo, err := goGit.PlainInit(tempDir, false)
	if err != nil {
		b.Fatalf("failed to setup benchmark repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tempDir, testFileName)
	err = os.WriteFile(testFile, []byte(testFileContent), 0644)
	if err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		b.Fatalf("failed to get worktree: %v", err)
	}

	_, err = worktree.Add(testFileName)
	if err != nil {
		b.Fatalf("failed to add file: %v", err)
	}

	_, err = worktree.Commit(testCommitMsg, &goGit.CommitOptions{
		Author: &object.Signature{
			Name:  testAuthorName,
			Email: testAuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		b.Fatalf("failed to commit: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := HeadInfo(tempDir)
		if err != nil {
			b.Fatalf("HeadInfo failed: %v", err)
		}
	}
}

func BenchmarkStatus(b *testing.B) {
	// Setup repository once for all benchmark iterations
	tempDir := b.TempDir()
	repo, err := goGit.PlainInit(tempDir, false)
	if err != nil {
		b.Fatalf("failed to setup benchmark repo: %v", err)
	}

	// Create initial commit
	testFile := filepath.Join(tempDir, testFileName)
	err = os.WriteFile(testFile, []byte(testFileContent), 0644)
	if err != nil {
		b.Fatalf("failed to create test file: %v", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		b.Fatalf("failed to get worktree: %v", err)
	}

	_, err = worktree.Add(testFileName)
	if err != nil {
		b.Fatalf("failed to add file: %v", err)
	}

	_, err = worktree.Commit(testCommitMsg, &goGit.CommitOptions{
		Author: &object.Signature{
			Name:  testAuthorName,
			Email: testAuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		b.Fatalf("failed to commit: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := Status(tempDir)
		if err != nil {
			b.Fatalf("Status failed: %v", err)
		}
	}
}

// Table-driven tests
func TestHeadInfoErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		setupDir      func(t *testing.T) string
		expectedError string
	}{
		{
			name: "non-existent directory",
			setupDir: func(t *testing.T) string {
				return "/absolutely/non/existent/path"
			},
			expectedError: "is not valid git repository",
		},
		{
			name: "regular directory",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedError: "is not valid git repository",
		},
		{
			name: "empty git repository",
			setupDir: func(t *testing.T) string {
				return setupEmptyTestRepo(t)
			},
			expectedError: "unable to get git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupDir(t)

			name, rev, err := HeadInfo(dir)

			assert.Error(t, err)
			assert.Empty(t, name)
			assert.Empty(t, rev)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestStatusErrorMessages(t *testing.T) {
	tests := []struct {
		name          string
		setupDir      func(t *testing.T) string
		expectedError string
	}{
		{
			name: "non-existent directory",
			setupDir: func(t *testing.T) string {
				return "/absolutely/non/existent/path"
			},
			expectedError: "is not valid git repository",
		},
		{
			name: "regular directory",
			setupDir: func(t *testing.T) string {
				return t.TempDir()
			},
			expectedError: "is not valid git repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupDir(t)

			isClean, err := Status(dir)

			assert.Error(t, err)
			assert.False(t, isClean)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

// Edge case tests
func TestEdgeCases(t *testing.T) {
	t.Run("head info with empty directory path", func(t *testing.T) {
		name, rev, err := HeadInfo("")

		assert.Error(t, err)
		assert.Empty(t, name)
		assert.Empty(t, rev)
	})

	t.Run("status with empty directory path", func(t *testing.T) {
		isClean, err := Status("")

		assert.Error(t, err)
		assert.False(t, isClean)
	})

	t.Run("head info with dot directory", func(t *testing.T) {
		originalDir, err := os.Getwd()
		require.NoError(t, err)

		repoDir := setupTestRepo(t)
		err = os.Chdir(repoDir)
		require.NoError(t, err)
		defer func() {
			err := os.Chdir(originalDir)
			require.NoError(t, err)
		}()

		name, rev, err := HeadInfo(".")

		assert.NoError(t, err)
		assert.NotEmpty(t, name)
		assert.NotEmpty(t, rev)
	})
}
