package err_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-errr/go/err"
	"github.com/stretchr/testify/assert"
)

func TestRecover_Swallow(t *testing.T) {
	var handled any

	func() {
		defer err.Recover(func(e any) {
			handled = e
		})
		panic("boom")
	}()

	assert.Equal(t, "boom", handled)
}

func TestRecover_Panic(t *testing.T) {
	called := false

	func() {
		defer err.Recover(func(e any) {
			called = true
			panic("handler boom")
		})
		panic("boom")
	}()

	assert.True(t, called)
}

func TestCatch_Swallow(t *testing.T) {
	var handled any

	func() {
		defer err.Catch(func(e any) {
			handled = e
		})
		panic("boom")
	}()

	assert.Equal(t, "boom", handled)
}

func TestCatch_Panic(t *testing.T) {
	defer func() {
		r := recover()
		assert.NotNil(t, r, "expected panic")
		assert.Equal(t, "handler boom", r)
	}()

	func() {
		defer err.Catch(func(e any) {
			panic("handler boom")
		})
		panic("boom")
	}()
}

func TestInterrupted(t *testing.T) {
	tests := []struct {
		name     string
		err      any
		expected bool
	}{
		{
			name:     "nil",
			err:      nil,
			expected: false,
		},
		{
			name:     "non error",
			err:      "boom",
			expected: false,
		},
		{
			name:     "interrupted exception",
			err:      err.NewInterruptedException("interrupted"),
			expected: true,
		},
		{
			name:     "wrapped interrupted exception",
			err:      err.NewRuntimeExceptionFrom("wrapped", err.NewInterruptedException("interrupted")),
			expected: true,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: true,
		},
		{
			name:     "deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: false,
		},
		{
			name:     "ordinary error",
			err:      errors.New("boom"),
			expected: false,
		},
		{
			name:     "wrapped ordinary error",
			err:      err.NewRuntimeExceptionFrom("wrapped", errors.New("boom")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, err.Interrupted(tt.err))
		})
	}
}
