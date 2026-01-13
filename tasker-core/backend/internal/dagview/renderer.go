package dagview

import (
	"fmt"
	"strings"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/domain"
)

// DAGNode represents a node in the task dependency graph
type DAGNode struct {
	Task     *domain.Task
	Children []*DAGNode
	Parents  []*DAGNode
	Level    int
	Visited  bool
}

// DAGRenderer renders task dependency graphs as ASCII art
type DAGRenderer struct {
	nodes           map[string]*DAGNode
	taskIDFormatter func(string) string
}

// NewDAGRenderer creates a new DAG renderer
func NewDAGRenderer() *DAGRenderer {
	return &DAGRenderer{
		nodes:           make(map[string]*DAGNode),
		taskIDFormatter: func(id string) string { return id }, // Default: no formatting
	}
}

// SetTaskIDFormatter sets a custom formatter for task IDs in the rendered output
func (r *DAGRenderer) SetTaskIDFormatter(formatter func(string) string) {
	if formatter != nil {
		r.taskIDFormatter = formatter
	}
}

// BuildGraph builds the dependency graph from a list of tasks
func (r *DAGRenderer) BuildGraph(tasks []*domain.Task) {
	r.nodes = make(map[string]*DAGNode)

	// Create nodes for all tasks
	for _, task := range tasks {
		if task == nil {
			continue
		}
		r.nodes[task.ID] = &DAGNode{
			Task:     task,
			Children: make([]*DAGNode, 0),
			Parents:  make([]*DAGNode, 0),
		}
	}

	// Build relationships
	for _, task := range tasks {
		if task == nil {
			continue
		}

		node := r.nodes[task.ID]

		// Connect to dependent tasks (outflows)
		for _, outflowID := range task.Outflows {
			if childNode, exists := r.nodes[outflowID]; exists {
				node.Children = append(node.Children, childNode)
				childNode.Parents = append(childNode.Parents, node)
			}
		}
	}

	// Calculate levels (topological sort)
	r.calculateLevels()
}

// calculateLevels assigns levels to nodes using topological sort
func (r *DAGRenderer) calculateLevels() {
	// Reset all nodes
	for _, node := range r.nodes {
		node.Level = 0
		node.Visited = false
	}

	// Find root nodes (no parents)
	var roots []*DAGNode
	for _, node := range r.nodes {
		if len(node.Parents) == 0 {
			roots = append(roots, node)
		}
	}

	// DFS to assign levels
	for _, root := range roots {
		r.assignLevel(root, 0)
	}
}

// assignLevel recursively assigns levels using DFS
func (r *DAGRenderer) assignLevel(node *DAGNode, level int) {
	if node.Visited && node.Level >= level {
		return
	}

	node.Level = level
	node.Visited = true

	for _, child := range node.Children {
		r.assignLevel(child, level+1)
	}
}

// RenderASCII renders the DAG as ASCII art
func (r *DAGRenderer) RenderASCII() string {
	if len(r.nodes) == 0 {
		return "No tasks to display"
	}

	// Group nodes by level
	levels := make(map[int][]*DAGNode)
	maxLevel := 0

	for _, node := range r.nodes {
		level := node.Level
		levels[level] = append(levels[level], node)
		if level > maxLevel {
			maxLevel = level
		}
	}

	var result strings.Builder
	result.WriteString("Task Dependency Graph:\n")
	result.WriteString(strings.Repeat("=", 50) + "\n\n")

	// Render each level
	for level := 0; level <= maxLevel; level++ {
		nodes := levels[level]
		if len(nodes) == 0 {
			continue
		}

		result.WriteString(fmt.Sprintf("Level %d:\n", level))

		for i, node := range nodes {
			task := node.Task
			stageIcon := r.getStageIcon(task.Stage)

			// Task info - handle short IDs safely
			taskIDDisplay := r.taskIDFormatter(task.ID)
			if len(taskIDDisplay) > 8 {
				taskIDDisplay = taskIDDisplay[:8]
			}
			result.WriteString(fmt.Sprintf("  %s [%s] %s (%s)\n",
				stageIcon, taskIDDisplay, task.Name, task.Stage.String()))

			// Show dependencies
			if len(task.Inflows) > 0 {
				result.WriteString(fmt.Sprintf("    ↑ Depends on: %s\n",
					strings.Join(r.getTaskNames(task.Inflows), ", ")))
			}

			if len(task.Outflows) > 0 {
				result.WriteString(fmt.Sprintf("    ↓ Enables: %s\n",
					strings.Join(r.getTaskNames(task.Outflows), ", ")))
			}

			// Add spacing between tasks
			if i < len(nodes)-1 {
				result.WriteString("    │\n")
			}
		}

		// Add level separator
		if level < maxLevel {
			result.WriteString("    ↓\n")
		}
		result.WriteString("\n")
	}

	return result.String()
}

// RenderCompact renders a compact view of the DAG
func (r *DAGRenderer) RenderCompact() string {
	if len(r.nodes) == 0 {
		return "No tasks to display"
	}

	var result strings.Builder
	result.WriteString("Task Dependencies (Compact):\n")
	result.WriteString(strings.Repeat("-", 40) + "\n")

	// Show all tasks with their immediate dependencies
	for _, node := range r.nodes {
		task := node.Task
		stageIcon := r.getStageIcon(task.Stage)

		result.WriteString(fmt.Sprintf("%s %s", stageIcon, task.Name))

		if len(task.Inflows) > 0 {
			result.WriteString(fmt.Sprintf(" ← %s", strings.Join(r.getTaskNames(task.Inflows), ", ")))
		}

		if len(task.Outflows) > 0 {
			result.WriteString(fmt.Sprintf(" → %s", strings.Join(r.getTaskNames(task.Outflows), ", ")))
		}

		result.WriteString("\n")
	}

	return result.String()
}

// getStageIcon returns an icon for the task stage
func (r *DAGRenderer) getStageIcon(stage domain.TaskStage) string {
	switch stage {
	case domain.StageInbox:
		return "📥"
	case domain.StagePending:
		return "⏳"
	case domain.StageStaging:
		return "🎭"
	case domain.StageActive:
		return "🔄"
	case domain.StageArchived:
		return "✅"
	default:
		return "❓"
	}
}

// getTaskNames converts task IDs to task names for display
func (r *DAGRenderer) getTaskNames(taskIDs []string) []string {
	names := make([]string, 0, len(taskIDs))
	for _, id := range taskIDs {
		if node, exists := r.nodes[id]; exists && node.Task != nil {
			names = append(names, node.Task.Name)
		} else {
			// fallback to formatted ID - handle short IDs safely
			formattedID := r.taskIDFormatter(id)
			if len(formattedID) > 8 {
				formattedID = formattedID[:8]
			}
			names = append(names, formattedID+"...")
		}
	}
	return names
}

// GetStats returns statistics about the DAG
func (r *DAGRenderer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"total_tasks": len(r.nodes),
		"root_tasks":  0,
		"leaf_tasks":  0,
		"max_level":   0,
	}

	for _, node := range r.nodes {
		if len(node.Parents) == 0 {
			stats["root_tasks"] = stats["root_tasks"].(int) + 1
		}
		if len(node.Children) == 0 {
			stats["leaf_tasks"] = stats["leaf_tasks"].(int) + 1
		}
		if node.Level > stats["max_level"].(int) {
			stats["max_level"] = node.Level
		}
	}

	return stats
}
