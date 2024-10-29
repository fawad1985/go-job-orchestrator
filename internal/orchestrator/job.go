// job.go contains core job execution logic for the orchestrator
// It manages job lifecycle from enqueuing to completion
// Handles task execution, state management, and error handling
package orchestrator

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/fawad1985/go-job-orchestrator/pkg/models"
)

// EnqueueJob adds a new job to the execution queue
// It creates a new job execution instance and stores it in the database
// Returns the execution ID for tracking the job
func (o *Orchestrator) EnqueueJob(definitionID string, data map[string]interface{}) (string, error) {
	// Create a new job execution instance with unique ID and initial state
	// Uses timestamp-based ID for uniqueness and temporal tracking
	execution := &models.JobExecution{
		ID:           fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		DefinitionID: definitionID,
		Status:       models.JobStatusQueued,
		StartTime:    time.Now(),
		Data:         data,
	}

	// Store the job execution in the database
	// This persists the initial state before queueing
	if err := o.db.StoreJobExecution(execution); err != nil {
		return "", err
	}

	// Add the job to the execution queue
	// Once queued, workers can pick it up for execution
	if err := o.db.EnqueueJob(execution.ID); err != nil {
		return "", err
	}

	return execution.ID, nil
}

// ExecuteJob runs a job and all its tasks in sequence
// Manages the complete lifecycle of a job execution
// Handles state transitions, task execution, and error cases
func (o *Orchestrator) ExecuteJob(ctx context.Context, executionID string) error {
	// Retrieve the job execution details from storage
	// This includes current state and execution parameters
	je, err := o.db.GetJobExecution(executionID)
	if err != nil {
		return fmt.Errorf("failed to get job execution: %w", err)
	}

	// Skip if job is already in a terminal state
	// Prevents re-execution of completed or failed jobs
	if je.Status == models.JobStatusCompleted || je.Status == models.JobStatusFailed {
		return nil
	}

	// Get the job definition that specifies what tasks to run
	// This contains the task sequence and configuration
	jd, err := o.db.GetJobDefinition(je.DefinitionID)
	if err != nil {
		return fmt.Errorf("failed to get job definition: %w", err)
	}

	// Update job status to running and track in memory
	// This marks the beginning of job execution
	je.Status = models.JobStatusRunning
	if err := o.db.UpdateJobExecution(je); err != nil {
		return fmt.Errorf("failed to update job execution status to running: %w", err)
	}

	// Track this job as currently executing
	// Used for system state monitoring
	o.ongoingJobs.Store(executionID, struct{}{})

	// Ensure cleanup happens regardless of execution outcome
	// Updates final state and removes from tracking
	defer func() {
		o.ongoingJobs.Delete(executionID)
		je.EndTime = time.Now()
		if err := o.db.UpdateJobExecution(je); err != nil {
			log.Printf("Failed to update job execution after completion: %v", err)
		}
		if err := o.db.RemoveFromQueue(executionID); err != nil {
			log.Printf("Failed to remove job %s from queue: %v", executionID, err)
		}
	}()

	// Initialize task status tracking if needed
	// Maps task IDs to their current execution status
	if je.TaskStatuses == nil {
		je.TaskStatuses = make(map[string]models.TaskStatus)
	}

	// Execute each task in the job sequentially
	// Handles task state management and error cases
	for _, task := range jd.Tasks {
		select {
		case <-ctx.Done():
			// Handle context cancellation
			// Updates job and task state to failed
			je.Status = models.JobStatusFailed
			je.TaskStatuses[task.ID] = models.TaskStatusFailed
			return ctx.Err()

		default:
			// Update task status to running
			// Tracks progress through the task sequence
			je.TaskStatuses[task.ID] = models.TaskStatusRunning
			if err := o.db.UpdateJobExecution(je); err != nil {
				log.Printf("Failed to update task status to running: %v", err)
			}

			// Execute the task with its configured handler
			// Attempts execution with retry logic
			if err := o.executeTask(ctx, task, je.Data); err != nil {
				je.TaskStatuses[task.ID] = models.TaskStatusFailed
				je.Status = models.JobStatusFailed
				if updateErr := o.db.UpdateJobExecution(je); updateErr != nil {
					log.Printf("Failed to update job execution after task failure: %v", updateErr)
				}
				return fmt.Errorf("task %s failed: %w", task.ID, err)
			}

			// Update task status to completed
			// Marks successful task execution
			je.TaskStatuses[task.ID] = models.TaskStatusCompleted
			if err := o.db.UpdateJobExecution(je); err != nil {
				log.Printf("Failed to update task status to completed: %v", err)
			}
		}
	}

	// Update job status to completed after all tasks succeed
	// Marks successful job completion
	je.Status = models.JobStatusCompleted
	if err := o.db.UpdateJobExecution(je); err != nil {
		return fmt.Errorf("failed to update job execution status to completed: %w", err)
	}

	return nil
}

// GetJobExecutionState retrieves the current state of a job execution
// Combines job execution state with task states for status reporting
// Returns a complete snapshot of job and task status
func (o *Orchestrator) GetJobExecutionState(executionID string) (*models.JobExecutionState, error) {
	// Get the current job execution state
	// Includes status, timing, and task states
	je, err := o.db.GetJobExecution(executionID)
	if err != nil {
		return nil, err
	}

	// Get the corresponding job definition
	// Used to include task metadata in state
	jd, err := o.db.GetJobDefinition(je.DefinitionID)
	if err != nil {
		return nil, err
	}

	// Create the state response structure
	// Combines execution state with job definition details
	state := &models.JobExecutionState{
		ID:           je.ID,
		DefinitionID: je.DefinitionID,
		Status:       je.Status,
		StartTime:    je.StartTime,
	}

	// Build task state list combining definition and execution state
	// Provides complete task execution progress
	for _, task := range jd.Tasks {
		taskState := models.TaskState{
			ID:     task.ID,
			Name:   task.Name,
			Status: je.TaskStatuses[task.ID],
		}
		state.Tasks = append(state.Tasks, taskState)
	}

	return state, nil
}
