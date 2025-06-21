// Copyright (c) 2022 EPAM Systems, Inc.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package git

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// captureLogOutput captures log output during test execution
func captureLogOutput(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	originalPrefix := log.Prefix()

	// Set up log capture
	log.SetOutput(&buf)
	log.SetFlags(0) // Remove timestamp and other flags for cleaner testing
	log.SetPrefix("")

	// Restore original log settings
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		log.SetPrefix(originalPrefix)
	}()

	fn()

	return buf.String()
}

func TestPrintLocalGitRepos(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		output := captureLogOutput(t, func() {
			printLocalGitRepos([]LocalGitRepo{})
		})

		assert.Empty(t, output, "empty slice should produce no output")
	})

	t.Run("nil slice", func(t *testing.T) {
		output := captureLogOutput(t, func() {
			printLocalGitRepos(nil)
		})

		assert.Empty(t, output, "nil slice should produce no output")
	})

	t.Run("single repo basic info", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "https://github.com/user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project => `https://github.com/user/repo.git` at `refs/heads/main`\n"
		assert.Equal(t, expected, output)
	})

	t.Run("single repo with optimized remote", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "git@github.com:user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project => `git@github.com:user/repo.git` (https://github.com/user/repo.git) at `refs/heads/main`\n"
		assert.Equal(t, expected, output)
	})

	t.Run("single repo with different ref", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "https://github.com/user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "refs/heads/develop",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project => `https://github.com/user/repo.git` at `refs/heads/main` (refs/heads/develop)\n"
		assert.Equal(t, expected, output)
	})

	t.Run("single repo with subdirectory", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "https://github.com/user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "",
				SubDir:          "backend/api",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project => `https://github.com/user/repo.git` at `refs/heads/main` [/backend/api]\n"
		assert.Equal(t, expected, output)
	})

	t.Run("single repo with all fields different", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "git@github.com:user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "refs/heads/develop",
				SubDir:          "frontend/app",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project => `git@github.com:user/repo.git` (https://github.com/user/repo.git) at `refs/heads/main` (refs/heads/develop) [/frontend/app]\n"
		assert.Equal(t, expected, output)
	})

	t.Run("multiple repos", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project1",
				Remote:          "https://github.com/user/repo1.git",
				OptimizedRemote: "https://github.com/user/repo1.git",
				HeadRef:         "refs/heads/main",
				Ref:             "",
				SubDir:          "",
			},
			{
				AbsDir:          "/home/user/project2",
				Remote:          "https://github.com/user/repo2.git",
				OptimizedRemote: "git@github.com:user/repo2.git",
				HeadRef:         "refs/heads/develop",
				Ref:             "refs/heads/feature",
				SubDir:          "services",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		lines := strings.Split(output, "\n")
		assert.Len(t, lines, 3, "should have exactly 3 lines of output")

		expectedLine1 := "\t/home/user/project1 => `https://github.com/user/repo1.git` at `refs/heads/main`"
		expectedLine2 := "\t/home/user/project2 => `git@github.com:user/repo2.git` (https://github.com/user/repo2.git) at `refs/heads/develop` (refs/heads/feature) [/services]"

		assert.Equal(t, expectedLine1, lines[0])
		assert.Equal(t, expectedLine2, lines[1])
		assert.Equal(t, "", lines[2])
	})

	t.Run("repo with empty strings", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "",
				Remote:          "",
				OptimizedRemote: "",
				HeadRef:         "",
				Ref:             "",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t => `` at ``\n"
		assert.Equal(t, expected, output)
	})

	t.Run("repo with same ref and head ref", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project",
				Remote:          "https://github.com/user/repo.git",
				OptimizedRemote: "https://github.com/user/repo.git",
				HeadRef:         "refs/heads/main",
				Ref:             "refs/heads/main", // Same as HeadRef
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		// Should not show the ref in parentheses since it's the same as HeadRef
		expected := "\t/home/user/project => `https://github.com/user/repo.git` at `refs/heads/main`\n"
		assert.Equal(t, expected, output)
	})

	t.Run("repo with special characters", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/home/user/project with spaces",
				Remote:          "https://github.com/user/repo-with-dashes.git",
				OptimizedRemote: "git@github.com:user/repo-with-dashes.git",
				HeadRef:         "refs/heads/feature/special-branch",
				Ref:             "refs/tags/v1.0.0-beta",
				SubDir:          "path/with/multiple/levels",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		expected := "\t/home/user/project with spaces => `git@github.com:user/repo-with-dashes.git` (https://github.com/user/repo-with-dashes.git) at `refs/heads/feature/special-branch` (refs/tags/v1.0.0-beta) [/path/with/multiple/levels]\n"
		assert.Equal(t, expected, output)
	})
}

// Test the formatting logic in isolation
func TestPrintLocalGitReposFormatting(t *testing.T) {
	tests := []struct {
		name     string
		repo     LocalGitRepo
		expected string
	}{
		{
			name: "minimal repo",
			repo: LocalGitRepo{
				AbsDir:          "/path",
				Remote:          "remote",
				OptimizedRemote: "remote",
				HeadRef:         "main",
				Ref:             "",
				SubDir:          "",
			},
			expected: "\t/path => `remote` at `main`\n",
		},
		{
			name: "repo with optimized remote",
			repo: LocalGitRepo{
				AbsDir:          "/path",
				Remote:          "https://remote",
				OptimizedRemote: "ssh://remote",
				HeadRef:         "main",
				Ref:             "",
				SubDir:          "",
			},
			expected: "\t/path => `ssh://remote` (https://remote) at `main`\n",
		},
		{
			name: "repo with different ref",
			repo: LocalGitRepo{
				AbsDir:          "/path",
				Remote:          "remote",
				OptimizedRemote: "remote",
				HeadRef:         "main",
				Ref:             "develop",
				SubDir:          "",
			},
			expected: "\t/path => `remote` at `main` (develop)\n",
		},
		{
			name: "repo with subdirectory",
			repo: LocalGitRepo{
				AbsDir:          "/path",
				Remote:          "remote",
				OptimizedRemote: "remote",
				HeadRef:         "main",
				Ref:             "",
				SubDir:          "sub",
			},
			expected: "\t/path => `remote` at `main` [/sub]\n",
		},
		{
			name: "repo with all options",
			repo: LocalGitRepo{
				AbsDir:          "/path",
				Remote:          "https://remote",
				OptimizedRemote: "ssh://remote",
				HeadRef:         "main",
				Ref:             "develop",
				SubDir:          "sub",
			},
			expected: "\t/path => `ssh://remote` (https://remote) at `main` (develop) [/sub]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureLogOutput(t, func() {
				printLocalGitRepos([]LocalGitRepo{tt.repo})
			})

			assert.Equal(t, tt.expected, output)
		})
	}
}

// Benchmark tests
func BenchmarkPrintLocalGitRepos(b *testing.B) {
	repos := []LocalGitRepo{
		{
			AbsDir:          "/home/user/project1",
			Remote:          "https://github.com/user/repo1.git",
			OptimizedRemote: "git@github.com:user/repo1.git",
			HeadRef:         "refs/heads/main",
			Ref:             "refs/heads/develop",
			SubDir:          "backend",
		},
		{
			AbsDir:          "/home/user/project2",
			Remote:          "https://github.com/user/repo2.git",
			OptimizedRemote: "https://github.com/user/repo2.git",
			HeadRef:         "refs/heads/master",
			Ref:             "",
			SubDir:          "",
		},
	}

	// Redirect log output to discard during benchmark
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalOutput)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		printLocalGitRepos(repos)
	}
}

func BenchmarkPrintLocalGitReposSingle(b *testing.B) {
	repo := []LocalGitRepo{
		{
			AbsDir:          "/home/user/project",
			Remote:          "https://github.com/user/repo.git",
			OptimizedRemote: "git@github.com:user/repo.git",
			HeadRef:         "refs/heads/main",
			Ref:             "refs/heads/develop",
			SubDir:          "backend/api",
		},
	}

	// Redirect log output to discard during benchmark
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalOutput)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		printLocalGitRepos(repo)
	}
}

func BenchmarkPrintLocalGitReposLarge(b *testing.B) {
	// Create a large slice of repos
	repos := make([]LocalGitRepo, 100)
	for i := 0; i < 100; i++ {
		repos[i] = LocalGitRepo{
			AbsDir:          fmt.Sprintf("/home/user/project%d", i),
			Remote:          fmt.Sprintf("https://github.com/user/repo%d.git", i),
			OptimizedRemote: fmt.Sprintf("git@github.com:user/repo%d.git", i),
			HeadRef:         "refs/heads/main",
			Ref:             "refs/heads/develop",
			SubDir:          "backend",
		}
	}

	// Redirect log output to discard during benchmark
	originalOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalOutput)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		printLocalGitRepos(repos)
	}
}

// Edge case tests
func TestPrintLocalGitReposEdgeCases(t *testing.T) {
	t.Run("repo with backticks in values", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/path/with`backtick",
				Remote:          "remote`with`backticks",
				OptimizedRemote: "remote`with`backticks",
				HeadRef:         "main`branch",
				Ref:             "",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		// The function should still work, just with backticks in the output
		assert.Contains(t, output, "/path/with`backtick")
		assert.Contains(t, output, "`remote`with`backticks`")
		assert.Contains(t, output, "`main`branch`")
	})

	t.Run("repo with newlines in values", func(t *testing.T) {
		repos := []LocalGitRepo{
			{
				AbsDir:          "/path\nwith\nnewlines",
				Remote:          "remote\nwith\nnewlines",
				OptimizedRemote: "remote\nwith\nnewlines",
				HeadRef:         "main\nbranch",
				Ref:             "",
				SubDir:          "",
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		// Should contain the newlines as-is
		assert.Contains(t, output, "\n")
	})

	t.Run("very long values", func(t *testing.T) {
		longString := strings.Repeat("a", 1000)
		repos := []LocalGitRepo{
			{
				AbsDir:          longString,
				Remote:          longString,
				OptimizedRemote: longString,
				HeadRef:         longString,
				Ref:             "",
				SubDir:          longString,
			},
		}

		output := captureLogOutput(t, func() {
			printLocalGitRepos(repos)
		})

		assert.Contains(t, output, longString)
		assert.Contains(t, output, fmt.Sprintf("[/%s]", longString))
	})
}

// Test concurrent access (if the function were to be used concurrently)
func TestPrintLocalGitReposConcurrent(t *testing.T) {
	repos := []LocalGitRepo{
		{
			AbsDir:          "/home/user/project",
			Remote:          "https://github.com/user/repo.git",
			OptimizedRemote: "https://github.com/user/repo.git",
			HeadRef:         "refs/heads/main",
			Ref:             "",
			SubDir:          "",
		},
	}

	// Capture output in a thread-safe way
	var buf bytes.Buffer
	originalOutput := log.Writer()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer log.SetOutput(originalOutput)

	// Run multiple goroutines
	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			printLocalGitRepos(repos)
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	output := buf.String()

	// Should have output from all goroutines
	assert.NotEmpty(t, output)

	// Count the number of lines (should be numGoroutines)
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, numGoroutines)
}
