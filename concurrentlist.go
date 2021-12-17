package list

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

type List interface {
	Insert(v int) bool
	Delete(v int) bool
	Contains(v int) bool
	Range(fn func(v int) bool)
	Len() int
}

func NewInt() List {
	return &intList{
		head: unsafe.Pointer(new(node)),
	}
}

type node struct {
	marked bool
	val    int
	next   unsafe.Pointer
	mu     sync.Mutex
}

type intList struct {
	head unsafe.Pointer
	n    int32
}

func (l *intList) Len() int {
	return int(l.n)
}

func (l *intList) Range(fn func(v int) bool) {
	p := load(&load(&l.head).next)
	for p != nil {
		if !fn(p.val) {
			break
		}
		p = load(&p.next)
	}
}

func (l *intList) Insert(v int) bool {
RETRY:
	a := load(&l.head)
	var b *node
	for {
		b = load(&a.next)
		if b == nil || b.val > v {
			break
		}
		if b.val == v {
			return false
		}
		a = b
	}

	a.mu.Lock()
	if load(&a.next) != b {
		a.mu.Unlock()
		goto RETRY
	}

	x := &node{val: v, next: unsafe.Pointer(b)}
	atomic.StorePointer(&a.next, unsafe.Pointer(x))
	a.mu.Unlock()

	atomic.AddInt32(&l.n, 1)
	return true
}

func (l *intList) Delete(v int) bool {
RETRY:
	a := load(&l.head)
	var b *node
	for {
		b = load(&a.next)
		if b == nil || b.val > v {
			return false
		}
		if b.val == v {
			break
		}
		a = b
	}

	b.mu.Lock()
	if b.marked {
		b.mu.Unlock()
		goto RETRY
	}

	a.mu.Lock()
	if load(&a.next) != b || a.marked {
		a.mu.Unlock()
		b.mu.Unlock()
		goto RETRY
	}

	b.marked = true
	atomic.StorePointer(&a.next, b.next)
	a.mu.Unlock()
	b.mu.Unlock()

	atomic.AddInt32(&l.n, -1)
	return true
}

func (l *intList) Contains(v int) bool {
	p := load(&load(&l.head).next)
	for p != nil {
		if p.val == v {
			return true
		}
		p = load(&p.next)
	}
	return false
}

func load(p *unsafe.Pointer) *node {
	return (*node)(atomic.LoadPointer(p))
}
