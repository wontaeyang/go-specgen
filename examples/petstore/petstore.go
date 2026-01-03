//	@api {
//	  @title Petstore API
//	  @version 1.0.0
//	  @description A sample Pet Store API demonstrating go-specgen features.
//	  This API allows you to manage pets in a store with full CRUD operations.
//	  @termsOfService https://petstore.example.com/terms
//	  @contact {
//	    @name API Support
//	    @email support@petstore.example.com
//	    @url https://petstore.example.com/support
//	  }
//	  @license {
//	    @name MIT
//	    @url https://opensource.org/licenses/MIT
//	  }
//	  @server https://api.petstore.example.com {
//	    @description Production server
//	  }
//	  @server https://staging.petstore.example.com {
//	    @description Staging server
//	  }
//	  @tag pets {
//	    @description Pet management operations
//	  }
//	  @tag store {
//	    @description Store operations
//	  }
//	  @securityScheme bearerAuth {
//	    @type http
//	    @scheme bearer
//	    @bearerFormat JWT
//	    @description JWT token authentication
//	  }
//	  @security {
//	    @with bearerAuth
//	  }
//	  @defaultContentType json
//	}
package petstore

import (
	"net/http"
	"time"
)

// -----------------------------------------------------------------------------
// Schemas
// -----------------------------------------------------------------------------

// Pet represents a pet in the store
//
//	@schema {
//	  @description A pet available in the store
//	}
type Pet struct {
	// @field { @description Unique identifier @format int64 @example 12345 }
	ID int64 `json:"id"`

	// @field { @description Pet name @example Fluffy @minLength 1 @maxLength 100 }
	Name string `json:"name"`

	// @field { @description Pet category @example dog @enum dog,cat,bird,fish,other }
	Category string `json:"category"`

	// @field { @description Pet availability status @enum available,pending,sold @default available }
	Status string `json:"status"`

	// @field { @description Pet tags for searching }
	Tags []string `json:"tags"`

	// @field { @description Pet age in years @minimum 0 @maximum 100 }
	Age *int `json:"age,omitempty"`

	// @field { @description When the pet was added to the store }
	CreatedAt time.Time `json:"created_at"`

	// @field { @description When the pet was last updated }
	UpdatedAt time.Time `json:"updated_at"`
}

// CreatePetRequest is the request body for creating a pet
// @schema
type CreatePetRequest struct {
	// @field { @description Pet name @minLength 1 @maxLength 100 }
	Name string `json:"name"`

	// @field { @description Pet category @enum dog,cat,bird,fish,other }
	Category string `json:"category"`

	// @field { @description Initial status @default available @enum available,pending }
	Status *string `json:"status,omitempty"`

	// @field { @description Pet tags for searching }
	Tags []string `json:"tags,omitempty"`

	// @field { @description Pet age in years @minimum 0 @maximum 100 }
	Age *int `json:"age,omitempty"`
}

// UpdatePetRequest is the request body for updating a pet
// @schema
type UpdatePetRequest struct {
	// @field { @description Pet name @minLength 1 @maxLength 100 }
	Name *string `json:"name,omitempty"`

	// @field { @description Pet category @enum dog,cat,bird,fish,other }
	Category *string `json:"category,omitempty"`

	// @field { @description Pet status @enum available,pending,sold }
	Status *string `json:"status,omitempty"`

	// @field { @description Pet tags for searching }
	Tags []string `json:"tags,omitempty"`

	// @field { @description Pet age in years @minimum 0 @maximum 100 }
	Age *int `json:"age,omitempty"`
}

// Error represents an API error response
//
//	@schema {
//	  @description Standard error response
//	}
type Error struct {
	// @field { @description Error code }
	Code string `json:"code"`

	// @field { @description Human-readable error message }
	Message string `json:"message"`

	// @field { @description Additional error details }
	Details []string `json:"details,omitempty"`
}

// -----------------------------------------------------------------------------
// Parameters
// -----------------------------------------------------------------------------

// PetIDPath contains path parameters for pet endpoints
// @path
type PetIDPath struct {
	// @field { @description Pet ID @format int64 }
	ID int64 `path:"id"`
}

// ListPetsQuery contains query parameters for listing pets
// @query
type ListPetsQuery struct {
	// @field { @description Maximum results to return @minimum 1 @maximum 100 @default 20 }
	Limit *int `query:"limit"`

	// @field { @description Offset for pagination @minimum 0 @default 0 }
	Offset *int `query:"offset"`

	// @field { @description Filter by status @enum available,pending,sold }
	Status *string `query:"status"`

	// @field { @description Filter by category @enum dog,cat,bird,fish,other }
	Category *string `query:"category"`

	// @field { @description Filter by tags (can specify multiple) }
	Tags []string `query:"tags"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// ListPets returns all pets with optional filtering
//
//	@endpoint GET /pets {
//	  @operationID listPets
//	  @summary List all pets
//	  @description Returns a paginated list of pets with optional filtering by status, category, and tags.
//	  @tag pets
//	  @query ListPetsQuery
//	  @response 200 {
//	    @body []Pet
//	    @description Successful response with list of pets
//	  }
//	  @response 400 {
//	    @body Error
//	    @description Invalid query parameters
//	  }
//	}
func ListPets(w http.ResponseWriter, r *http.Request) {}

// GetPet retrieves a pet by ID
//
//	@endpoint GET /pets/{id} {
//	  @operationID getPet
//	  @summary Get a pet by ID
//	  @description Returns a single pet by its unique identifier.
//	  @tag pets
//	  @path PetIDPath
//	  @response 200 {
//	    @body Pet
//	    @description Successful response with pet details
//	  }
//	  @response 404 {
//	    @body Error
//	    @description Pet not found
//	  }
//	}
func GetPet(w http.ResponseWriter, r *http.Request) {}

// CreatePet creates a new pet
//
//	@endpoint POST /pets {
//	  @operationID createPet
//	  @summary Create a new pet
//	  @description Creates a new pet in the store.
//	  @tag pets
//	  @request {
//	    @body CreatePetRequest
//	  }
//	  @response 201 {
//	    @body Pet
//	    @description Pet created successfully
//	  }
//	  @response 400 {
//	    @body Error
//	    @description Invalid request body
//	  }
//	}
func CreatePet(w http.ResponseWriter, r *http.Request) {}

// UpdatePet updates an existing pet
//
//	@endpoint PUT /pets/{id} {
//	  @operationID updatePet
//	  @summary Update a pet
//	  @description Updates an existing pet's information.
//	  @tag pets
//	  @path PetIDPath
//	  @request {
//	    @body UpdatePetRequest
//	  }
//	  @response 200 {
//	    @body Pet
//	    @description Pet updated successfully
//	  }
//	  @response 400 {
//	    @body Error
//	    @description Invalid request body
//	  }
//	  @response 404 {
//	    @body Error
//	    @description Pet not found
//	  }
//	}
func UpdatePet(w http.ResponseWriter, r *http.Request) {}

// DeletePet removes a pet from the store
//
//	@endpoint DELETE /pets/{id} {
//	  @operationID deletePet
//	  @summary Delete a pet
//	  @description Removes a pet from the store permanently.
//	  @tag pets
//	  @path PetIDPath
//	  @response 204 {
//	    @contentType empty
//	    @description Pet deleted successfully
//	  }
//	  @response 404 {
//	    @body Error
//	    @description Pet not found
//	  }
//	}
func DeletePet(w http.ResponseWriter, r *http.Request) {}

// GetStoreInventory returns pet counts by status
//
//	@endpoint GET /store/inventory {
//	  @operationID getInventory
//	  @summary Get store inventory
//	  @description Returns a map of status codes to quantities.
//	  This endpoint is deprecated and will be removed in a future version.
//	  @tag store
//	  @deprecated
//	  @response 200 {
//	    @body map[string]int
//	    @description Inventory counts by status
//	  }
//	}
func GetStoreInventory(w http.ResponseWriter, r *http.Request) {}
