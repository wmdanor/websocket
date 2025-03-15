package internal

func Mask(bytes []byte, key [4]byte) {
	MaskOffset(bytes, key, 0)
}

func MaskOffset(bytes []byte, key [4]byte, offset int) {
	for i, b := range bytes {
		pos := i + offset
		masked := b ^ key[pos%4]
		bytes[i] = masked
	}
}
