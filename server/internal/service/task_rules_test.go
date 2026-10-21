package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"kindergarten-server/internal/domain"
)

func TestValidTaskFrequency(t *testing.T) {
	assert.True(t, validTaskFrequency(domain.TaskFrequencyDaily))
	assert.True(t, validTaskFrequency(domain.TaskFrequencyWeekly))
	assert.True(t, validTaskFrequency(domain.TaskFrequencyOnce))
	assert.False(t, validTaskFrequency(domain.TaskFrequency("monthly")))
	assert.False(t, validTaskFrequency(domain.TaskFrequency("")))
}

func TestSameTaskDay(t *testing.T) {
	loc := time.FixedZone("CST", 8*3600)
	a := time.Date(2026, 10, 20, 9, 0, 0, 0, loc)
	b := time.Date(2026, 10, 20, 23, 59, 0, 0, loc)
	c := time.Date(2026, 10, 21, 0, 0, 0, 0, loc)
	assert.True(t, sameTaskDay(a, b))
	assert.False(t, sameTaskDay(a, c))
}
