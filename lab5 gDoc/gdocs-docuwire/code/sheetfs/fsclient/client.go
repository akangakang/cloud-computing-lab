package fsclient

import (
	"context"
	"errors"
	"fmt"
	fsrpc "github.com/fourstring/sheetfs/protocol"
	"github.com/go-zookeeper/zk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"io/fs"
	"reflect"
	"strings"
	"sync"
	"time"
)

type ClientConfig struct {
	ZookeeperServers    []string
	ZookeeperTimeout    time.Duration
	MasterZnode         string
	DataNodeZnodePrefix string
	MaxRetry            int
}

type masterNode struct {
	c       fsrpc.MasterNodeClient
	version uint64
}

type dataNode struct {
	c       fsrpc.DataNodeClient
	version uint64
}

type Client struct {
	cfg         *ClientConfig
	mu          sync.RWMutex
	zk          *zk.Conn
	master      *masterNode
	datanodeMap map[string]*dataNode
}

func NewClient(config *ClientConfig) (*Client, error) {
	conn, _, err := zk.Connect(config.ZookeeperServers, config.ZookeeperTimeout)
	if err != nil {
		return nil, err
	}

	c := &Client{
		cfg:         config,
		zk:          conn,
		master:      &masterNode{},
		datanodeMap: map[string]*dataNode{},
	}

	var rpcc fsrpc.MasterNodeClient

	for i := 0; i < config.MaxRetry; i++ {
		rpcc, _, err = c.reAskMasterNode()
		if err == nil {
			break
		}
	}

	if err != nil {
		return nil, err
	}

	c.master.c = rpcc

	return c, nil
}

func (c *Client) readMaster() (fsrpc.MasterNodeClient, uint64) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.master.c, c.master.version
}

func (c *Client) readDataNode(datanodeGroupName string) (fsrpc.DataNodeClient, uint64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	n, ok := c.datanodeMap[datanodeGroupName]
	if !ok {
		return nil, 0, false
	}
	return n.c, n.version, ok
}

func checkMethodParams(method reflect.Value, params ...interface{}) ([]reflect.Value, error) {
	if method.Kind() != reflect.Func {
		return nil, errors.New("no such method")
	}
	// exclude method receiver
	numIn := method.Type().NumIn() - 1
	if numIn != len(params) {
		return nil, fmt.Errorf("invalid params count, expected: %d, actual: %d", numIn, len(params))
	}
	paramVals := make([]reflect.Value, numIn)
	for i := 0; i < len(paramVals); i++ {
		paramVals[i] = reflect.ValueOf(params[i])
	}
	return paramVals, nil
}

func (c *Client) ensureMasterRPCWithRetry(name string, params ...interface{}) (interface{}, error) {
	masterClient, lastMasterVersion := c.readMaster()
	method := reflect.ValueOf(masterClient).MethodByName(name)
	// construct parameter list
	paramVals, e := checkMethodParams(method, params...)
	if e != nil {
		return nil, e
	}
	ret := method.Call(paramVals)
	rep, err := ret[0], ret[1]
	if !err.IsNil() {
		e = err.Interface().(error)
		rpcerr, _ := status.FromError(e)
		// check if the error indicates a node failure
		if rpcerr.Code() != codes.DeadlineExceeded {
			// not, return directly
			return nil, e
		}
		// retry for cfg.MaxRetry times
		for i := 0; i < c.cfg.MaxRetry; i++ {
			// may update client state, acquire writer lock
			c.mu.Lock()
			// check client and version of its node now,
			// because client may be used concurrently, it's possible that some other goroutine
			// have handled the broken connection. If so, we should try to use the repaired
			// rpc client directly to avoid redundant re-asks.
			curClient, curVersion := c.master.c, c.master.version
			if curVersion == lastMasterVersion {
				// there is no other goroutines who have already handled the broken rpc client
				curClient, _, e = c.reAskMasterNode()
				if e != nil {
					c.mu.Unlock()
					continue
				}
				// update client state, new version should be lastMasterVersion + 1
				curVersion = lastMasterVersion + 1
				c.master = &masterNode{
					c:       curClient,
					version: curVersion,
				}
			}
			// re-call
			ret = reflect.ValueOf(curClient).MethodByName(name).Call(paramVals)
			rep, err = ret[0], ret[1]
			if err.IsNil() {
				c.mu.Unlock()
				break
			}
			// update lastMasterVersion to version used by the re-call above, the curVersion.
			lastMasterVersion = curVersion
			c.mu.Unlock()
		}
	}
	if !err.IsNil() {
		return nil, err.Interface().(error)
	}
	return rep.Interface(), nil
}

func (c *Client) ensureDataNodeRPCWithRetry(datanodeGroup, name string, params ...interface{}) (interface{}, error) {
	datanodeClient, lastDatanodeVersion, _ := c.readDataNode(datanodeGroup)
	method := reflect.ValueOf(datanodeClient).MethodByName(name)
	paramVals, e := checkMethodParams(method, params...)
	if e != nil {
		return nil, e
	}
	ret := method.Call(paramVals)
	rep, err := ret[0], ret[1]
	if !err.IsNil() {
		e = err.Interface().(error)
		rpcerr, _ := status.FromError(e)
		if rpcerr.Code() != codes.DeadlineExceeded {
			return nil, e
		}
		for i := 0; i < c.cfg.MaxRetry; i++ {
			c.mu.Lock()
			d := c.datanodeMap[datanodeGroup]
			curClient, curVersion := d.c, d.version
			if curVersion == lastDatanodeVersion {
				curClient, _, e = c.reAskDataNode(datanodeGroup)
				if e != nil {
					c.mu.Unlock()
					continue
				}
				curVersion = lastDatanodeVersion + 1
				c.datanodeMap[datanodeGroup] = &dataNode{
					c:       curClient,
					version: curVersion,
				}
			}
			ret = reflect.ValueOf(curClient).MethodByName(name).Call(paramVals)
			rep, err = ret[0], ret[1]
			if err.IsNil() {
				c.mu.Unlock()
				break
			}
			lastDatanodeVersion = curVersion
			c.mu.Unlock()
		}
	}
	if !err.IsNil() {
		return nil, err.Interface().(error)
	}
	return rep.Interface(), nil
}

/*
Create
@para
	name(string):  the name of the file
@return
	f(*File): fd
	error(error): nil is no error
				fs.ErrExist: already exist
				fs.ErrInvalid: wrong para
*/
func (c *Client) Create(ctx context.Context, name string) (f *File, err error) {
	// check filename
	if name == "" || strings.Contains(name, "/") ||
		strings.Contains(name, "\\") {
		return nil, fs.ErrInvalid
	}
	// create the file
	req := fsrpc.CreateSheetRequest{Filename: name}
	// reply, err := c.master.CreateSheet(ctx, &req)
	_reply, err := c.ensureMasterRPCWithRetry("CreateSheet", ctx, &req)

	if err != nil {
		return nil, err
	}

	reply := _reply.(*fsrpc.CreateSheetReply)

	switch reply.Status {
	case fsrpc.Status_OK:
		return newFile(reply.Fd, name, c), nil
	case fsrpc.Status_Exist:
		return nil, fs.ErrExist
	default:
		return nil, NewUnexpectedStatusError(reply.Status)
	}
}

/*
Delete
@para
	name(string) : the name of the file
@return
	error(error): nil is no error
				fs.ErrExist: already exist
				fs.ErrInvalid: wrong para
*/
func (c *Client) Delete(ctx context.Context, name string) (err error) {
	// DeleteSheet
	req := fsrpc.DeleteSheetRequest{Filename: name}
	// reply, err := c.master.DeleteSheet(ctx, &req)
	_reply, err := c.ensureMasterRPCWithRetry("DeleteSheet", ctx, &req)

	if err != nil {
		return err
	}

	reply := _reply.(*fsrpc.DeleteSheetReply)

	switch reply.Status {
	case fsrpc.Status_OK:
		return nil
	case fsrpc.Status_NotFound:
		return fs.ErrNotExist
	default:
		return NewUnexpectedStatusError(reply.Status)
	}
}

/*
Open
@para
	name(string):  the name of the file
@return
	fd(uint64): the fd of the open file
	status(Status)
	error(error)
*/
func (c *Client) Open(ctx context.Context, name string) (f *File, err error) {
	// check filename
	if name == "" || strings.Contains(name, "/") ||
		strings.Contains(name, "\\") {
		return nil, fs.ErrInvalid
	}
	// open the required file
	req := fsrpc.OpenSheetRequest{Filename: name}
	// reply, err := c.master.OpenSheet(ctx, &req)
	_reply, err := c.ensureMasterRPCWithRetry("OpenSheet", ctx, &req)

	if err != nil {
		return nil, err
	}

	reply := _reply.(*fsrpc.OpenSheetReply)

	switch reply.Status {
	case fsrpc.Status_OK: // open correctly
		return newFile(reply.Fd, name, c), err
	case fsrpc.Status_NotFound: // not found
		return nil, fs.ErrNotExist
	default: // should never reach here
		return nil, NewUnexpectedStatusError(reply.Status)
	}
}

func (c *Client) checkNewDataNode(chunks []*fsrpc.Chunk) error {
	for _, chunk := range chunks {
		// register new client node
		groupName := chunk.Datanode
		if _, ok := c.datanodeMap[groupName]; !ok {
			c.mu.Lock()
			addr, _, err := c.zk.Get(c.cfg.DataNodeZnodePrefix + groupName)
			if err != nil {
				c.mu.Unlock()
				return err
			}
			conn, err := grpc.Dial(string(addr), grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				c.mu.Unlock()
				return err
			}
			client := fsrpc.NewDataNodeClient(conn)
			c.datanodeMap[groupName] = &dataNode{
				c: client,
			}
			c.mu.Unlock()
		}
	}
	return nil
}

func (c *Client) concurrentReadChunk(ctx context.Context, chunk *fsrpc.Chunk, in *fsrpc.ReadChunkRequest) (*fsrpc.ReadChunkReply, error) {
	// reply, err := dc.ReadChunk(ctx, in, opts...)
	_reply, err := c.ensureDataNodeRPCWithRetry(chunk.Datanode, "ReadChunk", ctx, in)

	// current datanode address can not serve
	if err != nil {
		return nil, err
	}

	reply := _reply.(*fsrpc.ReadChunkReply)

	return reply, nil
}

func (c *Client) concurrentWriteChunk(ctx context.Context, chunk *fsrpc.Chunk, in *fsrpc.WriteChunkRequest) (*fsrpc.WriteChunkReply, error) {
	_reply, err := c.ensureDataNodeRPCWithRetry(chunk.Datanode, "WriteChunk", ctx, in)

	// current datanode address can not serve
	if err != nil {
		return nil, err
	}

	reply := _reply.(*fsrpc.WriteChunkReply)
	return reply, nil
}

func (c *Client) reAskMasterNode() (fsrpc.MasterNodeClient, string, error) {
	masterAddr, _, err := c.zk.Get(c.cfg.MasterZnode)
	if err != nil { // retry
		return nil, "", err
	}
	conn, err := grpc.Dial(string(masterAddr), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, "", err
	}

	return fsrpc.NewMasterNodeClient(conn), string(masterAddr), nil
}

func (c *Client) reAskDataNode(datanode string) (fsrpc.DataNodeClient, string, error) {
	datanodeAddr, _, err := c.zk.Get(c.cfg.DataNodeZnodePrefix + datanode)
	if err != nil { // retry
		return nil, "", err
	}
	// connect
	conn, err := grpc.Dial(string(datanodeAddr), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil { // retry
		return nil, "", err
	}

	return fsrpc.NewDataNodeClient(conn), string(datanodeAddr), nil
}
