//	@api {
//	  @title Decode Rule Fixture
//	  @version 1.0.0
//	  @defaultContentType json
//	}
package decode

import "net/http"

// Tag is a named schema referenced from the inline request below. It is
// directionless, so its own fields keep the marshal rule.
// @schema
type Tag struct {
	Label *string `json:"label"`
}

// Create exercises every branch of the decode rule on an inline @request,
// and confirms an inline @response is left on the marshal rule.
// @endpoint POST /things {
// }
func Create(w http.ResponseWriter, r *http.Request) {
	// @request
	var req struct {
		Plain    string  `json:"plain"`
		Ptr      *string `json:"ptr"`
		Omit     string  `json:"omit,omitempty"`
		PtrOmit  *string `json:"ptr_omit,omitempty"`
		Slice    []int   `json:"slice"`
		SlicePtr *[]int  `json:"slice_ptr"`

		// @field { @required false }
		Optional string `json:"optional"`
		// @field { @nullable true }
		Nullable *string `json:"nullable"`
		// @field { @required true @nullable true }
		Forced *string `json:"forced"`

		Nested struct {
			Inner    string  `json:"inner"`
			InnerPtr *string `json:"inner_ptr"`
		} `json:"nested"`

		Items []struct {
			Item    string  `json:"item"`
			ItemPtr *string `json:"item_ptr"`
		} `json:"items"`

		ByKey map[string]struct {
			Val    string  `json:"val"`
			ValPtr *string `json:"val_ptr"`
		} `json:"by_key"`

		Tag Tag `json:"tag"`
	}

	// @response 201
	var created struct {
		ID    string  `json:"id"`
		Alias *string `json:"alias"`
	}

	_ = req
	_ = created
}
