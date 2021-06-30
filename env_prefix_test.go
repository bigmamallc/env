package env

import (
	"os"
	"testing"
)

func TestEnvStringWithPrefix(t *testing.T) {
	os.Setenv("BAR_PROP", "FOO")

	config := struct {
		Prop string `env:"PROP"`
	}{}

	ErrorNil(t, SetWithEnvPrefix(&config, "BAR_"))
	Equals(t, "FOO", config.Prop)
}

