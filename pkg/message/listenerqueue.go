package message

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

var ErrNegativeAckNotSupported = errors.New("negative acknowledgement not supported")

type (
	// ListenerProcessingQueue is used to request new messages from a broker and acknowledge completed messages in a
	// broker-specific way. It controls the specified maximum number of simultaneously processed messages by issuing
	// ProcessingTokens(). New tokens are issued after the messages are processed and there is free space in the queue.
	ListenerProcessingQueue interface {
		ProcessingTokens() <-chan struct{}
		AddProcessing(*ConsumerMessage) error
		AcknowledgeResult(context.Context, *ConsumerMessage, error) error
	}

	commitOffsetQueue struct {
		mutex            *sync.Mutex
		acknowledge      CommitOffsetStrategy
		processingTokens chan struct{}
		processingQueue  []listenerMessage
	}

	ackNackQueue struct {
		maxSize          int
		mutex            *sync.Mutex
		acknowledge      AckNackStrategy
		processingTokens chan struct{}
		processingQueue  map[*ConsumerMessage]struct{}
	}

	ackQueue struct {
		ListenerProcessingQueue
	}

	nackNotSupportedAdapter struct {
		AckStrategy
	}

	listenerMessage struct {
		Message   *ConsumerMessage
		Processed bool
	}
)

func NewCommitOffsetQueue(ack CommitOffsetStrategy, maxSize int) ListenerProcessingQueue {
	if maxSize < 1 {
		maxSize = 1
	}

	tokensCh := make(chan struct{}, maxSize)
	for range maxSize {
		tokensCh <- struct{}{}
	}

	return &commitOffsetQueue{
		mutex:            &sync.Mutex{},
		acknowledge:      ack,
		processingTokens: tokensCh,
		processingQueue:  make([]listenerMessage, 0, maxSize),
	}
}

func (q *commitOffsetQueue) ProcessingTokens() <-chan struct{} {
	return q.processingTokens
}

func (q *commitOffsetQueue) AddProcessing(msg *ConsumerMessage) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.processingQueue) == cap(q.processingQueue) {
		return fmt.Errorf("processing queue is full")
	}

	q.processingQueue = append(q.processingQueue, listenerMessage{
		Message:   msg,
		Processed: false,
	})

	return nil
}

func (q *commitOffsetQueue) AcknowledgeResult(ctx context.Context, msg *ConsumerMessage, result error) error {
	if result != nil {
		return fmt.Errorf("commit offset %w: %w", ErrNegativeAckNotSupported, result)
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	for i := range q.processingQueue {
		if q.processingQueue[i].Message == msg {
			q.processingQueue[i].Processed = true
		}
	}

	var (
		lastProcessed      *ConsumerMessage
		lastProcessedIndex = -1
	)

	for i := range q.processingQueue {
		if q.processingQueue[i].Processed {
			lastProcessedIndex = i
			v := *q.processingQueue[i].Message
			lastProcessed = &v
		} else {
			break
		}
	}
	if lastProcessedIndex == -1 {
		return nil
	}

	err := q.acknowledge.CommitOffset(ctx, lastProcessed)
	if err != nil {
		return fmt.Errorf("commit offset: %w", err)
	}

	processedItemsCount := lastProcessedIndex + 1
	for i := range len(q.processingQueue) - processedItemsCount {
		q.processingQueue[i] = q.processingQueue[i+processedItemsCount]
	}
	q.processingQueue = q.processingQueue[:len(q.processingQueue)-processedItemsCount]
	for range processedItemsCount {
		q.processingTokens <- struct{}{}
	}

	return nil
}

func NewAckNackQueue(ack AckNackStrategy, maxSize int) ListenerProcessingQueue {
	if maxSize < 1 {
		maxSize = 1
	}

	tokensCh := make(chan struct{}, maxSize)
	for range maxSize {
		tokensCh <- struct{}{}
	}

	return &ackNackQueue{
		maxSize:          maxSize,
		mutex:            &sync.Mutex{},
		acknowledge:      ack,
		processingTokens: tokensCh,
		processingQueue:  make(map[*ConsumerMessage]struct{}, maxSize),
	}
}

func (q *ackNackQueue) ProcessingTokens() <-chan struct{} {
	return q.processingTokens
}

func (q *ackNackQueue) AddProcessing(msg *ConsumerMessage) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.processingQueue) == q.maxSize {
		return fmt.Errorf("processing queue is full")
	}
	if _, ok := q.processingQueue[msg]; ok {
		return fmt.Errorf("message %s already in queue", msg.Message.ID)
	}

	q.processingQueue[msg] = struct{}{}
	return nil
}

func (q *ackNackQueue) AcknowledgeResult(ctx context.Context, msg *ConsumerMessage, result error) error {
	if result == nil {
		err := q.acknowledge.Ack(ctx, msg)
		if err != nil {
			return fmt.Errorf("ack: %w", err)
		}
	} else {
		err := q.acknowledge.Nack(ctx, msg)
		if err != nil {
			return fmt.Errorf("nack: %w", err)
		}
	}

	q.mutex.Lock()
	defer q.mutex.Unlock()

	delete(q.processingQueue, msg)
	q.processingTokens <- struct{}{}
	return nil
}

func NewAckQueue(ack AckStrategy, maxSize int) ListenerProcessingQueue {
	return &ackQueue{NewAckNackQueue(nackNotSupportedAdapter{ack}, maxSize)}
}

func (q *ackQueue) AcknowledgeResult(ctx context.Context, msg *ConsumerMessage, result error) error {
	if result != nil {
		return fmt.Errorf("ack %w: %w", ErrNegativeAckNotSupported, result)
	}

	return q.ListenerProcessingQueue.AcknowledgeResult(ctx, msg, nil)
}

func (a nackNotSupportedAdapter) Nack(context.Context, *ConsumerMessage) error {
	return ErrNegativeAckNotSupported
}
