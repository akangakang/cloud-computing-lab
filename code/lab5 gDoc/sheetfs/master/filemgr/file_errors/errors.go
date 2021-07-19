package file_errors

import "fmt"

type FileNotFoundError struct {
	filename string
}

func NewFileNotFoundError(filename string) *FileNotFoundError {
	return &FileNotFoundError{filename: filename}
}

func (f *FileNotFoundError) Error() string {
	return fmt.Sprintf("File %s not found!", f.filename)
}

type FileExistsError struct {
	filename string
}

func NewFileExistsError(filename string) *FileExistsError {
	return &FileExistsError{filename: filename}
}

func (f *FileExistsError) Error() string {
	return fmt.Sprintf("File %s exists!", f.filename)
}

type FdNotFoundError struct {
	fd uint64
}

func NewFdNotFoundError(fd uint64) *FdNotFoundError {
	return &FdNotFoundError{fd: fd}
}

func (f *FdNotFoundError) Error() string {
	return fmt.Sprintf("fd %d not found!", f.fd)
}

type CellNotFoundError struct {
	row, col uint32
}

func (c *CellNotFoundError) Error() string {
	return fmt.Sprintf("Cell %d,%d not found!", c.row, c.col)
}

func NewCellNotFoundError(row uint32, col uint32) *CellNotFoundError {
	return &CellNotFoundError{row: row, col: col}
}
