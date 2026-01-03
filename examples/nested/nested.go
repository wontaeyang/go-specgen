//	@api {
//	  @title Nested Schemas Example
//	  @version 1.0.0
//	  @description Demonstrates nested schema references and $ref generation.
//	  When a schema field references another @schema type, go-specgen
//	  automatically generates $ref references in the OpenAPI output.
//	  @defaultContentType json
//	}
package nested

import "net/http"

// -----------------------------------------------------------------------------
// Basic Nested Schemas
// -----------------------------------------------------------------------------

// Address represents a physical address
// @schema
type Address struct {
	// @field { @description Street address }
	Street string `json:"street"`

	// @field { @description City name }
	City string `json:"city"`

	// @field { @description State or province }
	State string `json:"state"`

	// @field { @description Postal code }
	PostalCode string `json:"postal_code"`

	// Using escaped braces for regex quantifier in inline format
	// @field { @description Country code @pattern ^[A-Z]\{2\}$ }
	Country string `json:"country"`
}

// User represents a user in the system
// @schema
type User struct {
	// @field { @description User ID @format uuid }
	ID string `json:"id"`

	// @field { @description User email @format email }
	Email string `json:"email"`

	// @field { @description User display name }
	Name string `json:"name"`

	// @field { @description User's home address }
	HomeAddress Address `json:"home_address"`

	// @field { @description User's work address (optional) }
	WorkAddress *Address `json:"work_address,omitempty"`
}

// -----------------------------------------------------------------------------
// Array of Nested Schemas
// -----------------------------------------------------------------------------

// OrderItem represents an item in an order
// @schema
type OrderItem struct {
	// @field { @description Product ID @format uuid }
	ProductID string `json:"product_id"`

	// @field { @description Product name }
	ProductName string `json:"product_name"`

	// @field { @description Quantity ordered @minimum 1 }
	Quantity int `json:"quantity"`

	// @field { @description Price per unit in cents @minimum 0 }
	UnitPriceCents int `json:"unit_price_cents"`
}

// Order represents a customer order with nested schemas
// @schema
type Order struct {
	// @field { @description Order ID @format uuid }
	ID string `json:"id"`

	// @field { @description Customer who placed the order }
	Customer User `json:"customer"`

	// @field { @description Shipping address }
	ShippingAddress Address `json:"shipping_address"`

	// @field { @description Billing address }
	BillingAddress Address `json:"billing_address"`

	// @field { @description Order items (array of nested schema) }
	Items []OrderItem `json:"items"`

	// @field { @description Order status @enum pending,processing,shipped,delivered }
	Status string `json:"status"`

	// @field { @description Total in cents @minimum 0 }
	TotalCents int `json:"total_cents"`
}

// -----------------------------------------------------------------------------
// Deep Nesting (3+ levels)
// -----------------------------------------------------------------------------

// Member represents a team member
// @schema
type Member struct {
	// @field { @description Member ID @format uuid }
	ID string `json:"id"`

	// @field { @description Member name }
	Name string `json:"name"`

	// @field { @description Member role @enum admin,member,viewer }
	Role string `json:"role"`
}

// Team represents a team within an organization
// @schema
type Team struct {
	// @field { @description Team ID @format uuid }
	ID string `json:"id"`

	// @field { @description Team name }
	Name string `json:"name"`

	// @field { @description Team lead }
	Lead Member `json:"lead"`

	// @field { @description Team members }
	Members []Member `json:"members"`
}

// Organization represents an organization with nested teams
// @schema
type Organization struct {
	// @field { @description Organization ID @format uuid }
	ID string `json:"id"`

	// @field { @description Organization name }
	Name string `json:"name"`

	// @field { @description Headquarters address }
	Headquarters Address `json:"headquarters"`

	// @field { @description Organization teams }
	Teams []Team `json:"teams"`

	// @field { @description Primary contact }
	PrimaryContact User `json:"primary_contact"`
}

// -----------------------------------------------------------------------------
// Self-Referencing Schema
// -----------------------------------------------------------------------------

// Employee represents an employee with a manager (self-reference)
// @schema
type Employee struct {
	// @field { @description Employee ID @format uuid }
	ID string `json:"id"`

	// @field { @description Employee name }
	Name string `json:"name"`

	// @field { @description Employee title }
	Title string `json:"title"`

	// @field { @description Employee's manager (another Employee) }
	Manager *Employee `json:"manager,omitempty"`

	// @field { @description Direct reports }
	DirectReports []Employee `json:"direct_reports,omitempty"`
}

// -----------------------------------------------------------------------------
// Path Parameters
// -----------------------------------------------------------------------------

// OrderIDPath for order endpoints
// @path
type OrderIDPath struct {
	// @field { @description Order ID @format uuid }
	ID string `path:"id"`
}

// OrgIDPath for organization endpoints
// @path
type OrgIDPath struct {
	// @field { @description Organization ID @format uuid }
	ID string `path:"id"`
}

// EmployeeIDPath for employee endpoints
// @path
type EmployeeIDPath struct {
	// @field { @description Employee ID @format uuid }
	ID string `path:"id"`
}

// -----------------------------------------------------------------------------
// Endpoints
// -----------------------------------------------------------------------------

// GetOrder returns an order with all nested schemas
//
//	@endpoint GET /orders/{id} {
//	  @operationID getOrder
//	  @summary Get an order
//	  @description Returns an order with customer, addresses, and items.
//	  @path OrderIDPath
//	  @response 200 {
//	    @body Order
//	    @description Order with nested schemas
//	  }
//	}
func GetOrder(w http.ResponseWriter, r *http.Request) {}

// GetOrganization returns an organization with deeply nested schemas
//
//	@endpoint GET /organizations/{id} {
//	  @operationID getOrganization
//	  @summary Get an organization
//	  @description Returns an organization with teams, members, and contact.
//	  @path OrgIDPath
//	  @response 200 {
//	    @body Organization
//	    @description Organization with deep nesting
//	  }
//	}
func GetOrganization(w http.ResponseWriter, r *http.Request) {}

// GetEmployee returns an employee with self-referencing schema
//
//	@endpoint GET /employees/{id} {
//	  @operationID getEmployee
//	  @summary Get an employee
//	  @description Returns an employee with manager and direct reports.
//	  @path EmployeeIDPath
//	  @response 200 {
//	    @body Employee
//	    @description Employee with self-reference
//	  }
//	}
func GetEmployee(w http.ResponseWriter, r *http.Request) {}

// ListOrders returns a list of orders
//
//	@endpoint GET /orders {
//	  @operationID listOrders
//	  @summary List orders
//	  @description Returns an array of orders with nested schemas.
//	  @response 200 {
//	    @body []Order
//	    @description List of orders
//	  }
//	}
func ListOrders(w http.ResponseWriter, r *http.Request) {}
