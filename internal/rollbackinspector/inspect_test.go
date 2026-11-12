package rollbackinspector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInspectRetainedRollbackFiles(t *testing.T) {
	const attempt = "4f322276-81e4-4d70-9f94-9951ded10e75"
	root := t.TempDir()
	match := filepath.Join(root, "123-"+attempt)
	require.NoError(t, os.Mkdir(match, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(match, "unrelated.txt"), []byte("ignored"), 0600))
	other := filepath.Join(root, "124-other-attempt")
	require.NoError(t, os.Mkdir(other, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(other, "rollback_1234_ab12cd34.py"), nil, 0600))
	found, err := Inspect(root, attempt)
	require.NoError(t, err)
	require.False(t, found)

	require.NoError(t, os.Symlink(filepath.Join(other, "rollback_1234_ab12cd34.py"), filepath.Join(match, "rollback_1234_ab12cd34.py")))
	found, err = Inspect(root, attempt)
	require.NoError(t, err)
	require.False(t, found)

	// Replace the symlink so the positive case is a regular file.
	require.NoError(t, os.Remove(filepath.Join(match, "rollback_1234_ab12cd34.py")))
	require.NoError(t, os.WriteFile(filepath.Join(match, "rollback_1234_ab12cd34.py"), nil, 0600))
	found, err = Inspect(root, attempt)
	require.NoError(t, err)
	require.True(t, found)

	message := filepath.Join(t.TempDir(), "termination-log")
	require.NoError(t, Run(root, attempt, message))
	content, err := os.ReadFile(message)
	require.NoError(t, err)
	require.Equal(t, "available", string(content))
}

func TestInspectRequiresReadableDirectory(t *testing.T) {
	_, err := Inspect(filepath.Join(t.TempDir(), "missing"), "attempt")
	require.Error(t, err)
	_, err = Inspect(t.TempDir(), "../attempt")
	require.Error(t, err)
}
