package blockcutter

import (
	"time"

	cb "github.com/hyperledger/fabric-protos-go/common"
	"github.com/hyperledger/fabric-protos-go/orderer"
	"github.com/osdi23p228/fabric/common/util"
)

// strawman codes vvvvvvvvvvvvvvvvvvvvvvv
type Scheduler struct {
	metrics               *Metrics
	channelID             string
	envelopes             map[int]*cb.Envelope
	startSequence         int
	nextSequence          int
	batchSize             uint32
	pendingBatchStartTime time.Time
}

func NewScheduler(metrics *Metrics, channelID string) *Scheduler {
	s := &Scheduler{
		metrics:       metrics,
		channelID:     channelID,
		envelopes:     make(map[int]*cb.Envelope),
		startSequence: 0,
		nextSequence:  0,
		batchSize:     0,
	}
	return s
}

func (s *Scheduler) Schedule(msg *cb.Envelope, txID string, batchSize *orderer.BatchSize) ([][]*cb.Envelope, bool) {
	messageBatches := make([][]*cb.Envelope, 0)
	txSequence, _ := util.ParseCustomTXID(txID)

	s.envelopes[txSequence] = msg

	if s.nextSequence == txSequence {
		if s.nextSequence == s.startSequence {
			s.pendingBatchStartTime = time.Now()
		}

		for {
			_, exist := s.envelopes[s.nextSequence]
			if !exist {
				break
			}

			s.nextSequence += 1

			messageSize := messageSizeBytes(msg)
			newBatchSize := s.batchSize + messageSize
			if newBatchSize < batchSize.PreferredMaxBytes {
				s.batchSize = newBatchSize

			} else if newBatchSize > batchSize.PreferredMaxBytes {
				if s.nextSequence-s.startSequence == 1 {
					// If there is only one sufficiently large enough envelope,
					// cut it.
					batch := s.Cut(s.nextSequence)
					messageBatches = append(messageBatches, batch)
					s.batchSize = 0
				} else {
					// If there are multiple envelopes, cut all of them before the
					// current envelope.
					batch1 := s.Cut(s.nextSequence - 1)
					messageBatches = append(messageBatches, batch1)

					if messageSize >= batchSize.PreferredMaxBytes {
						// If this envelope is bigger than batchSize.PreferredMaxBytes
						// then cut it as a block
						s.pendingBatchStartTime = time.Now()

						batch2 := s.Cut(s.nextSequence)
						messageBatches = append(messageBatches, batch2)

						s.batchSize = 0
					} else {
						s.batchSize = messageSize
					}
				}
			} else {
				batch := s.Cut(s.nextSequence)
				messageBatches = append(messageBatches, batch)
				s.batchSize = 0
			}

			if uint32(s.nextSequence-s.startSequence) >= batchSize.MaxMessageCount {
				batch := s.Cut(s.nextSequence)
				messageBatches = append(messageBatches, batch)
				s.batchSize = 0
			}
		}
	}

	pending := len(s.envelopes) > 0

	return messageBatches, pending
}

func (s *Scheduler) Cut(end int) []*cb.Envelope {
	start := s.startSequence

	if start >= end {
		logger.Panicf("Scheduler.Cut() does not allow start >= end")
	}
	s.metrics.BlockFillDuration.With("channel", s.channelID).Observe(time.Since(s.pendingBatchStartTime).Seconds())

	batch := make([]*cb.Envelope, 0, end-start)
	for i := start; i < end; i++ {
		batch = append(batch, s.envelopes[i])
		delete(s.envelopes, i)
	}

	s.startSequence = end

	return batch
}

// strawman codes ^^^^^^^^^^^^^^^^^^^^^^^
