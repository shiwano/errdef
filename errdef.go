package errdef

import "errors"

// Define creates a new error definition with the specified kind and options.
//
// NOTE:
// The error identity check performed by errors.Is relies on an internal identifier
// associated with each Definition instance, not on the string value of Kind.
//
// While this means that using the same Kind string for different definitions
// will not cause incorrect identity checks, it is strongly recommended to use
// a unique Kind value across your application to prevent confusion in logs and
// monitoring tools.
func Define(kind Kind, opts ...Option) Definition {
	def := &definition{
		kind:   kind,
		fields: newFields(),
	}
	def.applyOptions(opts)
	return def
}

// DefineField creates a field option constructor and extractor for the given field name.
// The constructor can be used to set a field value in error options,
// and the extractor can be used to retrieve the field value from errors.
//
// NOTE:
// The identity of a field is determined by the returned constructor and extractor
// instances, not by the provided name string. This ensures that fields created
// by different calls to DefineField, even with the same name, are distinct
// and do not collide.
//
// The name string is used as the key when an error's fields are serialized
// (e.g., to JSON). To avoid ambiguity in logs and other serialized representations,
// it is strongly recommended to use a unique name for each defined field.
func DefineField[T any](name string) (FieldConstructor[T], FieldExtractor[T]) {
	k := &fieldKey[T]{name: name}
	ctor := func(value T) Option {
		return &field[T]{key: k, value: value}
	}
	extr := func(err error) (T, bool) {
		return fieldValueFrom[T](err, k)
	}
	return ctor, extr
}

// KindFrom extracts the Kind from an error.
// It returns the Kind and true if the error implements the Kind() method.
// Otherwise, it returns an empty Kind and false.
func KindFrom(err error) (Kind, bool) {
	if err == nil {
		return "", false
	}
	var e kindGetter
	if ok := errors.As(err, &e); !ok {
		return "", false
	}
	return e.Kind(), true
}

// FieldsFrom extracts the Fields from an error.
// It returns the Fields and true if the error implements the Fields() method and
// the Fields are non-empty. Otherwise, it returns nil and false.
func FieldsFrom(err error) (Fields, bool) {
	if err == nil {
		return nil, false
	}
	var e fieldsGetter
	if ok := errors.As(err, &e); !ok {
		return nil, false
	}
	fields := e.Fields()
	if fields.IsZero() {
		return nil, false
	}
	return fields, true
}

// StackFrom extracts the Stack from an error.
// It returns the Stack and true if the error implements the Stack() method and
// the Stack is non-empty. Otherwise, it returns nil and false.
func StackFrom(err error) (Stack, bool) {
	if err == nil {
		return nil, false
	}
	var e stackGetter
	if ok := errors.As(err, &e); !ok {
		return nil, false
	}
	stack := e.Stack()
	if stack.Len() == 0 {
		return nil, false
	}
	return stack, true
}

// UnwrapTreeFrom extracts the error cause tree from an error.
// It returns the Nodes and true if the error implements the UnwrapTree() method
// and the returned Nodes are non-empty. Otherwise, it returns nil and false.
func UnwrapTreeFrom(err error) (Nodes, bool) {
	if err == nil {
		return nil, false
	}
	var e treeUnwrapper
	if ok := errors.As(err, &e); !ok {
		return nil, false
	}
	causes := e.UnwrapTree()
	if len(causes) == 0 {
		return nil, false
	}
	return causes, true
}
