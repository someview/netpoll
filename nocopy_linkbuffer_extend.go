package netpoll

import (
	"errors"
	"fmt"
	"sync"
)

func (b *LinkBuffer) SliceInto(n int, r Reader) (err error) {
	var p, ok = r.(*LinkBuffer)
	if !ok {
		return errors.New("unsupported writer which is not LinkBuffer")
	}

	if n <= 0 {
		return nil
	}
	// check whether enough or not.
	if b.Len() < n {
		return fmt.Errorf("link buffer readv[%d] not enough", n)
	}
	b.recalLen(-n) // re-cal length

	if !p.IsEmpty() {
		return fmt.Errorf("dst buffer is not empty")
	}
	p.length = int64(n)

	defer func() {
		// set to read-only
		p.flush = p.flush.next
		p.write = p.flush
	}()

	// single node
	if b.isSingleNode(n) {
		node := b.read.Refer(n)
		p.head, p.read, p.flush = node, node, node
		return nil
	}
	// multiple nodes
	var l = b.read.Len()
	node := b.read.Refer(l)
	b.read = b.read.next

	p.head, p.read, p.flush = node, node, node
	for ack := n - l; ack > 0; ack = ack - l {
		l = b.read.Len()
		if l >= ack {
			p.flush.next = b.read.Refer(ack)
			p.flush = p.flush.next
			break
		} else if l > 0 {
			p.flush.next = b.read.Refer(l)
			p.flush = p.flush.next
		}
		b.read = b.read.next
	}
	return b.Release()
}

// Reuse 将已关闭或者是Append到其他LinkBuffer后的LinkBuffer重新启用
func (b *LinkBuffer) Reuse(size ...int) {
	if b.head == nil {
		b.Initialize(size...)
	}
	b.Initialize(size...)
}

// Initialize 初始化LinkBuffer
func (b *LinkBuffer) Initialize(size ...int) {
	var l int
	if len(size) > 0 {
		l = size[0]
	}
	var node = newLinkBufferNode(l)
	b.head, b.read, b.flush, b.write = node, node, node, node
}

var linkBufferPool = sync.Pool{New: func() any {
	return NewLinkBuffer()
}}

func NewSizedLinkBuffer(size int) *LinkBuffer {
	lb := linkBufferPool.Get().(*LinkBuffer)
	lb.Reuse(size)
	return lb
}

// Recycle 回收AsyncLinkBuffer
func (b *LinkBuffer) Recycle() {
	b.Release()
	linkBufferPool.Put(b)
}
