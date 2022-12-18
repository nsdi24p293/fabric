/*
Copyright IBM Corp. 2016 All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorser

import (
	"container/heap"
	"sync"

	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/osdi23p228/fabric/common/util"
)

// strawman codes vvvvvvvvvvvvvvvvvvvvvvv

type ProposalOrderer struct {
	nextSequence int
	mutex        sync.Mutex
	queue        *PriorityQueue
	processCh    chan *UnpackedProposalWrapper
	trySendCh    chan struct{}
}

func NewProposalOrderer() *ProposalOrderer {
	// allocate a slice with length 0 and capacity 1024
	q := make(PriorityQueue, 0, 1024)

	po := &ProposalOrderer{
		nextSequence: 0,
		mutex:        sync.Mutex{},
		queue:        &q,
		processCh:    make(chan *UnpackedProposalWrapper, 1e5),
		trySendCh:    make(chan struct{}, 1e5),
	}

	go po.SendProposalsInOrder()

	return po
}

func (po *ProposalOrderer) SendProposalsInOrder() {
	for {
		select {
		case <-po.trySendCh:
			po.mutex.Lock()
			for po.queue.Len() > 0 && (*po.queue)[0].Sequence == po.nextSequence {
				upw := heap.Pop(po.queue).(*UnpackedProposalWrapper)
				po.nextSequence += 1
				po.processCh <- upw
			}
			po.mutex.Unlock()
		}
	}
}

func (po *ProposalOrderer) Push(upw *UnpackedProposalWrapper) {
	po.mutex.Lock()
	heap.Push(po.queue, upw)
	po.mutex.Unlock()

	po.trySendCh <- struct{}{}
}

func (po *ProposalOrderer) Pop() *UnpackedProposalWrapper {
	upw := <-po.processCh
	return upw
}

type UnpackedProposalWrapper struct {
	UnpackedProposal *UnpackedProposal
	Sequence         int
	ProposalResponse *pb.ProposalResponse
	Err              error
	doneCh           chan struct{}
}

func NewUnpackedProposalWrapper(up *UnpackedProposal) *UnpackedProposalWrapper {
	sequence, _ := util.ParseCustomTXID(up.ChannelHeader.TxId)

	upw := &UnpackedProposalWrapper{
		UnpackedProposal: up,
		Sequence:         sequence,
		doneCh:           make(chan struct{}),
	}

	return upw
}

func (upw *UnpackedProposalWrapper) Done() {
	upw.doneCh <- struct{}{}
}

func (upw *UnpackedProposalWrapper) Wait() {
	<-upw.doneCh
}

type PriorityQueue []*UnpackedProposalWrapper

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].Sequence < pq[j].Sequence
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*UnpackedProposalWrapper)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // avoid memory leak
	*pq = old[0 : n-1]
	return item
}

// strawman codes ^^^^^^^^^^^^^^^^^^^^^^^
