package buffmgr

func GetPaddedBytes(s string, size uint64) []byte {
	// Fill padding with padByte.
	padByte := []byte(s)[0]
	paddedData := make([]byte, size)
	for i := uint64(0); i < size; i++ {
		paddedData[i] = padByte
	}
	return paddedData
}
