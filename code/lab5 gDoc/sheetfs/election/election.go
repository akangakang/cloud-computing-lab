package election

import (
	"errors"
	"fmt"
	"github.com/go-zookeeper/zk"
	"sort"
	"time"
)

/*
Elector
Providing Object-Oriented API to manage the election process. Every node who wants to be the
primary should initialize an Elector and participate in the election through it.
*/
type Elector struct {
	conn          *zk.Conn
	electionZnode string
	proposePath   string
	electionAck   string
	proposal      string
}

/*
NewElector
Create an Elector instance and ensures that both of {electionZnode} and {electionAck} has been
created. Caller has no need to create them manually.

Elector helps the caller to engage the election by creating Ephemeral|Sequential Znode named
as '{electionZnode}/{electionPrefix}', and when a primary won the election, it should write
its identify info(e.g address) to Znode '{electionAck}'.

@param
	servers: a list of Zookeeper servers
	timeout: Zookeeper session timeout. When a session timeouts, the node established it
	is regarded as crashed, and lose it primary role(if it used to be) or its position
	in primary waiting queue.
	electionZnode, electionPrefix, electionAck: see Elector

@return
	*Elector
	error: not nil if failed to create those znodes.
*/
func NewElector(servers []string, timeout time.Duration, electionZnode string, electionPrefix string, electionAck string) (*Elector, error) {
	conn, _, err := zk.Connect(servers, timeout)
	if err != nil {
		return nil, err
	}
	_, err = conn.Create(electionZnode, []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil && !errors.Is(err, zk.ErrNodeExists) {
		return nil, err
	}
	_, err = conn.Create(electionAck, []byte{}, 0, zk.WorldACL(zk.PermAll))
	if err != nil && !errors.Is(err, zk.ErrNodeExists) {
		return nil, err
	}
	return &Elector{conn: conn, electionZnode: electionZnode, proposePath: fmt.Sprintf("%s/%s", electionZnode, electionPrefix), electionAck: electionAck}, nil
}

/*
CreateProposal
Create a Sequential|Ephemeral Znode named as '{electionZnode}/{electionPrefix}'.
Because Sequential Flag is applied, the name of finally created Znode will be
'{electionZnode}/{electionPrefix}-XXXXXXXXXX'. As mentioned in Zookeeper's
programming guide, the suffix will be a sequential, 0-padded 10-digits decimal number.
And the part of child's name in the new Znode, i.e. '{electionPrefix}-XXXXXXXXXX'
is called "Proposal" of caller SheetFS node(MasterNode or DataNode). The proposal
will be recorded by the Elector automatically, and is also returned for logging or other
usages.

@return
	string: proposal of this node
	error: not nil if fail to create proposal Znode.
*/
func (e *Elector) CreateProposal() (string, error) {
	proposal, err := e.conn.Create(e.proposePath, []byte{}, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		return "", err
	}
	// strip proceeding "{electionZnode}/"
	proposal = proposal[len(e.electionZnode)+1:]
	e.proposal = proposal
	return proposal, nil
}

/*
TryBeLeader
Check all children Znode of electionZnode(or all proposals) with smaller suffix than
proposal of called itself. Due to sequential guarantee of Zookeeper, a node always gain a
consistent view of its proceeding proposals. If the caller owns smallest proposal, than it is
granted as primary, or it should turn into a secondary one, watching for its closest predecessor.
When the latter crashed, its proposal will be deleted by Zookeeper according to Ephemeral flag,
and this node is awaken. Caller should call this method again after that, to TryBeLeader.

@return
	bool: whether this node is primary or not
	string: full path of znode to be watched, for logging or other purposes.
	<-chan zk.Event: a channel for notification of crashed predecessor. If the predecessor
	died, a Event can be received from this channel, and caller should call this method
	again.
	error: not nil if failed to read all proposals or watch predecessor.
*/
func (e *Elector) TryBeLeader() (bool, string, <-chan zk.Event, error) {
	// children will be []string{"{electionPrefix}-xxxxx",...}, or "Proposal"s.
	children, _, err := e.conn.Children(e.electionZnode)
	if err != nil {
		return false, "", nil, err
	}
	sort.Strings(children)
	lastLt := ""
	for _, child := range children {
		if child < e.proposal {
			lastLt = child
		} else {
			break
		}
	}
	if lastLt == "" {
		return true, "", nil, nil
	} else {
		watch := fmt.Sprintf("%s/%s", e.electionZnode, lastLt)
		_, _, c, err := e.conn.GetW(watch)
		if err != nil {
			return false, "", nil, err
		}
		return false, watch, c, nil
	}
}

/*
AckLeader
Write info into electionAck Znode if a node become the primary one successfully. And
the others can understand that who is the primary through reading electionAck.

The new primary should only call this method when it has been prepared to serving
requests, because outer world would be able to contact with it after AckLeader.

Secondary should never call this method.

@param
	conn: *zk.Conn created by Connect
	electionAck: Znode path to store acknowledgement info
	info: data to be written into electionAck

@return
	error: not nil if failed to write info.
*/
func (e *Elector) AckLeader(info string) error {
	_, err := e.conn.Set(e.electionAck, []byte(info), -1)
	return err
}
