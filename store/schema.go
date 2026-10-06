package store

// Schema mirrors StandardSchemaV1 (only `~standard.validate` is used).
// Validate returns (result, nil) for synchronous validation; a non-nil
// promise mirrors a validate() that returns a Promise, which ValidateSchemas
// rejects with reason ReasonAsyncValidationUnsupported.
type Schema interface {
	Validate(value any) (SchemaResult, *Promise[SchemaResult])
}

// SchemaResult mirrors StandardSchemaV1.Result: success when Issues is empty.
type SchemaResult struct {
	Value  any
	Issues []SchemaIssue
}

// SchemaIssue mirrors StandardSchemaV1.Issue.
type SchemaIssue struct {
	Message string
	Path    []any
}

// SchemaFunc adapts a function to Schema.
type SchemaFunc func(value any) (SchemaResult, *Promise[SchemaResult])

func (f SchemaFunc) Validate(value any) (SchemaResult, *Promise[SchemaResult]) { return f(value) }
