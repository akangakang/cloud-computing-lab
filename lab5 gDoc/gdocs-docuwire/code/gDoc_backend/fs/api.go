package fs

import (
	"context"
	"io/fs"
	"time"

	"github.com/astaxie/beego"
	"github.com/fourstring/sheetfs/fsclient"
	fs_rpc "github.com/fourstring/sheetfs/protocol"
)

type File struct {
	*fsclient.File
}

var (
	ctx         context.Context
	cancel      context.CancelFunc
	ErrNotExist = fs.ErrNotExist
)

func getCtx() context.Context {
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	return ctx
}

func Create(filename string) (file *File, err error) {
	client := getClient()
	fd, err := client.Create(getCtx(), filename)
	file = &File{
		File: fd,
	}
	return file, err
}

func Delete(filename string) error {
	client := getClient()
	return client.Delete(getCtx(), filename)
}

func Open(filename string) (file *File, err error) {
	client := getClient()
	fd, err := client.Open(getCtx(), filename)
	file = &File{
		File: fd,
	}
	return file, err
}

func (*File) Close() {

}

func (f *File) Read() (b []byte, n int64, err error) {
	b, n, err = f.File.Read(getCtx())
	switch _err := err.(type) {
	case *fsclient.CancelledError:
		beego.Error("[FS] Read operation timed out")
		cancel()
	case *fsclient.UnexpectedStatusError:
		if _err.Status() == fs_rpc.Status_Invalid {
			err = ErrNotExist
		} else {
			beego.Error("[FS] Read operation returned unexpected error")
		}
	}
	return b, n, err
}

// func (f *File) Write(buffer []byte, n int32) (int64, error) {
// 	return f.File.WriteAt(getCtx(), buffer, 1, 1, "")
// }

func (f *File) ReadAt(buffer []byte, row uint32, col uint32) (n int64, err error) {
	n, err = f.File.ReadAt(getCtx(), buffer, row, col)
	switch _err := err.(type) {
	case *fsclient.CancelledError:
		beego.Error("[FS] ReadAt operation timed out")
		cancel()
	case *fsclient.UnexpectedStatusError:
		if _err.Status() == fs_rpc.Status_Invalid {
			err = ErrNotExist
			return n, err
		} else {
			beego.Error("[FS] ReadAt operation returned unexpected error")
		}
	}
	for i := len(buffer) - 1; i > 0; i-- {
		if buffer[i] == ',' {
			buffer[i] = ' '
			break
		}
	}
	return n, err
}

func (f *File) ReadMeta(buffer []byte) (n int64, err error) {
	n, err = f.File.ReadAt(getCtx(), buffer[1:], fsclient.MetaCellRow, fsclient.MetaCellCol)
	switch _err := err.(type) {
	case *fsclient.CancelledError:
		beego.Error("[FS] ReadAt operation timed out")
		cancel()
	case *fsclient.UnexpectedStatusError:
		if _err.Status() == fs_rpc.Status_Invalid {
			err = ErrNotExist
		} else {
			beego.Error("[FS] ReadAt operation returned unexpected error")
		}
	}
	buffer[0] = '{'
	buffer[len(buffer)-1] = '}'
	return n, err
}

func (f *File) WriteAt(buffer []byte, row uint32,
	col uint32, padding string) (n int64, err error) {
	buffer = append(buffer, ',')
	beego.Debug("WriteAt:", row, col, string(buffer))
	n, err = f.File.WriteAt(getCtx(), buffer, row, col, padding)
	switch err.(type) {
	case *fsclient.CancelledError:
		beego.Error("[FS] WriteAt operation timed out")
		cancel()
	case *fsclient.UnexpectedStatusError:
		beego.Error("[FS] WriteAt operation returned unexpected error")
	}
	return n, err
}

func (f *File) WriteMeta(buffer []byte) (n int64, err error) {
	beego.Debug("WriteMeta:", string(buffer[1:len(buffer)-1]))
	n, err = f.File.WriteAt(getCtx(), buffer[1:len(buffer)-1],
		fsclient.MetaCellRow, fsclient.MetaCellCol, " ")
	switch err.(type) {
	case *fsclient.CancelledError:
		beego.Error("[FS] WriteMeta operation timed out")
		cancel()
	case *fsclient.UnexpectedStatusError:
		beego.Error("[FS] WriteMeta operation returned unexpected error")
	}
	return n, err
}

func List() ([]string, error) {
	return make([]string, 0), nil
}
