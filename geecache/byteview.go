package geecache

type ByteView struct {
	bytes []byte
}

func (bv ByteView) Len() int {
	return len(bv.bytes)
}

func (bv ByteView) String() string {
	return string(bv.bytes)
}

func (bv ByteView) ByteSlice() []byte {
	return cloneBytes(bv.bytes)
}

func cloneBytes(bytes []byte) []byte {
	b := make([]byte, len(bytes))
	copy(b, bytes)
	return b
}
