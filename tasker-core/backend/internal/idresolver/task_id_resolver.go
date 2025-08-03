package idresolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
)

// TrieNode represents a node in the task ID trie
type TrieNode struct {
	children map[rune]*TrieNode
	taskIDs  []string // task IDs that have this prefix
	isEnd    bool
}

// NewTrieNode creates a new trie node
func NewTrieNode() *TrieNode {
	return &TrieNode{
		children: make(map[rune]*TrieNode),
		taskIDs:  []string{},
		isEnd:    false,
	}
}

// TaskIDResolver provides task ID resolution functionality
type TaskIDResolver struct {
	userTries map[string]*TrieNode // user ID -> trie
	taskMap   map[string]*domain.Task
}

// NewTaskIDResolver creates a new task ID resolver
func NewTaskIDResolver() *TaskIDResolver {
	return &TaskIDResolver{
		userTries: make(map[string]*TrieNode),
		taskMap:   make(map[string]*domain.Task),
	}
}

// UpdateTasks updates the internal tries with the current set of tasks
func (r *TaskIDResolver) UpdateTasks(tasks []*domain.Task) {
	// Reset the tries and map
	r.userTries = make(map[string]*TrieNode)
	r.taskMap = make(map[string]*domain.Task)

	// Add all tasks to the map and user-specific tries
	for _, task := range tasks {
		if task == nil {
			continue // Skip nil tasks
		}
		r.taskMap[task.ID] = task

		// Ensure user trie exists
		if _, exists := r.userTries[task.UserID]; !exists {
			r.userTries[task.UserID] = NewTrieNode()
		}

		r.insertTaskID(task.ID, task.UserID)
	}
}

// insertTaskID inserts a task ID into the user-specific trie
func (r *TaskIDResolver) insertTaskID(taskID, userID string) {
	trie, exists := r.userTries[userID]
	if !exists {
		trie = NewTrieNode()
		r.userTries[userID] = trie
	}

	node := trie
	for i, ch := range strings.ToLower(taskID) {
		if _, exists := node.children[ch]; !exists {
			node.children[ch] = NewTrieNode()
		}
		node = node.children[ch]
		node.taskIDs = append(node.taskIDs, taskID)

		// Mark as end if this is the last character
		if i == len(taskID)-1 {
			node.isEnd = true
		}
	}
}

// ResolveTaskID resolves a partial task ID to a full task ID for a specific user
func (r *TaskIDResolver) ResolveTaskID(partialID string) (string, error) {
	return r.ResolveTaskIDForUser(partialID, "")
}

// ResolveTaskIDForUser resolves a partial task ID to a full task ID for a specific user
func (r *TaskIDResolver) ResolveTaskIDForUser(partialID, userID string) (string, error) {
	if partialID == "" {
		return "", fmt.Errorf("empty task ID provided")
	}

	// If it's already a full ID and exists, return it
	if task, exists := r.taskMap[partialID]; exists {
		// If userID is specified, check if the task belongs to that user
		if userID != "" && task.UserID != userID {
			return "", fmt.Errorf("task '%s' does not belong to user '%s'", partialID, userID)
		}
		return partialID, nil
	}

	// If userID is specified, search in user-specific trie
	if userID != "" {
		trie, exists := r.userTries[userID]
		if !exists {
			return "", fmt.Errorf("no tasks found for user '%s'", userID)
		}

		return r.searchInTrie(trie, partialID)
	}

	// Search in all user tries if no specific user
	var allMatches []string
	for _, trie := range r.userTries {
		if matches := r.getMatchesFromTrie(trie, partialID); len(matches) > 0 {
			allMatches = append(allMatches, matches...)
		}
	}

	if len(allMatches) == 0 {
		return "", fmt.Errorf("no task found with prefix '%s'", partialID)
	}

	if len(allMatches) == 1 {
		return allMatches[0], nil
	}

	// Multiple matches - return error with suggestions
	sort.Strings(allMatches)
	return "", fmt.Errorf("ambiguous task ID '%s', matches: %s", partialID, strings.Join(allMatches, ", "))
}

// searchInTrie searches for matches in a specific trie
func (r *TaskIDResolver) searchInTrie(trie *TrieNode, partialID string) (string, error) {
	node := trie
	for _, ch := range strings.ToLower(partialID) {
		if child, exists := node.children[ch]; exists {
			node = child
		} else {
			return "", fmt.Errorf("no task found with prefix '%s'", partialID)
		}
	}

	// Check how many tasks match this prefix
	if len(node.taskIDs) == 0 {
		return "", fmt.Errorf("no task found with prefix '%s'", partialID)
	}

	if len(node.taskIDs) == 1 {
		return node.taskIDs[0], nil
	}

	// Multiple matches - return error with suggestions
	sort.Strings(node.taskIDs)
	return "", fmt.Errorf("ambiguous task ID '%s', matches: %s", partialID, strings.Join(node.taskIDs, ", "))
}

// getMatchesFromTrie gets all matches from a trie for a partial ID
func (r *TaskIDResolver) getMatchesFromTrie(trie *TrieNode, partialID string) []string {
	node := trie
	for _, ch := range strings.ToLower(partialID) {
		if child, exists := node.children[ch]; exists {
			node = child
		} else {
			return []string{}
		}
	}
	return node.taskIDs
}

// GetTask retrieves a task by partial ID
func (r *TaskIDResolver) GetTask(partialID string) (*domain.Task, error) {
	fullID, err := r.ResolveTaskID(partialID)
	if err != nil {
		return nil, err
	}

	task, exists := r.taskMap[fullID]
	if !exists {
		return nil, fmt.Errorf("task with ID '%s' not found", fullID)
	}

	return task, nil
}

// GetMinimumUniquePrefix returns the minimum unique prefix for a given task ID
func (r *TaskIDResolver) GetMinimumUniquePrefix(taskID string) string {
	task, exists := r.taskMap[taskID]
	if !exists {
		return taskID // Task doesn't exist, return full ID
	}

	return r.GetMinimumUniquePrefixForUser(taskID, task.UserID)
}

// GetMinimumUniquePrefixForUser returns the minimum unique prefix for a given task ID within a user's context
func (r *TaskIDResolver) GetMinimumUniquePrefixForUser(taskID, userID string) string {
	trie, exists := r.userTries[userID]
	if !exists {
		return taskID // User doesn't exist, return full ID
	}

	node := trie
	prefix := ""

	for _, ch := range strings.ToLower(taskID) {
		prefix += string(ch)
		if child, exists := node.children[ch]; exists {
			node = child
			// If this prefix uniquely identifies the task, return it
			if len(node.taskIDs) == 1 {
				return prefix
			}
		} else {
			// This shouldn't happen if the task exists in the trie
			return taskID
		}
	}

	return taskID // Return full ID if no unique prefix found
}

// ListTasksWithPrefixes returns all tasks with their minimum unique prefixes
func (r *TaskIDResolver) ListTasksWithPrefixes() map[string]string {
	result := make(map[string]string)
	for taskID := range r.taskMap {
		result[taskID] = r.GetMinimumUniquePrefix(taskID)
	}
	return result
}

// ListTasksWithPrefixesForUser returns all tasks for a user with their minimum unique prefixes
func (r *TaskIDResolver) ListTasksWithPrefixesForUser(userID string) map[string]string {
	result := make(map[string]string)
	for taskID, task := range r.taskMap {
		if task.UserID == userID {
			result[taskID] = r.GetMinimumUniquePrefixForUser(taskID, userID)
		}
	}
	return result
}

// SuggestSimilarIDs suggests similar task IDs for a given partial ID
func (r *TaskIDResolver) SuggestSimilarIDs(partialID string, maxSuggestions int) []string {
	return r.SuggestSimilarIDsForUser(partialID, "", maxSuggestions)
}

// SuggestSimilarIDsForUser suggests similar task IDs for a given partial ID within a user's context
func (r *TaskIDResolver) SuggestSimilarIDsForUser(partialID, userID string, maxSuggestions int) []string {
	suggestions := []string{}

	// Find all task IDs that start with the partial ID
	for taskID, task := range r.taskMap {
		// If userID is specified, only suggest tasks for that user
		if userID != "" && task.UserID != userID {
			continue
		}

		if strings.HasPrefix(strings.ToLower(taskID), strings.ToLower(partialID)) {
			suggestions = append(suggestions, taskID)
		}
	}

	// Sort suggestions
	sort.Strings(suggestions)

	// Limit the number of suggestions
	if len(suggestions) > maxSuggestions {
		suggestions = suggestions[:maxSuggestions]
	}

	return suggestions
}
