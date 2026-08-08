// Package undefined_tag cites an endpoint tag that the API never defines.
//
// The validator has a rule for exactly this, but it is guarded by
// "len(pkg.API.Tags) > 0", so it only runs once at least one tag has been
// defined. An API that defines none — like this one — skips the check entirely,
// and the undefined tag reaches the spec.
//
// This fixture moves to testdata/errors when the guard is dropped.
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
