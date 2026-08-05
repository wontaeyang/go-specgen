//	@api {
//	  @title ArrayConstraints Fixture
//	  @version 1.0.0
//	}
package arrayconstraints

// TagList demonstrates array item constraints.
//
//	@schema {
//	  @description A list of tags
//	}
type TagList struct {
	// @field { @description Unique tags @uniqueItems @minItems 1 @maxItems 10 }
	Tags []string `json:"tags"`

	// @field { @description Scores @minItems 2 }
	Scores []int `json:"scores,omitempty"`
}

// GetTags returns tags.
//
//	@endpoint GET /tags {
//	  @response 200 { @body TagList @description Tag list }
//	}
func GetTags() {}
