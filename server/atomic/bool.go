package atomic

import "sync/atomic"

type Bool struct {
	val int32
}

func (b *Bool) CompareAndSwap(value bool) bool {
	old := getValueFromBool(!value)
	new := getValueFromBool(value)

	return atomic.CompareAndSwapInt32(&b.val, old, new)
}

func getValueFromBool(b bool) int32 {
	var i int32

	if b {
		i = 1
	}

	return i
}

func (b *Bool) Set(value bool) {
	i := getValueFromBool(value)

	atomic.StoreInt32(&b.val, i)
}

func (b *Bool) Get() bool {
	return atomic.LoadInt32(&b.val) != 0
}
