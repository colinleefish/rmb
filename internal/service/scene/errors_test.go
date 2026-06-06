package scene

import (
	"errors"
	"testing"
)

func TestIsTransientLLMError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{errors.New(`llm http 429: rate limit`), true},
		{errors.New(`您的账户已达到速率限制`), true},
		{errors.New(`context deadline exceeded`), true},
		{errors.New(`parse build scenes: invalid json`), false},
	}
	for _, tc := range cases {
		if got := isTransientLLMError(tc.err); got != tc.want {
			t.Fatalf("isTransientLLMError(%q) = %v, want %v", tc.err, got, tc.want)
		}
	}
}
