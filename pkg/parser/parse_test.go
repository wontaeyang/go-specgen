package parser

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// parseValid parses the comprehensive fixture package once per test that needs
// it. Parsing is cheap enough (one packages.Load) not to bother caching.
func parseValid(t *testing.T) *Package {
	t.Helper()
	pkg, err := Parse("./testdata/valid")
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return pkg
}

func TestParse_Package(t *testing.T) {
	pkg := parseValid(t)

	if pkg.Name != "valid" {
		t.Errorf("Name = %q, want %q", pkg.Name, "valid")
	}
	if pkg.GoPkg == nil || pkg.GoPkg.TypesInfo == nil {
		t.Fatal("GoPkg with type info should be handed over to the resolver")
	}
}

func TestParse_API(t *testing.T) {
	api := parseValid(t).API
	if api == nil {
		t.Fatal("API is nil")
	}

	if api.Title != "Parser Fixture API" {
		t.Errorf("Title = %q", api.Title)
	}
	if api.Version != "2.1.0" {
		t.Errorf("Version = %q", api.Version)
	}
	wantDesc := "Fixture package covering every annotation the parser reads.\nThis second line continues the description."
	if api.Description != wantDesc {
		t.Errorf("Description = %q, want %q", api.Description, wantDesc)
	}
	if api.TermsOfService != "https://example.com/terms" {
		t.Errorf("TermsOfService = %q", api.TermsOfService)
	}
	if api.DefaultContentType != "application/json" {
		t.Errorf("DefaultContentType = %q, want the MIME expansion", api.DefaultContentType)
	}

	if api.Contact == nil {
		t.Fatal("Contact is nil")
	}
	if api.Contact.Name != "API Team" || api.Contact.Email != "support@example.com" || api.Contact.URL != "https://example.com/support" {
		t.Errorf("Contact = %+v", api.Contact)
	}

	if api.License == nil {
		t.Fatal("License is nil")
	}
	if api.License.Name != "MIT" || api.License.URL != "https://opensource.org/licenses/MIT" {
		t.Errorf("License = %+v", api.License)
	}

	if len(api.Servers) != 2 {
		t.Fatalf("got %d servers, want 2", len(api.Servers))
	}
	if api.Servers[0].URL != "https://api.example.com/v1" || api.Servers[0].Description != "Production" {
		t.Errorf("Servers[0] = %+v", api.Servers[0])
	}
	if api.Servers[1].URL != "https://staging.example.com/v1" {
		t.Errorf("Servers[1] = %+v", api.Servers[1])
	}

	if len(api.SecuritySchemes) != 2 {
		t.Fatalf("got %d security schemes, want 2", len(api.SecuritySchemes))
	}
	bearer := api.SecuritySchemes[0]
	if bearer.Name != "bearerAuth" || bearer.Type != "http" || bearer.Scheme != "bearer" || bearer.BearerFormat != "JWT" {
		t.Errorf("SecuritySchemes[0] = %+v", bearer)
	}
	if bearer.Description != "JWT bearer token" {
		t.Errorf("SecuritySchemes[0].Description = %q", bearer.Description)
	}
	apiKey := api.SecuritySchemes[1]
	if apiKey.Name != "apiKeyAuth" || apiKey.Type != "apiKey" || apiKey.In != "header" || apiKey.ParameterName != "X-API-Key" {
		t.Errorf("SecuritySchemes[1] = %+v", apiKey)
	}

	if len(api.Security) != 2 {
		t.Fatalf("got %d security groups, want 2", len(api.Security))
	}
	if len(api.Security[0]) != 1 || api.Security[0][0].SchemeName != "bearerAuth" {
		t.Fatalf("Security[0] = %+v", api.Security[0])
	}
	if got := api.Security[0][0].Scopes; len(got) != 2 || got[0] != "read:users" || got[1] != "write:users" {
		t.Errorf("Security[0] scopes = %v", got)
	}
	if len(api.Security[1]) != 1 || api.Security[1][0].SchemeName != "apiKeyAuth" {
		t.Errorf("Security[1] = %+v", api.Security[1])
	}
	if len(api.Security[1][0].Scopes) != 0 {
		t.Errorf("Security[1] scopes = %v, want none", api.Security[1][0].Scopes)
	}

	if len(api.Tags) != 2 {
		t.Fatalf("got %d tags, want 2", len(api.Tags))
	}
	if api.Tags[0].Name != "users" || api.Tags[0].Description != "User operations" {
		t.Errorf("Tags[0] = %+v", api.Tags[0])
	}
	if api.Tags[1].Name != "admin" || api.Tags[1].Description != "" {
		t.Errorf("Tags[1] = %+v, want a bare tag", api.Tags[1])
	}
}

func TestParse_MissingAPI(t *testing.T) {
	_, err := Parse("./testdata/noapi")
	if err == nil {
		t.Fatal("Parse() succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "missing @api annotation") {
		t.Errorf("error = %v, want it to mention the missing @api", err)
	}
}

func schemaNamed(pkg *Package, name string) *Schema {
	for _, s := range pkg.Schemas {
		if s.Name == name {
			return s
		}
	}
	return nil
}

func fieldNamed(fields []*Field, goName string) *Field {
	for _, f := range fields {
		if f.GoName == goName {
			return f
		}
	}
	return nil
}

func TestParse_Schemas(t *testing.T) {
	pkg := parseValid(t)

	user := schemaNamed(pkg, "User")
	if user == nil {
		t.Fatal("User schema missing")
	}
	wantDesc := "A user of the system.\nSecond line of the schema description."
	if user.Description != wantDesc {
		t.Errorf("User.Description = %q, want %q", user.Description, wantDesc)
	}
	if user.Deprecated {
		t.Error("User should not be deprecated")
	}
	if user.Pos.Line == 0 || !strings.HasSuffix(user.Pos.Filename, "valid.go") {
		t.Errorf("User.Pos = %v, want the declaration position", user.Pos)
	}

	legacy := schemaNamed(pkg, "LegacyUser")
	if legacy == nil || !legacy.Deprecated {
		t.Errorf("LegacyUser deprecated = %v, want true", legacy)
	}

	// Only struct declarations and type aliases can carry @schema.
	if schemaNamed(pkg, "Status") != nil {
		t.Error("Status is not a struct, so it must not become a schema")
	}
}

func TestParse_SchemaFieldsAreDeclarationOrdered(t *testing.T) {
	user := schemaNamed(parseValid(t), "User")

	want := []string{"ID", "Email", "Username", "Role", "Age", "Labels", "Password", "Legacy", "Nickname"}
	if len(user.Fields) != len(want) {
		t.Fatalf("got %d fields %v, want %d", len(user.Fields), fieldNames(user.Fields), len(want))
	}
	for i, name := range want {
		if user.Fields[i].GoName != name {
			t.Errorf("field %d = %q, want %q", i, user.Fields[i].GoName, name)
		}
	}

	// Unannotated fields produce no entry: the resolver walks the Go struct
	// for the full list and matches these by GoName.
	if fieldNamed(user.Fields, "Internal") != nil {
		t.Error("unannotated field Internal should have no Field entry")
	}
}

func TestParse_AnonymousStructFields(t *testing.T) {
	anonymous := schemaNamed(parseValid(t), "Anonymous")
	if anonymous == nil {
		t.Fatal("Anonymous schema missing")
	}

	// Declaration order holds at both levels, and a field with nothing to say
	// about itself still appears to carry its nested annotations.
	if got := fieldNames(anonymous.Fields); !reflect.DeepEqual(got, []string{"Address", "Untagged", "Note"}) {
		t.Fatalf("fields = %v", got)
	}

	address := fieldNamed(anonymous.Fields, "Address")
	if address.Description != "The shipping address" {
		t.Errorf("Address description = %q", address.Description)
	}
	if got := fieldNames(address.Fields); !reflect.DeepEqual(got, []string{"Street", "Country"}) {
		t.Errorf("Address nested fields = %v", got)
	}
	if got := fieldNamed(address.Fields, "Country").Description; got != "Two-letter country code" {
		t.Errorf("Address.Country description = %q", got)
	}

	// The element struct of a slice is reached the same way.
	untagged := fieldNamed(anonymous.Fields, "Untagged")
	if untagged.Description != "" {
		t.Errorf("Untagged should carry no annotation of its own, got %q", untagged.Description)
	}
	quantity := fieldNamed(untagged.Fields, "Quantity")
	if quantity == nil || quantity.Minimum == nil || *quantity.Minimum != 1 {
		t.Errorf("Untagged.Quantity = %+v, want @minimum 1", quantity)
	}

	if got := fieldNamed(anonymous.Fields, "Note").Fields; got != nil {
		t.Errorf("a field with no anonymous struct should carry no nested fields, got %v", fieldNames(got))
	}
}

func fieldNames(fields []*Field) []string {
	names := make([]string, len(fields))
	for i, f := range fields {
		names[i] = f.GoName
	}
	return names
}

func TestParse_FieldConstraints(t *testing.T) {
	user := schemaNamed(parseValid(t), "User")

	id := fieldNamed(user.Fields, "ID")
	if id.Description != "User ID" || id.Format != "uuid" || !id.ReadOnly {
		t.Errorf("ID = %+v", id)
	}
	if id.Pos.Line == 0 {
		t.Error("field position not recorded")
	}

	email := fieldNamed(user.Fields, "Email")
	if email.Example != "alice@example.com" {
		t.Errorf("Email.Example = %q, want the unescaped address", email.Example)
	}

	username := fieldNamed(user.Fields, "Username")
	if username.MinLength == nil || *username.MinLength != 3 {
		t.Errorf("Username.MinLength = %v, want 3", username.MinLength)
	}
	if username.MaxLength == nil || *username.MaxLength != 32 {
		t.Errorf("Username.MaxLength = %v, want 32", username.MaxLength)
	}
	if username.Pattern != "^[a-z]{3,32}$" {
		t.Errorf("Username.Pattern = %q, want the regex verbatim", username.Pattern)
	}

	role := fieldNamed(user.Fields, "Role")
	if got := role.Enum; len(got) != 3 || got[0] != "admin" || got[1] != "user" || got[2] != "guest" {
		t.Errorf("Role.Enum = %v, want trimmed values", got)
	}
	if role.Default != "user" {
		t.Errorf("Role.Default = %q", role.Default)
	}

	age := fieldNamed(user.Fields, "Age")
	for _, tc := range []struct {
		name string
		got  *float64
		want float64
	}{
		{"Minimum", age.Minimum, 0},
		{"Maximum", age.Maximum, 130},
		{"ExclusiveMinimum", age.ExclusiveMinimum, 0.5},
		{"ExclusiveMaximum", age.ExclusiveMaximum, 129.5},
	} {
		if tc.got == nil || *tc.got != tc.want {
			t.Errorf("Age.%s = %v, want %v", tc.name, tc.got, tc.want)
		}
	}

	labels := fieldNamed(user.Fields, "Labels")
	if labels.MinItems == nil || *labels.MinItems != 1 {
		t.Errorf("Labels.MinItems = %v, want 1", labels.MinItems)
	}
	if labels.MaxItems == nil || *labels.MaxItems != 10 {
		t.Errorf("Labels.MaxItems = %v, want 10", labels.MaxItems)
	}
	if !labels.UniqueItems {
		t.Error("Labels.UniqueItems = false, want true")
	}

	if !fieldNamed(user.Fields, "Password").WriteOnly {
		t.Error("Password.WriteOnly = false, want true")
	}
	if !fieldNamed(user.Fields, "Legacy").Deprecated {
		t.Error("Legacy.Deprecated = false, want true")
	}
}

func TestParse_FieldOverridesAreTriState(t *testing.T) {
	user := schemaNamed(parseValid(t), "User")

	nickname := fieldNamed(user.Fields, "Nickname")
	if nickname.Required == nil || *nickname.Required {
		t.Errorf("Nickname.Required = %v, want an explicit false", nickname.Required)
	}
	if nickname.Nullable == nil || !*nickname.Nullable {
		t.Errorf("Nickname.Nullable = %v, want an explicit true", nickname.Nullable)
	}

	// A field that says nothing leaves both nil, so the resolver decides.
	id := fieldNamed(user.Fields, "ID")
	if id.Required != nil || id.Nullable != nil {
		t.Errorf("ID Required/Nullable = %v/%v, want nil (no override)", id.Required, id.Nullable)
	}
}

func TestParse_EscapedAndRawValues(t *testing.T) {
	escapes := schemaNamed(parseValid(t), "Escapes")
	if escapes == nil {
		t.Fatal("Escapes schema missing")
	}

	text := fieldNamed(escapes.Fields, "Text")
	if want := "Braces {like this} and an at @sign"; text.Description != want {
		t.Errorf("Text.Description = %q, want %q", text.Description, want)
	}

	phone := fieldNamed(escapes.Fields, "Phone")
	if want := `^\d{3}-\d{4}$`; phone.Pattern != want {
		t.Errorf("Phone.Pattern = %q, want %q", phone.Pattern, want)
	}
}

func TestParse_GenericsAndAliases(t *testing.T) {
	pkg := parseValid(t)

	response := schemaNamed(pkg, "Response")
	if response == nil || !response.IsGeneric {
		t.Fatalf("Response = %+v, want a generic schema", response)
	}

	userResponse := schemaNamed(pkg, "UserResponse")
	if userResponse == nil {
		t.Fatal("UserResponse alias schema missing")
	}
	if !userResponse.IsTypeAlias || userResponse.AliasOf != "Response[User]" {
		t.Errorf("UserResponse = %+v", userResponse)
	}
	if got := userResponse.TypeArgs; len(got) != 1 || got[0] != "User" {
		t.Errorf("UserResponse.TypeArgs = %v, want [User]", got)
	}
	if len(userResponse.Fields) != len(response.Fields) {
		t.Errorf("UserResponse has %d fields, want the %d copied from Response", len(userResponse.Fields), len(response.Fields))
	}
	if userResponse.Fields[0] == response.Fields[0] {
		t.Error("alias fields must be copies, not shared pointers")
	}

	pair := schemaNamed(pkg, "StringUserPair")
	if pair == nil {
		t.Fatal("StringUserPair alias schema missing")
	}
	if got := pair.TypeArgs; len(got) != 2 || got[0] != "string" || got[1] != "User" {
		t.Errorf("StringUserPair.TypeArgs = %v, want [string User]", got)
	}

	// An alias of a non-generic schema is not a schema of its own.
	if schemaNamed(pkg, "UserAlias") != nil {
		t.Error("UserAlias should not produce a derived schema")
	}
}

func TestParse_Parameters(t *testing.T) {
	pkg := parseValid(t)

	byName := map[string]*ParameterStruct{}
	for _, p := range pkg.Parameters {
		byName[p.Name] = p
	}

	for _, tt := range []struct {
		name string
		kind string
	}{
		{"UserPath", ParamPath},
		{"ListQuery", ParamQuery},
		{"RateLimitHeaders", ParamHeader},
		{"SessionCookie", ParamCookie},
		{"EmbeddedQueryParams", ParamQuery},
	} {
		param, ok := byName[tt.name]
		if !ok {
			t.Errorf("parameter struct %s missing", tt.name)
			continue
		}
		if param.Kind != tt.kind {
			t.Errorf("%s.Kind = %q, want %q", tt.name, param.Kind, tt.kind)
		}
	}

	// Parameter fields go through the same @field path as schema fields.
	limit := fieldNamed(byName["ListQuery"].Fields, "Limit")
	if limit == nil {
		t.Fatal("ListQuery.Limit missing")
	}
	if limit.Minimum == nil || *limit.Minimum != 1 || limit.Default != "20" {
		t.Errorf("ListQuery.Limit = %+v", limit)
	}
	if fieldNamed(byName["ListQuery"].Fields, "Cursor") != nil {
		t.Error("unannotated parameter field should have no Field entry")
	}
}

func endpointNamed(pkg *Package, funcName string) *Endpoint {
	for _, e := range pkg.Endpoints {
		if e.FuncName == funcName {
			return e
		}
	}
	return nil
}

func TestParse_Endpoint(t *testing.T) {
	endpoint := endpointNamed(parseValid(t), "GetUser")
	if endpoint == nil {
		t.Fatal("GetUser endpoint missing")
	}

	if endpoint.Method != "GET" || endpoint.Path != "/users/{id}" {
		t.Errorf("method/path = %q %q", endpoint.Method, endpoint.Path)
	}
	if endpoint.OperationID != "getUser" || endpoint.Summary != "Get a user" {
		t.Errorf("operationID/summary = %q %q", endpoint.OperationID, endpoint.Summary)
	}
	wantDesc := "Returns a single user.\nThis second line continues the description."
	if endpoint.Description != wantDesc {
		t.Errorf("Description = %q, want %q", endpoint.Description, wantDesc)
	}
	if endpoint.Auth != "bearerAuth" {
		t.Errorf("Auth = %q", endpoint.Auth)
	}
	if got := endpoint.Tags; len(got) != 2 || got[0] != "users" || got[1] != "admin" {
		t.Errorf("Tags = %v", got)
	}
	if got := endpoint.PathParams; len(got) != 1 || got[0] != "UserPath" {
		t.Errorf("PathParams = %v", got)
	}
	if got := endpoint.QueryParams; len(got) != 1 || got[0] != "ListQuery" {
		t.Errorf("QueryParams = %v", got)
	}
	if got := endpoint.HeaderParams; len(got) != 1 || got[0] != "RateLimitHeaders" {
		t.Errorf("HeaderParams = %v", got)
	}
	if got := endpoint.CookieParams; len(got) != 1 || got[0] != "SessionCookie" {
		t.Errorf("CookieParams = %v", got)
	}

	if len(endpoint.Responses) != 3 {
		t.Fatalf("got %d responses, want 3", len(endpoint.Responses))
	}

	ok := endpoint.Responses[0]
	if ok.Status != "200" || ok.ContentType != "application/json" || ok.Description != "Found" {
		t.Errorf("Responses[0] = %+v", ok)
	}
	if ok.Body == nil || ok.Body.Schema != "User" {
		t.Errorf("Responses[0].Body = %+v", ok.Body)
	}
	if got := ok.Headers; len(got) != 1 || got[0] != "RateLimitHeaders" {
		t.Errorf("Responses[0].Headers = %v", got)
	}

	if endpoint.Responses[1].Status != "404" {
		t.Errorf("Responses[1].Status = %q", endpoint.Responses[1].Status)
	}
	if endpoint.Responses[2].Status != "default" {
		t.Errorf("Responses[2].Status = %q, want the literal default", endpoint.Responses[2].Status)
	}
}

func TestParse_EndpointRequestAndBind(t *testing.T) {
	endpoint := endpointNamed(parseValid(t), "CreateUser")
	if endpoint == nil {
		t.Fatal("CreateUser endpoint missing")
	}

	if !endpoint.Deprecated {
		t.Error("Deprecated = false, want true")
	}
	if endpoint.Request == nil {
		t.Fatal("Request is nil")
	}
	if endpoint.Request.ContentType != "application/xml" {
		t.Errorf("Request.ContentType = %q, want the MIME expansion", endpoint.Request.ContentType)
	}
	if endpoint.Request.Body == nil || endpoint.Request.Body.Schema != "User" {
		t.Fatalf("Request.Body = %+v", endpoint.Request.Body)
	}
	bind := endpoint.Request.Body.Bind
	if bind == nil || bind.Wrapper != "Envelope" || bind.Field != "Data" {
		t.Errorf("Request.Body.Bind = %+v", bind)
	}

	created := endpoint.Responses[0]
	if created.Status != "201" || created.Body == nil || created.Body.Schema != "[]User" {
		t.Errorf("Responses[0] = %+v", created)
	}

	noContent := endpoint.Responses[1]
	if noContent.Status != "204" || noContent.ContentType != "" {
		t.Errorf("Responses[1] = %+v, want the empty content type", noContent)
	}
	if noContent.Body != nil {
		t.Errorf("Responses[1].Body = %+v, want nil", noContent.Body)
	}
}

func TestParse_RepeatedStatusIsLastWins(t *testing.T) {
	endpoint := endpointNamed(parseValid(t), "RepeatStatus")
	if endpoint == nil {
		t.Fatal("RepeatStatus endpoint missing")
	}

	if len(endpoint.Responses) != 1 {
		t.Fatalf("got %d responses, want 1", len(endpoint.Responses))
	}
	if got := endpoint.Responses[0].Description; got != "second" {
		t.Errorf("description = %q, want the last declaration to win", got)
	}
}

func TestParse_InlineDeclarations(t *testing.T) {
	endpoint := endpointNamed(parseValid(t), "InlineHandler")
	if endpoint == nil {
		t.Fatal("InlineHandler endpoint missing")
	}
	inline := endpoint.Inline
	if inline == nil {
		t.Fatal("Inline is nil")
	}

	if len(inline.Path) != 1 || inline.Path[0].VarName != "path" {
		t.Fatalf("Path = %+v", inline.Path)
	}
	if got := inline.Path[0].Fields; len(got) != 1 || got[0].Format != "uuid" {
		t.Errorf("path fields = %+v", got)
	}
	if inline.Path[0].Struct == nil || inline.Path[0].Struct.NumFields() != 1 {
		t.Errorf("path struct type not resolved: %v", inline.Path[0].Struct)
	}
	if inline.Path[0].Pos.Line == 0 {
		t.Error("inline declaration position not recorded")
	}

	// Repeatable categories keep every declaration, in declaration order.
	if len(inline.Query) != 2 || inline.Query[0].VarName != "filters" || inline.Query[1].VarName != "paging" {
		t.Errorf("Query = %+v", inline.Query)
	}
	if len(inline.Header) != 1 || inline.Header[0].VarName != "headers" {
		t.Errorf("Header = %+v", inline.Header)
	}
	if len(inline.Cookie) != 1 || inline.Cookie[0].VarName != "cookies" {
		t.Errorf("Cookie = %+v", inline.Cookie)
	}

	if inline.Request == nil {
		t.Fatal("Request is nil")
	}
	if inline.Request.ContentType != "application/xml" {
		t.Errorf("Request.ContentType = %q", inline.Request.ContentType)
	}
	if inline.Request.Description != "Order payload" {
		t.Errorf("Request.Description = %q", inline.Request.Description)
	}
	if bind := inline.Request.Bind; bind == nil || bind.Wrapper != "Envelope" || bind.Field != "Data" {
		t.Errorf("Request.Bind = %+v", bind)
	}
	if got := inline.Request.Fields; len(got) != 1 || got[0].GoName != "CustomerID" {
		t.Errorf("Request.Fields = %+v", got)
	}
}

func TestParse_InlineResponseStatuses(t *testing.T) {
	inline := endpointNamed(parseValid(t), "InlineHandler").Inline

	want := []struct {
		status  string
		varName string
	}{
		{"201", "created"},
		{"4XX", "clientError"},
		{"default", "fallback"},
		{"200", "unspecified"},
		// The standalone response declares nothing, so it has no var name.
		{"404", ""},
	}

	if len(inline.Responses) != len(want) {
		t.Fatalf("got %d inline responses, want %d", len(inline.Responses), len(want))
	}
	for i, tt := range want {
		got := inline.Responses[i]
		if got.Status != tt.status || got.VarName != tt.varName {
			t.Errorf("Responses[%d] = %s/%s, want %s/%s", i, got.Status, got.VarName, tt.status, tt.varName)
		}
	}

	clientError := inline.Responses[1]
	if clientError.ContentType != "application/json" || clientError.Description != "Client error" {
		t.Errorf("4XX response = %+v", clientError.InlineStruct)
	}
	if got := clientError.Headers; len(got) != 1 || got[0] != "RateLimitHeaders" {
		t.Errorf("4XX headers = %v", got)
	}
}

func TestParse_StandaloneResponse(t *testing.T) {
	inline := endpointNamed(parseValid(t), "InlineHandler").Inline

	var standalone *InlineResponse
	for _, response := range inline.Responses {
		if response.Status == "404" {
			standalone = response
		}
	}
	if standalone == nil {
		t.Fatal("the standalone @response was not parsed")
	}

	// It names its body instead of declaring a struct to be one.
	if standalone.Struct != nil {
		t.Error("a standalone response declares no struct")
	}
	if standalone.Body == nil || standalone.Body.Schema != "Error" {
		t.Fatalf("body = %+v, want a reference to Error", standalone.Body)
	}

	// The rest of the block reads exactly as it does on an endpoint response.
	if standalone.Description != "Order not found" {
		t.Errorf("description = %q", standalone.Description)
	}
	if got := standalone.Headers; len(got) != 1 || got[0] != "RateLimitHeaders" {
		t.Errorf("headers = %v", got)
	}
	if standalone.Pos.Line == 0 || !strings.HasSuffix(standalone.Pos.Filename, "valid.go") {
		t.Errorf("Pos = %v, want the annotation position", standalone.Pos)
	}
}

func TestParse_InlineInClosure(t *testing.T) {
	inline := endpointNamed(parseValid(t), "ClosureHandler").Inline
	if inline == nil {
		t.Fatal("Inline is nil")
	}

	if inline.Request == nil || inline.Request.VarName != "request" {
		t.Fatalf("Request = %+v, want the declaration from the factory body", inline.Request)
	}
	if len(inline.Responses) != 1 {
		t.Fatalf("got %d responses, want the one from the returned closure", len(inline.Responses))
	}
	if inline.Responses[0].VarName != "response" || inline.Responses[0].Status != "200" {
		t.Errorf("Responses[0] = %+v", inline.Responses[0])
	}
	if got := inline.Responses[0].Fields; len(got) != 1 || got[0].Description != "Greeting" {
		t.Errorf("response fields = %+v", got)
	}
}

func TestParse_InlineOutsideEndpointIsIgnored(t *testing.T) {
	pkg := parseValid(t)

	if endpointNamed(pkg, "helper") != nil {
		t.Error("a function without @endpoint must not become an endpoint")
	}
	for _, endpoint := range pkg.Endpoints {
		if endpoint.Inline == nil {
			continue
		}
		for _, query := range endpoint.Inline.Query {
			if query.VarName == "ignored" {
				t.Errorf("%s picked up an inline declaration from another function", endpoint.FuncName)
			}
		}
	}
}

func TestParse_AccumulatesErrorsPerItem(t *testing.T) {
	_, err := Parse("./testdata/broken")
	if err == nil {
		t.Fatal("Parse() succeeded, want errors")
	}

	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("error %T does not join multiple errors", err)
	}
	errs := joined.Unwrap()
	if len(errs) != 6 {
		t.Fatalf("got %d errors, want one per broken item:\n%v", len(errs), err)
	}

	wants := []string{
		`invalid @field for BadNumber.Count: @minimum value "abc" is not a valid number`,
		`invalid @field for BadBool.Name: @required value "maybe" is not a valid boolean`,
		"unknown annotation @summry in @endpoint",
		`in function DuplicateRequest: duplicate inline @request on "second" (previous: "first")`,
		"in function BodylessStandalone: standalone @response 400 must name its body with @body",
		// The declared struct is the body, so naming a second one is an error.
		`in function TwoBodies: failed to parse inline @response on "resp": failed to parse @response children: unknown annotation @body in @response`,
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %q, want it to contain %q", i, errs[i], want)
		}

		var perr *Error
		if !errors.As(errs[i], &perr) {
			t.Fatalf("error %d is not a *parser.Error: %v", i, errs[i])
		}
		if !strings.HasSuffix(perr.Pos.Filename, "broken.go") || perr.Pos.Line == 0 {
			t.Errorf("error %d position = %v, want a broken.go line", i, perr.Pos)
		}
		if !strings.HasPrefix(errs[i].Error(), perr.Pos.Filename+":") {
			t.Errorf("error %d = %q, want it to start with the position", i, errs[i])
		}
	}
}

func TestParse_FloatingAnnotationIsAnError(t *testing.T) {
	_, err := Parse("./testdata/floating")
	if err == nil {
		t.Fatal("an annotation attached to no declaration should be an error, not a silent no-op")
	}
	if !strings.Contains(err.Error(), "@field is attached to no declaration") {
		t.Errorf("error = %v, want it to name the stray annotation", err)
	}

	var perr *Error
	if !errors.As(err, &perr) {
		t.Fatalf("error is not a *parser.Error: %v", err)
	}
	if !strings.HasSuffix(perr.Pos.Filename, "floating.go") || perr.Pos.Line == 0 {
		t.Errorf("position = %v, want the annotation's line", perr.Pos)
	}
}

func TestParseBindTarget(t *testing.T) {
	tests := []struct {
		value       string
		wantWrapper string
		wantField   string
		wantNil     bool
	}{
		{value: "DataResponse.Data", wantWrapper: "DataResponse", wantField: "Data"},
		{value: "  Envelope . Payload  ", wantWrapper: "Envelope", wantField: "Payload"},
		{value: "Nested.Field.Deep", wantWrapper: "Nested", wantField: "Field.Deep"},
		{value: "NoDot", wantNil: true},
		{value: "", wantNil: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := ParseBindTarget(tt.value)
			if tt.wantNil {
				if got != nil {
					t.Fatalf("ParseBindTarget(%q) = %+v, want nil", tt.value, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("ParseBindTarget(%q) = nil", tt.value)
			}
			if got.Wrapper != tt.wantWrapper || got.Field != tt.wantField {
				t.Errorf("ParseBindTarget(%q) = %+v", tt.value, got)
			}
		})
	}
}

func TestExpandContentType(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"json", "application/json"},
		{"JSON", "application/json"},
		{"xml", "application/xml"},
		{"form", "application/x-www-form-urlencoded"},
		{"multipart", "multipart/form-data"},
		{"text", "text/plain"},
		{"csv", "text/csv"},
		{"binary", "application/octet-stream"},
		{"html", "text/html"},
		{"empty", ""},
		{"", ""},
		{"application/vnd.api+json", "application/vnd.api+json"},
		{"custom", "custom"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ExpandContentType(tt.input); got != tt.want {
				t.Errorf("ExpandContentType(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestValidStatusCode(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"200", true},
		{"404", true},
		{"599", true},
		{"2XX", true},
		{"default", true},
		{"600", false},
		{"20", false},
		{"2xx", false},
		{"abc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := ValidStatusCode(tt.status); got != tt.want {
				t.Errorf("ValidStatusCode(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
