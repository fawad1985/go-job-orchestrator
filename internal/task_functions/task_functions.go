// task_functions.go defines and implements the available task functions
// Provides concrete implementations of tasks that can be executed by jobs
// Acts as a registry of all possible task operations in the system
package task_functions

import (
	"context"
	"log"
	"time"
)

// TaskFunctions interface defines all available task operations
// Any new task type must be added to this interface
// Ensures consistent function signatures across all tasks
type TaskFunctions interface {
	Task1(ctx context.Context, data map[string]interface{}) error
	Task2(ctx context.Context, data map[string]interface{}) error
	Task3(ctx context.Context, data map[string]interface{}) error
	// Add more task function signatures here as needed
	// Example: ProcessData(ctx context.Context, data map[string]interface{}) error
}

// Task1 implements a sample task operation
// Currently simulates work with a delay
// Can be enhanced to perform actual business logic
func Task1(ctx context.Context, data map[string]interface{}) error {
	// Log task execution with input data
	// Useful for debugging and monitoring
	log.Printf("Executing Task 1 with data %v", data)

	// Simulate work with a 10-second delay
	// In real implementation, would contain actual business logic
	time.Sleep(10 * time.Second)

	return nil
}

// Task2 implements another sample task operation
// Demonstrates different execution duration
// Template for implementing more complex tasks
func Task2(ctx context.Context, data map[string]interface{}) error {
	// Log the task execution
	// Helps with execution tracking
	log.Println("Executing Task 2")

	// Simulate work with an 8-second delay
	// Would be replaced with real task logic
	time.Sleep(8 * time.Second)

	return nil
}

// Task3 implements a third sample task operation
// Shows another variation of task execution
// Framework for adding more task types
func Task3(ctx context.Context, data map[string]interface{}) error {
	// Log task execution
	// Part of execution audit trail
	log.Println("Executing Task 3")

	// Simulate work with a 5-second delay
	// Placeholder for actual implementation
	time.Sleep(5 * time.Second)

	return nil
}

// GetTaskFunction returns the implementation for a given task name
// Maps task names to their concrete implementations
// Used by the orchestrator to resolve task functions
func GetTaskFunction(name string) interface{} {
	// Map task names to their implementations
	// Add new cases here when adding new tasks
	switch name {
	case "Task1":
		return Task1
	case "Task2":
		return Task2
	case "Task3":
		return Task3
	// Add additional task mappings here
	// Example: case "ProcessData": return ProcessData
	default:
		return nil
	}
}

/* Task Function Guidelines:

1. Function Signature:
  - Must accept context.Context for cancellation
  - Must accept map[string]interface{} for flexible data
  - Must return error for status reporting

2. Implementation Requirements:
  - Should be idempotent when possible
  - Should respect context cancellation
  - Should handle input validation
  - Should provide meaningful logs
  - Should handle errors appropriately

3. Adding New Tasks:
  1. Add function signature to TaskFunctions interface
  2. Implement the function with required signature
  3. Add mapping in GetTaskFunction switch statement
  4. Update job definitions to use new task

*/
