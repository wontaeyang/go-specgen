//	@api {
//	  @title Enum Example
//	  @version 1.0.0
//	  @description Demonstrates enum support for strings, integers, and arrays.
//	  @defaultContentType json
//	}
package enum

import "net/http"

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// Task represents a task with various enum fields
// @schema
type Task struct {
	// @field { @description Task ID @format uuid }
	ID string `json:"id"`

	// @field { @description Task title }
	Title string `json:"title"`

	// @field { @description Task status @enum pending,in_progress,completed,cancelled }
	Status string `json:"status"`

	// @field { @description Priority level (1=low, 2=medium, 3=high) @enum 1,2,3 }
	Priority int `json:"priority"`

	// @field { @description Allowed categories for this task @enum work,personal,health,finance }
	Categories []string `json:"categories"`

	// @field { @description Allowed priority levels for subtasks @enum 1,2,3 }
	SubtaskPriorities []int `json:"subtask_priorities"`
}

// CreateTaskRequest represents a request to create a task
// @schema
type CreateTaskRequest struct {
	// @field { @description Task title }
	Title string `json:"title"`

	// @field { @description Initial status @enum pending,in_progress @default pending }
	Status string `json:"status"`

	// @field { @description Priority level @enum 1,2,3 @default 2 }
	Priority int `json:"priority"`

	// @field { @description Task categories @enum work,personal,health,finance }
	Categories []string `json:"categories,omitempty"`
}

// -----------------------------------------------------------------------------
// Query Parameters
// -----------------------------------------------------------------------------

// TaskQuery demonstrates enum in query parameters
// @query
type TaskQuery struct {
	// @field { @description Filter by status @enum pending,in_progress,completed,cancelled }
	Status *string `query:"status"`

	// @field { @description Filter by priority @enum 1,2,3 }
	Priority *int `query:"priority"`

	// @field { @description Filter by categories (can specify multiple) @enum work,personal,health,finance }
	Categories []string `query:"categories"`

	// @field { @description Sort order @enum asc,desc @default asc }
	Sort *string `query:"sort"`
}

// -----------------------------------------------------------------------------
// Path Parameters
// -----------------------------------------------------------------------------

// TaskPath contains the task ID path parameter
// @path
type TaskPath struct {
	// @field { @description Task ID @format uuid }
	ID string `path:"id"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// ListTasks returns tasks filtered by enum parameters
//
//	@endpoint GET /tasks {
//	  @operationID listTasks
//	  @summary List tasks
//	  @description List tasks with optional enum filters.
//	  @query TaskQuery
//	  @response 200 {
//	    @body []Task
//	    @description List of tasks
//	  }
//	}
func ListTasks(w http.ResponseWriter, r *http.Request) {}

// CreateTask creates a new task
//
//	@endpoint POST /tasks {
//	  @operationID createTask
//	  @summary Create a task
//	  @description Create a new task with enum-constrained fields.
//	  @request {
//	    @body CreateTaskRequest
//	  }
//	  @response 201 {
//	    @body Task
//	    @description Task created
//	  }
//	}
func CreateTask(w http.ResponseWriter, r *http.Request) {}

// GetTask returns a single task
//
//	@endpoint GET /tasks/{id} {
//	  @operationID getTask
//	  @summary Get a task
//	  @path TaskPath
//	  @response 200 {
//	    @body Task
//	    @description Task found
//	  }
//	}
func GetTask(w http.ResponseWriter, r *http.Request) {}
