package smtp

import (
	"bytes"
	"time"

	"github.com/satori/go.uuid"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/elastic/beats/packetbeat/protos/applayer"
)

type transaction interface{}

type transPrompt struct {
	resp *message
}

type transCommand struct {
	requ, resp *message
}

type transMail struct {
	applayer.Transaction

	// Envelope sender
	reversePath common.NetString
	// Envelope recipients
	forwardPaths []common.NetString

	// DATA payload request in requ
	requ, resp *message
}

type transactions struct {
	config    *transactionConfig
	sessionID uuid.UUID

	requests  messageList
	responses messageList

	current       transaction
	onTransaction transactionHandler
}

type transactionConfig struct {
	transactionTimeout time.Duration
}

type transactionHandler func(transaction, uuid.UUID) error

// List of messages available for correlation
type messageList struct {
	head, tail *message
}

func (trans *transactions) init(c *transactionConfig, cb transactionHandler) {
	trans.config = c
	trans.onTransaction = cb
	trans.sessionID = uuid.NewV4()
}

func (trans *transactions) onMessage(
	tuple *common.IPPortTuple,
	dir uint8,
	msg *message,
) error {
	msg.Tuple = *tuple
	msg.Transport = applayer.TransportTCP
	msg.CmdlineTuple = procs.ProcWatcher.FindProcessesTuple(&msg.Tuple)

	if msg.IsRequest {
		if isDebug {
			debugf("Request: %s %s", msg.command, msg.param)
		}
		trans.requests.append(msg)
	} else {
		if isDebug {
			debugf("Response: %d %s", msg.statusCode, msg.statusPhrases)
		}
		trans.responses.append(msg)
	}

	return trans.correlate()
}

func (trans *transactions) correlate() error {
	requests := &trans.requests
	responses := &trans.responses

	// Some transactions consist of a single response
	if requests.empty() {
		for !responses.empty() {
			resp := responses.pop()
			if complete := trans.add(nil, resp); complete {
				err := trans.onTransaction(trans.current, trans.sessionID)
				if err != nil {
					return err
				}
			} else {
				logp.Warn(
					"Ignoring response from unknown transaction: %d %s",
					resp.statusCode,
					resp.statusPhrases)
			}
		}
		return nil
	}

	for !responses.empty() && !requests.empty() {
		resp := responses.pop()
		requ := requests.pop()

		if complete := trans.add(requ, resp); complete {
			err := trans.onTransaction(trans.current, trans.sessionID)
			if err != nil {
				return err
			}
			trans.current = nil
		}
	}

	return nil
}

// Add iteratively processes request/response pairs to create 3 types
// of transactions:
// - PROMPT:  response only, possible codes 220, 421, 554
// - COMMAND: request/response (except MAIL-related ones)
// - MAIL:    combines `MAIL`, `RCPT`, `DATA` and `EOD`
//            requests/responses
func (trans *transactions) add(requ, resp *message) bool {
	if requ == nil {
		// Check for prompt responses
		switch resp.statusCode {
		case 220, 421, 554:
			trans.current = &transPrompt{resp}
			return true
		default:
			// Stray response
			return false
		}
	}

	// Treat MAIL-related commands as one big transaction, the rest as
	// simple request/response transactions
	switch {

	case bytes.Equal(requ.command, constMAIL):
		t := trans.ensureMailTransaction(requ, resp)
		t.reversePath = getPath(requ.param)
		// Error response ends transaction
		if resp.statusCode != 250 {
			return true
		}
		return false

	case bytes.Equal(requ.command, constRCPT):
		t := trans.ensureMailTransaction(requ, resp)
		t.forwardPaths =
			append(t.forwardPaths, getPath(requ.param))
		// Error response doesn't end transaction
		return false

	case bytes.Equal(requ.command, constDATA):
		trans.ensureMailTransaction(requ, resp)
		return false

	case bytes.Equal(requ.command, constEOD):
		trans.ensureMailTransaction(requ, resp)
		if resp.statusCode == 250 {
			return true
		}
		return false

	default:
		trans.current = &transCommand{requ, resp}
		return true
	}
}

func (trans *transactions) ensureMailTransaction(requ, resp *message) *transMail {
	// In case the mail-related command was issued before `MAIL`
	if _, ok := trans.current.(*transMail); !ok {
		trans.current = &transMail{}
	}

	t := trans.current.(*transMail)

	t.requ, t.resp = requ, resp
	t.BytesIn += requ.Size
	t.BytesOut += resp.Size

	// Collect error messages, if any
	if resp.statusCode >= 400 {
		for _, sp := range resp.statusPhrases {
			t.Notes = append(t.Notes, string(sp))
		}
		t.Status = common.SERVER_ERROR_STATUS
	} else {
		t.Status = common.OK_STATUS
	}

	return t
}

func (ml *messageList) append(msg *message) {
	if ml.tail == nil {
		ml.head = msg
	} else {
		ml.tail.next = msg
	}
	msg.next = nil
	ml.tail = msg
}

func (ml *messageList) empty() bool {
	return ml.head == nil
}

func (ml *messageList) pop() *message {
	if ml.head == nil {
		return nil
	}

	msg := ml.head
	ml.head = ml.head.next
	if ml.head == nil {
		ml.tail = nil
	}
	return msg
}

func (ml *messageList) first() *message {
	return ml.head
}

func (ml *messageList) last() *message {
	return ml.tail
}
