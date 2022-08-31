package ssdhelpers

import "context"

type Set struct {
	items       *Node
	vectorForID VectorForID
	distance    DistanceFunction
	center      []float32
	capacity    int
	size        int
}

type Node struct {
	data  IndexAndDistance
	left  *Node
	right *Node
}

type IndexAndDistance struct {
	index    uint64
	distance float32
	visited  bool
}

func NewSet(capacity int, vectorForID VectorForID, distance DistanceFunction, center []float32) *Set {
	return &Set{
		items:       nil,
		vectorForID: vectorForID,
		distance:    distance,
		center:      center,
		capacity:    capacity,
		size:        0,
	}
}

func (s *Set) Add(x uint64) *Set {
	vec, _ := s.vectorForID(context.Background(), x)
	dist := s.distance(vec, s.center)

	if s.size == s.capacity {
		if !s.RemoveLastIfBigger(dist) {
			return s
		}
		s.size--
	}
	s.size++

	data := IndexAndDistance{
		index:    x,
		distance: dist,
		visited:  false,
	}

	if s.items == nil {
		s.items = &Node{
			left:  nil,
			right: nil,
			data:  data,
		}
		return s
	}

	s.items.Add(data)
	return s
}

func (n *Node) Add(data IndexAndDistance) {
	if n.data.index == data.index {
		return
	}
	if n.data.distance > data.distance {
		if n.left == nil {
			n.left = &Node{
				left:  nil,
				right: nil,
				data:  data,
			}
			return
		}
		n.left.Add(data)
		return
	}

	if n.right == nil {
		n.right = &Node{
			left:  nil,
			right: nil,
			data:  data,
		}
		return
	}
	n.right.Add(data)
}

func (s *Set) RemoveLastIfBigger(dist float32) bool {
	last, parent := s.items.Last(nil)
	if last.data.distance < dist {
		return false
	}
	if parent == nil {
		s.items = s.items.left
		return true
	}
	parent.right = last.left
	return true
}

func (n *Node) Last(parent *Node) (*Node, *Node) {
	if n.right == nil {
		return n, parent
	}
	return n.right.Last(n)
}

func (s *Set) NotVisited() bool {
	return s.items.NotVisited()
}

func (n *Node) NotVisited() bool {
	if !n.data.visited {
		return true
	}
	return (n.left != nil && n.left.NotVisited()) || (n.right != nil && n.right.NotVisited())
}

func (s *Set) AddRange(indices []uint64) *Set {
	for _, item := range indices {
		s.Add(item)
	}
	return s
}

func (s *Set) Size() int {
	return s.size
}

func (s *Set) Top() uint64 {
	x, _ := s.items.Top()
	return x
}

func (n *Node) Top() (uint64, bool) {
	if n.left != nil {
		index, found := n.left.Top()
		if found {
			return index, found
		}
	}
	if !n.data.visited {
		n.data.visited = true
		return n.data.index, true
	}
	if n.right != nil {
		return n.right.Top()
	}
	return 0, false
}

func (s *Set) Elements() []uint64 {
	res := make([]uint64, s.size)
	i := s.items.Elements(res, 0)
	return res[:i]
}

func (n *Node) Elements(buffer []uint64, offset int) int {
	if n.left != nil {
		offset = n.left.Elements(buffer, offset)
	}
	buffer[offset] = n.data.index
	offset++
	if n.right != nil {
		offset = n.right.Elements(buffer, offset)
	}
	return offset
}