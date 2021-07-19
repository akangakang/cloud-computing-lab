package datanode_alloc

type NoDataNodeError struct {
}

func (n *NoDataNodeError) Error() string {
	return "No DataNode!"
}
