package service

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupExportFilesInRemovesExpiredOnly(t *testing.T) {
	dir := t.TempDir()
	oldPath := filepath.Join(dir, "old.xlsx")
	newPath := filepath.Join(dir, "new.xlsx")
	require.NoError(t, os.WriteFile(oldPath, []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(newPath, []byte("new"), 0o644))

	oldTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(oldPath, oldTime, oldTime))

	require.NoError(t, cleanupExportFilesIn(dir, 24*time.Hour))

	_, err := os.Stat(oldPath)
	assert.True(t, os.IsNotExist(err), "过期文件应被删除")
	_, err = os.Stat(newPath)
	assert.NoError(t, err, "未过期文件应保留")
}
