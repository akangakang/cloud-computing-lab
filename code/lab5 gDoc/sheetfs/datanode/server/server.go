package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/fourstring/sheetfs/common_journal"
	"github.com/fourstring/sheetfs/config"
	"github.com/fourstring/sheetfs/datanode/journal"
	"github.com/fourstring/sheetfs/datanode/utils"
	fsrpc "github.com/fourstring/sheetfs/protocol"
	"hash/crc32"
	"io/fs"
	"os"
	"path"
	"strconv"
)

type Server struct {
	fsrpc.UnimplementedDataNodeServer
	dataPath string
	writer   *common_journal.Writer
}

func NewServer(path string, writer *common_journal.Writer) *Server {
	fmt.Printf("start a new server with path %s\n", path)
	err := os.MkdirAll(path, 0777)
	if err != nil {
		fmt.Printf("server with path %s mkdir fail\n", path)
	}
	return &Server{
		dataPath: path,
		writer:   writer,
	}
}

func (s *Server) DeleteChunk(ctx context.Context, request *fsrpc.DeleteChunkRequest) (*fsrpc.DeleteChunkReply, error) {
	reply := new(fsrpc.DeleteChunkReply)
	var err error

	/* TODO: First write log to Kafka */
	entry := journal.ConstructDeleteEntry(request)
	for i := 0; i < config.ACK_MOST_TIMES; i++ {
		err = s.writer.CommitEntry(ctx, entry)
		if err == nil {
			break
		}
	}
	if err != nil { // write to kafka fail
		reply.Status = fsrpc.Status_Unavailable
		fmt.Println(err)
		return reply, nil
	}

	err = os.Remove(s.getFilename(request.Id))
	if err != nil {
		print("not delete")
	}
	reply.Status = fsrpc.Status_OK
	return reply, nil
}

func (s *Server) ReadChunk(ctx context.Context, request *fsrpc.ReadChunkRequest) (*fsrpc.ReadChunkReply, error) {
	reply := new(fsrpc.ReadChunkReply)

	file, err := os.Open(s.getFilename(request.Id))

	// this file does not exist
	if err != nil {
		file.Close()
		reply.Status = fsrpc.Status_NotFound
		return reply, nil
	}

	// check version
	curVersion := utils.GetVersion(file)
	if curVersion >= request.Version {
		// the version is correct
		data := make([]byte, request.Size)
		_, err = file.ReadAt(data, int64(request.Offset))
		file.Close()

		// can not read data at this pos
		if err != nil {
			reply.Status = fsrpc.Status_NotFound
			return reply, nil
		}
		// read the correct data
		reply.Data = data
		reply.Status = fsrpc.Status_OK
	} else {
		file.Close()
		reply.Status = fsrpc.Status_WrongVersion
		return reply, nil
	}

	return reply, nil
}

func (s *Server) WriteChunk(ctx context.Context, request *fsrpc.WriteChunkRequest) (*fsrpc.WriteChunkReply, error) {
	reply := new(fsrpc.WriteChunkReply)
	var err error

	/* get the padded data first */
	PaddedData := utils.GetPaddedData(request.Data, request.Size, request.TargetSize, request.Padding)

	/* TODO: First write log to Kafka */
	entry := journal.ConstructWriteEntry(request, PaddedData)
	for i := 0; i < config.ACK_MOST_TIMES; i++ {
		err = s.writer.CommitEntry(ctx, entry)
		if err == nil {
			break
		}
	}
	if err != nil { // write to kafka fail
		reply.Status = fsrpc.Status_Unavailable
		fmt.Println(err)
		return reply, nil
	}

	file, err := os.OpenFile(s.getFilename(request.Id), os.O_RDWR, 0755)

	// first time
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// MasterNode assigns version 1 to those chunks which don't exist before.
			if request.Version != 1 {
				reply.Status = fsrpc.Status_WrongVersion
				return reply, nil
			}
			// create the file
			file, err := os.Create(s.getFilename(request.Id))
			if err != nil {
				reply.Status = fsrpc.Status_Unavailable
				fmt.Println(err)
				return reply, nil
			}
			// write the data
			_, err = file.WriteAt(utils.GetPaddedFile(request.Data, request.Size,
				request.TargetSize, request.Padding, request.Offset), 0)
			if err != nil {
				reply.Status = fsrpc.Status_Unavailable
				return reply, nil
			}

			// update the version
			utils.SyncAndUpdateVersion(file, request.Version)
			reply.Status = fsrpc.Status_OK
			return reply, nil
		}
		reply.Status = fsrpc.Status_Unavailable
		return reply, nil
	}

	curVersion := utils.GetVersion(file)
	// print("current version: ", curVersion, ", request version: ", request.Version)
	if curVersion+1 == request.Version {
		// can update
		// write the data
		_, err := file.WriteAt(PaddedData, int64(request.Offset))
		if err != nil {
			file.Close()
			reply.Status = fsrpc.Status_NotFound
			return reply, nil
		}

		// update the version
		utils.SyncAndUpdateVersion(file, request.Version)

		reply.Status = fsrpc.Status_OK
		return reply, nil
	} else {
		// some backup write
		file.Close()
		reply.Status = fsrpc.Status_WrongVersion
		return reply, nil
	}
}

func (s *Server) getFilename(id uint64) string {
	return path.Join(s.dataPath, "chunk_"+strconv.FormatUint(id, 10))
}

func (s *Server) HandleWriteMsg(msg []byte) error {
	version := utils.BytesToUint64(msg[0:8])
	chunkid := utils.BytesToUint64(msg[8:16])
	offset := utils.BytesToUint64(msg[16:24])
	size := utils.BytesToUint64(msg[24:32])

	// try to open the file
	file, err := os.OpenFile(s.getFilename(chunkid), os.O_RDWR, 0755)

	// this file does not exist
	if err != nil {
		// create the file and write data
		for {
			file, err = os.Create(s.getFilename(chunkid))
			if err == nil {
				break
			}
		}
		for {
			_, err = file.WriteAt(utils.GetPaddedFile(msg[36:], size,
				size, " ", offset), 0)
			if err == nil {
				break
			}
		}

		// the version is newest
		utils.SyncAndUpdateVersion(file, version)
		return nil
	}

	// the file already exist
	// check the checksum first
	oldData := make([]byte, size)
	_, err = file.ReadAt(oldData, int64(offset))
	if err != nil {
		return err
	}
	dataCks := crc32.Checksum(oldData, config.Crc32q)

	// if they have different checksum or different version
	if utils.BytesToUint32(msg[32:36]) != dataCks ||
		version != utils.GetVersion(file) {
		// overwrite
		for {
			_, err = file.WriteAt(msg[36:], int64(offset))
			if err == nil {
				break
			}
		}
		// update the version
		utils.SyncAndUpdateVersion(file, version)
	}
	return nil
}

func (s *Server) HandleDeleteMsg(msg []byte) error {
	chunkid := utils.BytesToUint64(msg[0:8])
	err := os.Remove(s.getFilename(chunkid))
	if err != nil {
		fmt.Println("handle delete log: no such file")
	}
	fmt.Println("handle delete log: successfully remove file")
	return nil
}

func (s *Server) HandleMsg(msg []byte) error {
	// TODO: secondary handle message from kafka
	logType := utils.BytesToUint64(msg[0:8])

	switch logType {
	case config.WRITE_LOG_FLAG:
		return s.HandleWriteMsg(msg[8:])
	case config.DELETE_LOG_FLAG:
		return s.HandleDeleteMsg(msg[8:])
	default:
		return nil
	}
}
