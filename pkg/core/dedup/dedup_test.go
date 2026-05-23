package dedup

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRequestDeduplicator(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(5 * time.Second)
	assert.NotNil(t, d)
	assert.Equal(t, 0, d.Size())

	t.Run("zero ttl gets default", func(t *testing.T) {
		d2 := NewRequestDeduplicator(0)
		assert.NotNil(t, d2)
	})
}

func TestIsDuplicateFirstCall(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(time.Minute)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
	assert.Equal(t, 1, d.Size())
}

func TestIsDuplicateSecondCall(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(time.Minute)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
	assert.True(t, d.IsDuplicate("transfer", "from=0x1"))
	assert.Equal(t, 1, d.Size())
}

func TestIsDuplicateDifferentMethods(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(time.Minute)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
	assert.False(t, d.IsDuplicate("balanceOf", "addr=0x1"))
	assert.Equal(t, 2, d.Size())
}

func TestIsDuplicateDifferentParams(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(time.Minute)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
	assert.False(t, d.IsDuplicate("transfer", "from=0x2"))
	assert.Equal(t, 2, d.Size())
}

func TestIsDuplicateExpired(t *testing.T) {
	d := NewRequestDeduplicator(10 * time.Millisecond)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))

	time.Sleep(20 * time.Millisecond)

	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
}

func TestSize(t *testing.T) {
	d := NewRequestDeduplicator(time.Hour)
	d.IsDuplicate("methodA", "paramsA")
	d.IsDuplicate("methodB", "paramsB")
	assert.Equal(t, 2, d.Size())
}

func TestSizeEvictsExpired(t *testing.T) {
	d := NewRequestDeduplicator(10 * time.Millisecond)
	d.IsDuplicate("methodA", "paramsA")

	time.Sleep(20 * time.Millisecond)

	assert.Equal(t, 0, d.Size())
}

func TestClear(t *testing.T) {
	d := NewRequestDeduplicator(time.Hour)
	d.IsDuplicate("methodA", "paramsA")
	d.IsDuplicate("methodB", "paramsB")
	assert.Equal(t, 2, d.Size())

	d.Clear()
	assert.Equal(t, 0, d.Size())

	assert.False(t, d.IsDuplicate("methodA", "paramsA"))
}

func TestHash(t *testing.T) {
	d := NewRequestDeduplicator(time.Minute)
	h1 := d.hash("methodA", "paramsA")
	h2 := d.hash("methodA", "paramsA")
	assert.Equal(t, h1, h2)

	h3 := d.hash("methodA", "paramsB")
	assert.NotEqual(t, h1, h3)

	h4 := d.hash("methodB", "paramsA")
	assert.NotEqual(t, h1, h4)
}

func TestIsDuplicateAfterClear(t *testing.T) {
	d := NewRequestDeduplicator(time.Hour)
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
	d.Clear()
	assert.False(t, d.IsDuplicate("transfer", "from=0x1"))
}

func TestIsDuplicateEmptyStrings(t *testing.T) {
	t.Parallel()
	d := NewRequestDeduplicator(time.Minute)
	assert.False(t, d.IsDuplicate("", ""))
	assert.True(t, d.IsDuplicate("", ""))
}
