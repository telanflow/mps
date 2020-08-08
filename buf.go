package mps

type bufferPool struct {
	get func() []byte
	put func([]byte)
}

func (bp bufferPool) Get() []byte  { return bp.get() }
func (bp bufferPool) Put(v []byte) { bp.put(v) }
