package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseOptions 是纯函数：不读配置、不连库，缺参分支可以在无 DB 环境下覆盖。
func TestParseOptions(t *testing.T) {
	opts, err := parseOptions([]string{
		"-phone", "13800000000",
		"-kg", "7",
		"-password", "S3cret-Admin",
	}, func(string) string { return "" })
	require.NoError(t, err)
	assert.Equal(t, "13800000000", opts.Phone)
	assert.Equal(t, uint64(7), opts.KindergartenID)
	assert.Equal(t, "S3cret-Admin", opts.Password)
}

func TestParseOptionsMissingPhone(t *testing.T) {
	_, err := parseOptions([]string{"-kg", "7", "-password", "x"}, func(string) string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-phone")
}

func TestParseOptionsMissingKindergarten(t *testing.T) {
	_, err := parseOptions([]string{"-phone", "13800000000", "-password", "x"}, func(string) string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-kg")
}

func TestParseOptionsMissingPassword(t *testing.T) {
	_, err := parseOptions([]string{"-phone", "13800000000", "-kg", "7"}, func(string) string { return "" })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "口令")
}

func TestParseOptionsFallsBackToEnvPassword(t *testing.T) {
	opts, err := parseOptions([]string{"-phone", "13800000000", "-kg", "7"},
		func(key string) string {
			if key == "SETADMIN_PASSWORD" {
				return "EnvPass"
			}
			return ""
		})
	require.NoError(t, err)
	assert.Equal(t, "EnvPass", opts.Password)
}

func TestParseOptionsFlagPasswordWinsOverEnv(t *testing.T) {
	opts, err := parseOptions([]string{"-phone", "13800000000", "-kg", "7", "-password", "FlagPass"},
		func(string) string { return "EnvPass" })
	require.NoError(t, err)
	assert.Equal(t, "FlagPass", opts.Password)
}
