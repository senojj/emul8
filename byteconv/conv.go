package byteconv

const hextableUpper = "0123456789ABCDEF"

func Btoh(src []byte, n int) string {
	dst := make([]byte, len(src)*2)
	j := 0
	for _, v := range src {
		dst[j] = hextableUpper[v>>4]
		dst[j+1] = hextableUpper[v&0x0f]
		j += 2
	}
	return string(dst[len(dst)-n:])
}

func U16tob(i uint16) []byte {
	var b [2]byte
	b[0] = byte(i >> 8)
	b[1] = byte(i)
	return b[:]
}
