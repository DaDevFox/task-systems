package idresolver

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
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
	trie    *TrieNode
	taskMap map[string]*domain.Task
}

// NewTaskIDResolver creates a new task ID resolver
func NewTaskIDResolver() *TaskIDResolver {
	return &TaskIDResolver{
		trie:    NewTrieNode(),
		taskMap: make(map[string]*domain.Task),
	}
}

// UpdateTasks updates the internal trie with the current set of tasks
func (r *TaskIDResolver) UpdateTasks(tasks []*domain.Task) {
	// Reset the trie and map
	r.trie = NewTrieNode()
	r.taskMap = make(map[string]*domain.Task)

	// Add all tasks to the map and trie
	for _, task := range tasks {
		if task == nil {
			continue // Skip nil tasks
		}
		r.taskMap[task.ID] = task
		r.insertTaskID(task.ID)
	}
}

// insertTaskID inserts a task ID into the trie
func (r *TaskIDResolver) insertTaskID(taskID string) {
	node := r.trie
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

// ResolveTaskID resolves a partial task ID to a full task ID
func (r *TaskIDResolver) ResolveTaskID(partialID string) (string, error) {
	if partialID == "" {
		return "", fmt.Errorf("empty task ID provided")
	}

	// If it's already a full ID and exists, return it
	if _, exists := r.taskMap[partialID]; exists {
		return partialID, nil
	}

	// Search in the trie
	node := r.trie
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
	if _, exists := r.taskMap[taskID]; !exists {
		return taskID // Task doesn't exist, return full ID
	}

	node := r.trie
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

// SuggestSimilarIDs suggests similar task IDs for a given partial ID
func (r *TaskIDResolver) SuggestSimilarIDs(partialID string, maxSuggestions int) []string {
	suggestions := []string{}

	// Find all task IDs that start with the partial ID
	for taskID := range r.taskMap {
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
