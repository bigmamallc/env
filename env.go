package env

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
)

type configType string

const (
	configTypeEnvironment configType = "environment"
)

// Setter is called for any complex struct field with an
// implementation, allowing developers to override Set
// behaviour.
type Setter interface {
	Set(string) error
}

// SetWithEnvPrefix sets the fields of a struct from environment config.
// If a field is unexported or required configuration is not
// found, an error will be returned.
func SetWithEnvPrefix(i interface{}, envPrefix string) (err error) {
	return setWithEnvPrefix(i, envPrefix, false)
}

func SetDefaultOnly(i interface{}, envPrefix string) (err error) {
	return setWithEnvPrefix(i, envPrefix, true)
}

func setWithEnvPrefix(i interface{}, envPrefix string, ignoreMissing bool) (err error) {
	v := reflect.ValueOf(i)

	// Don't try to process a non-pointer value.
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return fmt.Errorf("%s is not a pointer", v.Kind())
	}

	v = v.Elem()
	t := reflect.TypeOf(i).Elem()

	for i := 0; i < t.NumField(); i++ {
		if err = processFieldIgnoreMissing(t.Field(i), v.Field(i), envPrefix, ignoreMissing); err != nil {
			return
		}
	}

	return
}

func Set(i interface{}) (err error) {
	return SetWithEnvPrefix(i, "")
}

// processField will lookup the "env" tag for the property
// and attempt to set it.  If not found, another check for the
// "required" tag will be performed to decided whether an error
// needs to be returned.
func processField(t reflect.StructField, v reflect.Value, envPrefix string) (err error) {
	return processFieldIgnoreMissing(t, v, envPrefix, false)
}

func processFieldIgnoreMissing(t reflect.StructField, v reflect.Value, envPrefix string, ignoreMissing bool) (err error) {
	envTag, ok := t.Tag.Lookup("env")
	if !ok {
		return
	}

	if v.Kind() == reflect.Struct {
		if err = setWithEnvPrefix(v.Addr().Interface(), envPrefix + envTag, ignoreMissing); err != nil {
			return fmt.Errorf("failed to set nested struct: %w", err)
		}
		return nil
	}

	// If the field is unexported or just not settable, bail at
	// this point because subsequent operations will fail.
	if !v.CanSet() {
		return fmt.Errorf("field '%s' cannot be set", t.Name)
	}

	// Lookup the environment variable and if found, set and
	// return
	env, ok := os.LookupEnv(envPrefix + envTag)
	if ok {
		return setField(t, v, env, envPrefix, ignoreMissing)
	}

	// If the value isn't found in the environment, look for a
	// user-defined default value
	d, ok := t.Tag.Lookup("default")
	if ok {
		return setField(t, v, d, envPrefix, ignoreMissing)
	}

	if !ignoreMissing {
		// An env tag has been provided but a matching environment
		// variable cannot be found, determine if we should return
		// an error or if a missing variable is ok/expected.
		return processMissing(t, envTag, configTypeEnvironment)
	} else {
		return nil
	}
}

func setField(t reflect.StructField, v reflect.Value, value string, envPrefix string, ignoreMissing bool) (err error) {
	// If field implements the Setter interface, invoke it now and
	// don't continue attempting to set the primitive values.
	if _, ok := v.Interface().(Setter); ok {
		instance := reflect.New(t.Type.Elem())
		v.Set(instance)

		// Re-assert the type with the newed-up instance and call.
		setter := v.Interface().(Setter)
		if err = setter.Set(value); err != nil {
			return fmt.Errorf("error in custom setter: %v", err)
		}
		return
	}

	// If the given type is a slice, create a slice and return,
	// otherwise, we're dealing with a primitive type
	if v.Kind() == reflect.Slice {
		return setSlice(t, v, value)
	}

	if err = setBuiltInField(v, value); err != nil {
		return fmt.Errorf("error setting %q: %v", t.Name, err)
	}

	return
}

// ProcessMissing returns an error if a required tag is found
// and is set to true.  A different error will be returned if
// the required tag was present but the value could not be parsed
// to a Boolean value.
func processMissing(t reflect.StructField, envTag string, ct configType) (err error) {
	reqTag, ok := t.Tag.Lookup("required")
	if !ok {
		// No required tag was found, this field doesn't expect
		// an env tag to be provided.
		return nil
	}

	var b bool
	if b, err = strconv.ParseBool(reqTag); err != nil {
		// The value provided for the required tag is not a valid
		// Boolean, so inform the user.
		return fmt.Errorf("invalid required tag %q: %v", reqTag, err)
	}

	if b {
		// The value provided for the required tag is valid and is
		// set to true, so the user needs to know that a required
		// environment variable could not be found.
		return fmt.Errorf("%s %s configuration was missing", envTag, ct)
	}

	return
}
