// Package undefined_tag cites an endpoint tag that the API never defines.
//
// The check used to be guarded by "len(pkg.API.Tags) > 0", so an API that
// declared no tags — exactly the case where every endpoint tag is undefined —
// skipped it, and the tag reached the spec.
//
//	@api {
//	  @title Undefined Tag Fixture
//	  @version 1.0.0
//	  @description An endpoint tag with no API-level definition.
//	  @defaultContentType json
//	}
package undefined_tag

import "net/http"

// ListWidgets is filed under a tag that was never declared.
//
//	@endpoint GET /widgets {
//	  @summary List widgets
//	  @tag inventory
//	  @response 200 { @body []string }
//	}
func ListWidgets(w http.ResponseWriter, r *http.Request) {}
