package pool_test

import (
	"math/rand"
	"os"
	"sort"
	"testing"

	"github.com/roy2220/fsm/internal/buddy"
	"github.com/roy2220/fsm/internal/pool"
	"github.com/stretchr/testify/assert"
)

type SpaceMapper struct {
	buffer []byte
}

type SpaceInfo struct {
	Ptr  int64
	Size int
}

func (sm *SpaceMapper) MapSpace(spaceSize int) error {
	buffer := make([]byte, spaceSize)
	copy(buffer, sm.buffer)
	sm.buffer = buffer
	return nil
}

func (sm *SpaceMapper) AccessSpace() []byte {
	return sm.buffer
}

func TestPoolAllocateSpace(t *testing.T) {
	p, _, sis := MakePool(t)

	sort.Slice(sis, func(i, j int) bool {
		return sis[i].Ptr < sis[j].Ptr
	})

	lastSpaceEnd := int64(0)

	for _, si := range sis {
		assert.GreaterOrEqual(t, si.Ptr, lastSpaceEnd)
		ss := p.GetSpaceSize(si.Ptr)
		assert.Equal(t, si.Size, ss)
		lastSpaceEnd = si.Ptr + int64(si.Size)
	}
}

func TestPoolFreeSpace(t *testing.T) {
	p, b, sis := MakePool(t)

	rand.Shuffle(len(sis), func(i, j int) {
		sis[i], sis[j] = sis[j], sis[i]
	})

	for _, si := range sis {
		p.FreeSpace(si.Ptr)
	}

	for _, si := range sis {
		assert.Panics(t, func() {
			p.FreeSpace(si.Ptr)
		})
	}

	b.ShrinkSpace()
	assert.Equal(t, 0, b.SpaceSize())
	_, ok := p.GetPooledBlocks()()

	if !assert.False(t, ok) {
		p.Fprint(os.Stdout)
	}
}

func MakePool(t *testing.T) (*pool.Pool, *buddy.Buddy, []*SpaceInfo) {
	spaceMapper := SpaceMapper{}
	buddy := new(buddy.Buddy).Init(&spaceMapper)
	pool1 := new(pool.Pool).Init(buddy)
	sis := make([]*SpaceInfo, 1000000)
	tss2 := 0

	for i := range sis {
		sis[i] = MakeSpaceInfo(t, pool1)
		tss2 += sis[i].Size

		if i%2 == 1 {
			j := rand.Intn(i + 1)
			pool1.FreeSpace(sis[j].Ptr)
			tss2 -= sis[j].Size
			sis[j] = MakeSpaceInfo(t, pool1)
			tss2 += sis[j].Size
		}
	}

	t.Logf("allocated space size: %d, total space size: %d", buddy.AllocatedSpaceSize(), tss2)
	t.Logf("total space size / allocated space size: %f", float64(tss2)/float64(buddy.AllocatedSpaceSize()))
	return pool1, buddy, sis
}

func MakeSpaceInfo(t *testing.T, pool1 *pool.Pool) *SpaceInfo {
	ss := rand.Intn(pool.BlockSize)
	f := rand.Float64()
	f *= f
	f *= f
	f *= f
	ss = int(float64(ss) * f)
	sptr, ss2 := pool1.AllocateSpace(ss)

	if !assert.GreaterOrEqual(t, ss2, ss) {
		t.FailNow()
	}

	return &SpaceInfo{sptr, ss2}
}
