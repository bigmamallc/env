package env

import (
	"os"
	"testing"
)

func TestNested(t *testing.T) {
	type EnvA struct {
		Foo string `env:"A_FOO" default:"bar"`
	}
	type EnvB struct {
		Flat   int  `env:"B_FLAT" default:"42"`
		Nested EnvA `env:"_NESTED_"`
	}

	env := &EnvB{}

	_ = os.Setenv("__PREFIX_NESTED_A_FOO", "baz")
	if err := SetWithEnvPrefix(env, "__PREFIX"); err != nil {
		t.Fatal(err)
	}

	if exp := 42; env.Flat != exp {
		t.Fatalf("expected: %v got: %v", exp, env.Flat)
	}
	if exp := "baz"; env.Nested.Foo != exp {
		t.Fatalf("expected: %v got: %v", exp, env.Nested.Foo)
	}
}
