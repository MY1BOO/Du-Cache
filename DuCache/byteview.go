package DuCache

//ByteView是存储真实的缓存值，也就是entry中的value
//选择 byte 类型是为了能够支持任意的数据类型的存储，例如字符串、图片等
type ByteView struct {
	b []byte
}

//entry的value要求必须实现Value接口的Len()方法
func (v ByteView) Len() int {
	return len(v.b)
}

//ByteView是只读的，返回一个拷贝防止对缓存值进行修改
func (v ByteView) ByteSlice() []byte {
	return cloneBytes(v.b)
}

//解码
func (v ByteView) String() string {
	return string(v.b)
}

func cloneBytes(b []byte) []byte {
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
