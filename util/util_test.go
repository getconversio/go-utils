package util

import (
	"errors"
	"io/ioutil"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Produces 32-bit string hash in hexadecimal form.
func TestHash32(t *testing.T) {
	assert.Equal(t, "368ad1fa", Hash32("hej"))
	assert.Equal(t, "40f2ddc1", Hash32("manyothercharacters-,.+0æøå'~"))
}

func TestPanicOnError(t *testing.T) {
	assert.Panics(t, func() {
		PanicOnError("oh no", errors.New("it's terrible"))
	})
}

func TestGetenvInt(t *testing.T) {
	err := os.Setenv("notint", "abc")
	require.NoError(t, err)

	err = os.Setenv("empty", "")
	require.NoError(t, err)

	err = os.Setenv("anint", "123")
	require.NoError(t, err)

	// Fallback
	assert.Equal(t, 111, GetenvInt("notint", 111))
	assert.Equal(t, 222, GetenvInt("empty", 222))

	// Parse the int
	assert.Equal(t, 123, GetenvInt("anint", 111))
}

func TestGetenv(t *testing.T) {
	err := os.Setenv("empty", "")
	require.NoError(t, err)

	err = os.Setenv("foo", "bar")
	require.NoError(t, err)

	// Has variable
	assert.Equal(t, "bar", Getenv("foo", "baz"))

	// Use fallback for both missing and empty fields
	assert.Equal(t, "baz", Getenv("empty", "baz"))
	assert.Equal(t, "baz", Getenv("noexists", "baz"))
}

func TestGracefulShutdown(t *testing.T) {
	cleaned := false
	ch := GracefulShutdown(func() { cleaned = true })

	// It should not run immediately
	assert.False(t, cleaned)

	// A SIGINT should run the cleanup function
	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
	select {
	case <-ch:
		break
	case <-time.After(1 * time.Second):
		t.Fatal("waited too long for signal")
	}

	// There should be nothing more in the channel
	assert.Empty(t, ch)
	assert.True(t, cleaned)
}

func TestReadFileEnvsubst(t *testing.T) {
	err := os.Setenv("MYVAR", "hej")
	require.NoError(t, err)
	err = os.Setenv("MY_VAR", "hej2")
	require.NoError(t, err)
	err = os.Setenv("ANOTHER_VAR", "hej3")
	require.NoError(t, err)
	defer os.Unsetenv("MYVAR")
	defer os.Unsetenv("MY_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tmpfile, err := ioutil.TempFile("", "tempconfig")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write([]byte(`{"mykey":"$MYVAR", "mykey2":"$MY_VAR.$ANOTHER_VAR"}`))
	require.NoError(t, err)
	err = tmpfile.Close()
	require.NoError(t, err)

	s := ReadFileEnvsubst(tmpfile.Name())
	assert.Equal(t, `{"mykey":"hej", "mykey2":"hej2.hej3"}`, s)
}
