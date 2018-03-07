package alert

import (
	"sync"
	"sync/atomic"

	"github.com/influxdata/kapacitor/models"
)

type InhibitorLookup struct {
	mu         sync.RWMutex
	inhibitors map[string][]*Inhibitor
}

func NewInhibitorLookup() *InhibitorLookup {
	return &InhibitorLookup{
		inhibitors: make(map[string][]*Inhibitor),
	}
}

func (l *InhibitorLookup) IsInhibited(name string, tags models.Tags) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	for _, i := range l.inhibitors[name] {
		if i.IsInhibited(name, tags) {
			return true
		}
	}
	return false
}

func (l *InhibitorLookup) AddInhibitor(in *Inhibitor) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inhibitors[in.name] = append(l.inhibitors[in.name], in)
}

func (l *InhibitorLookup) RemoveInhibitor(in *Inhibitor) {
	l.mu.Lock()
	defer l.mu.Unlock()

	inhibitors := l.inhibitors[in.name]
	for i := range inhibitors {
		if inhibitors[i] == in {
			inhibitors = inhibitors[:i]
			inhibitors = append(inhibitors, inhibitors[i+1:]...)
			l.inhibitors[in.name] = inhibitors
			break
		}
	}
}

// Inhibitor tracks whether and alert + tag set have been inhibited
type Inhibitor struct {
	name      string
	tags      models.Tags
	inhibited int32
}

func NewInhibitor(name string, tags models.Tags) *Inhibitor {
	return &Inhibitor{
		name:      name,
		tags:      tags,
		inhibited: 0,
	}
}

func (i *Inhibitor) Set(inhibited bool) {
	v := int32(0)
	if inhibited {
		v = 1
	}
	atomic.StoreInt32(&i.inhibited, v)
}

func (i *Inhibitor) IsInhibited(name string, tags models.Tags) bool {
	if atomic.LoadInt32(&i.inhibited) == 0 {
		return false
	}
	return i.isMatch(name, tags)
}

func (i *Inhibitor) isMatch(name string, tags models.Tags) bool {
	if name != i.name {
		return false
	}
	for k, v := range i.tags {
		if tags[k] != v {
			return false
		}
	}
	return true
}
