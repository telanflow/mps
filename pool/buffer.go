package pool

import "sync"

var DefaultBuffer = NewBuffer(2048)

type Buffer struct {
	pl   *sync.Pool
	size int
}

func NewBuffer(size int) *Buffer {
	bufPool := &Buffer{
		pl:   nil,
		size: size,
	}
	bufPool.pl = &sync.Pool{
		New: bufPool.newPl,
	}
	return bufPool
}

func (b *Buffer) Get() []byte {
	return b.pl.Get().([]byte)
}

func (b *Buffer) Put(buf []byte) {
	b.pl.Put(buf)
}

func (b *Buffer) newPl() interface{} {
	return make([]byte, b.size)
}
