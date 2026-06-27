package router

import (
	"strings"
)

type TaskSpec struct {
	ID          int    `json:"id"`
	Description string `json:"description"`
	Agent       string `json:"agent"`
	DependsOn   []int  `json:"depends_on"`
}

type TaskPlan struct {
	Reasoning string     `json:"reasoning"`
	Tasks     []TaskSpec `json:"tasks"`
}

func buildSystemPrompt(strengths map[string][]string, available []string, thinkingLevel string) string {
	var sb strings.Builder

	sb.WriteString("You are Trainstation, an AI task scheduler that routes coding tasks to the best AI coding agent.\n\n")

	sb.WriteString("Available agents and their strengths:")
	for _, name := range available {
		s := strengths[name]
		sb.WriteString("\n- " + name + ": " + joinStrings(s))
	}

	sb.WriteString("\n\nAnalyze the user's task and break it down into subtasks. For each subtask, assign the best agent based on its strengths.\n\n")

	sb.WriteString("Rules:\n")
	sb.WriteString("1. NEVER assign a user's prompt to a single agent. ALWAYS break the task down into multiple smaller subtasks.\n")
	sb.WriteString("2. Assign each subtask to the best agent for that specific job. You may assign multiple subtasks to the same agent if it is the most capable.\n")
	sb.WriteString("3. Maximize parallelization. Subtasks that do not strictly depend on each other MUST run in parallel (leave depends_on empty).\n")
	sb.WriteString("4. Dependent subtasks must list their dependencies in depends_on by ID.\n")
	sb.WriteString("5. Use only agent names from the available list above.\n")
	sb.WriteString("6. Keep task descriptions specific and actionable — they become the agent's prompt.\n")
	sb.WriteString("7. Include enough context in each task description so the agent can work autonomously.\n")

	switch thinkingLevel {
	case "low":
		sb.WriteString("\nBe quick and decisive. Minimal analysis, just route the task.\n")
	case "high":
		sb.WriteString("\nThink carefully about the optimal routing. Consider task complexity, agent specializations, and parallelization opportunities. Explain your reasoning thoroughly.\n")
	case "max":
		sb.WriteString("\nThink deeply and exhaustively. Consider every possible decomposition, parallelization strategy, and agent assignment. Provide detailed reasoning for each decision.\n")
	default:
		sb.WriteString("\nBalance speed and accuracy in your routing decisions.\n")
	}

	sb.WriteString("\nRespond with ONLY a JSON object (no markdown fences, no explanation):\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"reasoning\": \"Brief explanation of why you chose this routing\",\n")
	sb.WriteString("  \"tasks\": [\n")
	sb.WriteString("    {\"id\": 0, \"description\": \"specific task description with context\", \"agent\": \"agent_name\", \"depends_on\": []}\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n")

	return sb.String()
}

func joinStrings(s []string) string {
	result := ""
	for i, v := range s {
		if i > 0 {
			result += ", "
		}
		result += v
	}
	return result
}
