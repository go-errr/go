package lang_test

import (
	"testing"

	"github.com/go-errr/go/lang"
	"github.com/stretchr/testify/require"
)

func Test_IsNil(t *testing.T) {
	t.Run("should produce expected results'", func(t *testing.T) {
		require.Equal(t, false, lang.IsNil(0))
		require.Equal(t, false, lang.IsNil(""))

		var notInitializedAny any
		require.Equal(t, true, lang.IsNil(notInitializedAny))

		var notInitializedPtr *any
		require.Equal(t, true, lang.IsNil(notInitializedPtr))

		var notInitializedInt int
		require.Equal(t, false, lang.IsNil(notInitializedInt))

		var notInitializedMap map[string]any
		require.Equal(t, true, notInitializedMap == nil)
		require.Equal(t, true, lang.IsNil(notInitializedMap))

		var notInitializedString string
		require.Equal(t, false, lang.IsNil(notInitializedString))
	})
}
