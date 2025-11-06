package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/Infra-Forge/apix"
	"github.com/getkin/kin-openapi/openapi3"
)

// Builder converts registered routes into an OpenAPI 3.1 document.
type Builder struct {
	Info            openapi3.Info
	Servers         openapi3.Servers
	SecuritySchemes openapi3.SecuritySchemes
	GlobalSecurity  openapi3.SecurityRequirements
	Tags            openapi3.Tags

	doc         *openapi3.T
	schemaCache map[reflect.Type]*openapi3.SchemaRef
}

func NewBuilder() *Builder {
	return &Builder{
		Info: openapi3.Info{
			Title:   "API",
			Version: "1.0.0",
		},
	}
}

// Build transforms the route registry snapshot into an OpenAPI document.
func (b *Builder) Build(routes []*apix.RouteRef) (*openapi3.T, error) {
	paths := openapi3.NewPaths()
	components := &openapi3.Components{
		Schemas:         map[string]*openapi3.SchemaRef{},
		SecuritySchemes: b.SecuritySchemes,
	}
	doc := &openapi3.T{
		OpenAPI:    "3.1.0",
		Info:       &b.Info,
		Servers:    b.Servers,
		Components: components,
		Paths:      paths,
		Tags:       b.Tags,
	}

	if len(b.GlobalSecurity) > 0 {
		doc.Security = b.GlobalSecurity
	}

	b.doc = doc
	b.schemaCache = make(map[reflect.Type]*openapi3.SchemaRef)
	defer func() {
		b.doc = nil
		b.schemaCache = nil
	}()

	for _, route := range routes {
		if err := b.addRoute(doc, route); err != nil {
			return nil, err
		}
	}

	sortPaths(doc.Paths)
	return doc, nil
}

func (b *Builder) addRoute(doc *openapi3.T, ref *apix.RouteRef) error {
	pathItem := doc.Paths.Value(ref.Path)
	if pathItem == nil {
		pathItem = &openapi3.PathItem{}
		doc.Paths.Set(ref.Path, pathItem)
	}

	op := openapi3.NewOperation()
	op.OperationID = ref.OperationID
	op.Summary = ref.Summary
	op.Description = ref.Description
	op.Deprecated = ref.Deprecated
	op.Tags = ref.Tags

	if len(ref.Parameters) > 0 {
		params := append([]apix.Parameter(nil), ref.Parameters...)
		sort.SliceStable(params, func(i, j int) bool {
			if params[i].In == params[j].In {
				return params[i].Name < params[j].Name
			}
			return params[i].In < params[j].In
		})
		op.Parameters = make(openapi3.Parameters, 0, len(params))
		for _, p := range params {
			paramSchema := &openapi3.Schema{}
			schemaType(paramSchema, p.SchemaType)
			if paramSchema.Type == nil || len(*paramSchema.Type) == 0 {
				schemaType(paramSchema, "string")
			}
			param := &openapi3.Parameter{
				Name:        p.Name,
				In:          p.In,
				Description: p.Description,
				Required:    p.Required,
				Schema:      &openapi3.SchemaRef{Value: paramSchema},
			}
			if p.SchemaType == "" {
				schemaType(param.Schema.Value, "string")
			}
			if p.Example != nil {
				param.Example = p.Example
			}
			op.Parameters = append(op.Parameters, &openapi3.ParameterRef{Value: param})
		}
	}

	if len(ref.Security) > 0 {
		security := openapi3.NewSecurityRequirements()
		for _, sec := range ref.Security {
			req := openapi3.SecurityRequirement{}
			req[sec.Name] = sec.Scopes
			security.With(req)
		}
		op.Security = security
	}

	if bodyRef, err := b.buildRequestBody(ref); err != nil {
		return err
	} else if bodyRef != nil {
		op.RequestBody = bodyRef
	}

	statusCodes := make([]int, 0, len(ref.Responses))
	for code := range ref.Responses {
		statusCodes = append(statusCodes, code)
	}
	sort.Ints(statusCodes)

	for _, status := range statusCodes {
		respRef := ref.Responses[status]
		oaResp, err := b.buildResponse(status, respRef)
		if err != nil {
			return err
		}
		op.AddResponse(status, oaResp)
	}

	addDXDefaults(ref, op)

	switch strings.ToUpper(string(ref.Method)) {
	case http.MethodGet:
		pathItem.Get = op
	case http.MethodPost:
		pathItem.Post = op
	case http.MethodPut:
		pathItem.Put = op
	case http.MethodPatch:
		pathItem.Patch = op
	case http.MethodDelete:
		pathItem.Delete = op
	default:
		return fmt.Errorf("unsupported method %s", ref.Method)
	}

	doc.Paths.Set(ref.Path, pathItem)
	return nil
}

func (b *Builder) buildRequestBody(ref *apix.RouteRef) (*openapi3.RequestBodyRef, error) {
	schemaRef, err := b.schemaFromTypes(ref.ExplicitRequestModel, ref.RequestType)
	if err != nil {
		return nil, err
	}
	if schemaRef == nil {
		return nil, nil
	}

	contentType := ref.RequestContentType
	if contentType == "" {
		contentType = "application/json"
	}

	content := openapi3.Content{}
	media := &openapi3.MediaType{Schema: schemaRef}
	if ref.RequestExample != nil {
		media.Example = ref.RequestExample
	}
	content[contentType] = media

	required := ref.BodyRequired
	if !ref.BodyRequired {
		required = ref.RequestType != nil || ref.ExplicitRequestModel != nil
	}

	return &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Required: required,
			Content:  content,
		},
	}, nil
}

func (b *Builder) buildResponse(status int, ref *apix.ResponseRef) (*openapi3.Response, error) {
	schemaRef, err := b.schemaFromTypes(ref.ExplicitModelType, ref.ModelType)
	if err != nil {
		return nil, err
	}

	description := ref.Description
	if description == "" {
		description = defaultResponseDescription(status)
	}

	resp := &openapi3.Response{
		Description: &description,
		Content:     openapi3.Content{},
		Headers:     openapi3.Headers{},
	}

	if schemaRef != nil {
		contentType := ref.ContentType
		if contentType == "" {
			contentType = "application/json"
		}
		media := &openapi3.MediaType{Schema: schemaRef}
		if ref.Example != nil {
			media.Example = ref.Example
		}
		resp.Content[contentType] = media
	}

	for _, hdr := range ref.Headers {
		headerSchema := &openapi3.Schema{}
		schemaType(headerSchema, hdr.SchemaType)
		if headerSchema.Type == nil || len(*headerSchema.Type) == 0 {
			schemaType(headerSchema, "string")
		}
		resp.Headers[hdr.Name] = &openapi3.HeaderRef{
			Value: &openapi3.Header{
				Parameter: openapi3.Parameter{
					Description: hdr.Description,
					Required:    hdr.Required,
					Schema:      &openapi3.SchemaRef{Value: headerSchema},
				},
			},
		}
	}

	return resp, nil
}

func (b *Builder) schemaFromTypes(explicit reflect.Type, inferred reflect.Type) (*openapi3.SchemaRef, error) {
	t := explicit
	if t == nil {
		t = inferred
	}
	if t == nil {
		return nil, nil
	}
	return b.schemaRefFromType(t)
}

func (b *Builder) schemaRefFromType(t reflect.Type) (*openapi3.SchemaRef, error) {
	nullable := false
	for t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}

	baseRef, err := b.schemaRefNonNull(t)
	if err != nil || baseRef == nil {
		return baseRef, err
	}

	if !nullable {
		return baseRef, nil
	}
	return wrapNullable(baseRef), nil
}

func (b *Builder) schemaRefNonNull(t reflect.Type) (*openapi3.SchemaRef, error) {
	if cached, ok := b.schemaCache[t]; ok {
		return cached, nil
	}

	switch t.Kind() {
	case reflect.Bool:
		return schemaRef(openapi3.NewBoolSchema()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		s := openapi3.NewIntegerSchema()
		s.Format = "int32"
		return schemaRef(s), nil
	case reflect.Int64:
		s := openapi3.NewIntegerSchema()
		s.Format = "int64"
		return schemaRef(s), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		s := openapi3.NewIntegerSchema()
		s.Format = "int32"
		return schemaRef(s), nil
	case reflect.Uint64:
		s := openapi3.NewIntegerSchema()
		s.Format = "int64"
		return schemaRef(s), nil
	case reflect.Float32:
		s := openapi3.NewFloat64Schema()
		s.Format = "float"
		return schemaRef(s), nil
	case reflect.Float64:
		s := openapi3.NewFloat64Schema()
		return schemaRef(s), nil
	case reflect.String:
		return schemaRef(openapi3.NewStringSchema()), nil
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			s := openapi3.NewStringSchema()
			s.Format = "byte"
			return schemaRef(s), nil
		}
		fallthrough
	case reflect.Array:
		itemRef, err := b.schemaRefFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		s := openapi3.NewArraySchema()
		s.Items = itemRef
		return schemaRef(s), nil
	case reflect.Map:
		if t.Key().Kind() != reflect.String {
			return nil, fmt.Errorf("unsupported map key type %s", t.Key())
		}
		valueRef, err := b.schemaRefFromType(t.Elem())
		if err != nil {
			return nil, err
		}
		s := openapi3.NewObjectSchema()
		s.AdditionalProperties = openapi3.AdditionalProperties{Schema: valueRef}
		return schemaRef(s), nil
	case reflect.Interface:
		s := openapi3.NewObjectSchema()
		return schemaRef(s), nil
	case reflect.Struct:
		if t.AssignableTo(timeType) {
			s := openapi3.NewStringSchema()
			s.Format = "date-time"
			return schemaRef(s), nil
		}
		if isUUID(t) {
			s := openapi3.NewStringSchema()
			s.Format = "uuid"
			return schemaRef(s), nil
		}
		if isDecimal(t) {
			s := openapi3.NewStringSchema()
			s.Format = "decimal"
			s.Description = "Decimal number represented as string for precision"
			s.Example = "123.45"
			return schemaRef(s), nil
		}
		return b.buildStructSchema(t)
	default:
		return nil, fmt.Errorf("unsupported type %s", t)
	}
}

func (b *Builder) buildStructSchema(t reflect.Type) (*openapi3.SchemaRef, error) {
	name := componentName(t)

	if name != "" {
		if ref, ok := b.schemaCache[t]; ok {
			return ref, nil
		}
		ref := openapi3.NewSchemaRef("#/components/schemas/"+name, nil)
		b.schemaCache[t] = ref

		schema := openapi3.NewObjectSchema()
		schema.Properties = make(map[string]*openapi3.SchemaRef)
		schema.Required = []string{}
		schema.AllOf = nil
		b.doc.Components.Schemas[name] = &openapi3.SchemaRef{Value: schema}

		if err := b.populateStructSchema(schema, t); err != nil {
			return nil, err
		}
		return ref, nil
	}

	// anonymous/inline struct
	schema := openapi3.NewObjectSchema()
	schema.Properties = make(map[string]*openapi3.SchemaRef)
	schema.Required = []string{}
	if err := b.populateStructSchema(schema, t); err != nil {
		return nil, err
	}
	ref := &openapi3.SchemaRef{Value: schema}
	b.schemaCache[t] = ref
	return ref, nil
}

func (b *Builder) populateStructSchema(schema *openapi3.Schema, t reflect.Type) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" && !field.Anonymous {
			continue
		}

		if field.Anonymous {
			ref, err := b.schemaRefFromType(field.Type)
			if err != nil {
				return err
			}
			schema.AllOf = append(schema.AllOf, ref)
			continue
		}

		jsonName, skip := jsonName(field)
		if skip {
			continue
		}
		if schema.Properties == nil {
			schema.Properties = make(map[string]*openapi3.SchemaRef)
		}

		childRef, err := b.schemaRefFromType(field.Type)
		if err != nil {
			return err
		}
		schema.Properties[jsonName] = childRef

		if fieldDescription := field.Tag.Get("description"); fieldDescription != "" {
			ensureSchema(childRef).Description = fieldDescription
		}

		if isFieldRequired(field) {
			schema.Required = append(schema.Required, jsonName)
		}
	}
	return nil
}

func jsonName(field reflect.StructField) (name string, skip bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", true
	}
	if tag == "" {
		return field.Name, false
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return "", true
	}
	if parts[0] != "" {
		return parts[0], false
	}
	return field.Name, false
}

func isFieldRequired(field reflect.StructField) bool {
	tag := field.Tag.Get("json")
	if strings.Contains(tag, "omitempty") {
		if hasRequiredTag(field) {
			return true
		}
		return false
	}

	if hasRequiredTag(field) {
		return true
	}

	switch field.Type.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Map, reflect.Interface:
		return false
	default:
		return true
	}
}

func hasRequiredTag(field reflect.StructField) bool {
	validate := field.Tag.Get("validate")
	if strings.Contains(validate, "required") {
		return true
	}
	binding := field.Tag.Get("binding")
	if strings.Contains(binding, "required") {
		return true
	}
	return false
}

func headerRef(h apix.HeaderRef) *openapi3.HeaderRef {
	schema := &openapi3.Schema{}
	schemaType(schema, h.SchemaType)
	if schema.Type == nil || len(*schema.Type) == 0 {
		schemaType(schema, "string")
	}
	header := &openapi3.Header{}
	header.Description = h.Description
	header.Required = h.Required
	header.Schema = &openapi3.SchemaRef{Value: schema}
	return &openapi3.HeaderRef{Value: header}
}

func addDXDefaults(ref *apix.RouteRef, op *openapi3.Operation) {
	if ref.Method == apix.MethodPost {
		resp := op.Responses.Status(http.StatusCreated)
		if resp != nil && resp.Value != nil {
			if resp.Value.Headers == nil {
				resp.Value.Headers = openapi3.Headers{}
			}
			if _, ok := resp.Value.Headers["Location"]; !ok {
				description := "URI of the newly created resource"
				header := &openapi3.Header{}
				header.Description = description
				header.Required = true
				schema := &openapi3.Schema{}
				schemaType(schema, "string")
				schema.Format = "uri"
				header.Schema = &openapi3.SchemaRef{Value: schema}
				resp.Value.Headers["Location"] = &openapi3.HeaderRef{Value: header}
			}
		}
	}

	if len(ref.Security) > 0 {
		ensureResponse(op, http.StatusUnauthorized, "Unauthorized")
		ensureResponse(op, http.StatusForbidden, "Forbidden")
	}
}

func ensureResponse(op *openapi3.Operation, status int, description string) {
	if existing := op.Responses.Status(status); existing != nil {
		return
	}
	desc := description
	op.Responses.Set(fmt.Sprintf("%d", status), &openapi3.ResponseRef{Value: &openapi3.Response{Description: &desc}})
}

func sortPaths(paths *openapi3.Paths) {
	if paths == nil {
		return
	}
	keys := paths.InMatchingOrder()
	sorted := openapi3.NewPaths()
	for _, k := range keys {
		sorted.Set(k, paths.Value(k))
	}
	*paths = *sorted
}

func schemaRef(schema *openapi3.Schema) *openapi3.SchemaRef {
	return &openapi3.SchemaRef{Value: schema}
}

func wrapNullable(ref *openapi3.SchemaRef) *openapi3.SchemaRef {
	if ref == nil {
		return nil
	}
	if ref.Ref != "" && ref.Value == nil {
		return &openapi3.SchemaRef{
			Value: &openapi3.Schema{
				Nullable: true,
				AllOf:    []*openapi3.SchemaRef{ref},
			},
		}
	}
	if ref.Value == nil {
		return ref
	}
	clone := *ref.Value
	clone.Nullable = true
	return &openapi3.SchemaRef{Value: &clone}
}

func ensureSchema(ref *openapi3.SchemaRef) *openapi3.Schema {
	if ref == nil {
		ref = &openapi3.SchemaRef{Value: openapi3.NewObjectSchema()}
	}
	if ref.Value == nil {
		ref.Value = openapi3.NewObjectSchema()
	}
	return ref.Value
}

func componentName(t reflect.Type) string {
	if t.Name() == "" {
		return ""
	}
	pkg := t.PkgPath()
	if pkg == "" {
		return sanitizeComponentName(t.Name())
	}
	parts := strings.Split(pkg, "/")
	pkgPart := parts[len(parts)-1]
	return sanitizeComponentName(pkgPart + "_" + t.Name())
}

func sanitizeComponentName(name string) string {
	replacer := strings.NewReplacer(
		"-", "_",
		".", "_",
		" ", "_",
		"[", "_",
		"]", "_",
		"<", "_",
		">", "_",
		",", "_",
	)
	return replacer.Replace(name)
}

var timeType = reflect.TypeOf(time.Time{})

func isUUID(t reflect.Type) bool {
	return t.PkgPath() == "github.com/google/uuid" && t.Name() == "UUID"
}

func isDecimal(t reflect.Type) bool {
	return t.PkgPath() == "github.com/shopspring/decimal" && t.Name() == "Decimal"
}

func schemaType(schema *openapi3.Schema, t string) {
	if schema == nil {
		return
	}
	if schema.Type == nil {
		schema.Type = &openapi3.Types{}
	}
	if t != "" {
		*schema.Type = openapi3.Types{t}
	}
}

func defaultResponseDescription(status int) string {
	switch status {
	case http.StatusOK:
		return "OK"
	case http.StatusCreated:
		return "Created"
	case http.StatusAccepted:
		return "Accepted"
	case http.StatusNoContent:
		return "No Content"
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	default:
		return ""
	}
}
