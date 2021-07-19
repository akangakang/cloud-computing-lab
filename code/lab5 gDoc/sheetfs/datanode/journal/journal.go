package journal

import (
	"github.com/fourstring/sheetfs/config"
	"github.com/fourstring/sheetfs/datanode/utils"
	fsrpc "github.com/fourstring/sheetfs/protocol"
	"hash/crc32"
)

func ConstructWriteEntry(request *fsrpc.WriteChunkRequest, paddedData []byte) []byte {
	var entry []byte
	entry = append(entry, utils.Uint64ToBytes(config.WRITE_LOG_FLAG)...)
	entry = append(entry, utils.Uint64ToBytes(request.Version)...)
	entry = append(entry, utils.Uint64ToBytes(request.Id)...)
	entry = append(entry, utils.Uint64ToBytes(request.Offset)...)
	entry = append(entry, utils.Uint64ToBytes(uint64(len(paddedData)))...)
	checksum := crc32.Checksum(paddedData, config.Crc32q)
	entry = append(entry, utils.Uint32ToBytes(checksum)...)
	entry = append(entry, paddedData...)
	return entry
}

func ConstructDeleteEntry(request *fsrpc.DeleteChunkRequest) []byte {
	var entry []byte
	entry = append(entry, utils.Uint64ToBytes(config.DELETE_LOG_FLAG)...)
	entry = append(entry, utils.Uint64ToBytes(request.Id)...)
	return entry
}
