// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package wait provides utility functions for polling, listening using Go
// channel.
package wait

import (
	"log"
	"sync"
)

type WaitResult interface {
	GetResult() interface{}
	WaitC() <-chan struct{}
}

// Wait is an interface that provides the ability to wait and trigger events that
// are associated with IDs.
type Wait interface {
	// Register waits returns a chan that waits on the given ID.
	// The chan will be triggered when Trigger is called with
	// the same ID.
	Register(id uint64) WaitResult
	// Trigger triggers the waiting chans with the given ID.
	Trigger(id uint64, x interface{})
	IsRegistered(id uint64) bool
}

type multList [16]*list

type resultData struct {
	value interface{}
	done  chan struct{}
}

func newResultData() *resultData {
	return &resultData{
		done: make(chan struct{}, 1),
	}
}

func (rd *resultData) GetResult() interface{} {
	return rd.value
}

func (rd *resultData) WaitC() <-chan struct{} {
	return rd.done
}

type list struct {
	l sync.RWMutex
	m map[uint64]*resultData
}

// New creates a Wait.
func New() Wait {
	ml := multList{}
	for i, _ := range ml {
		ml[i] = &list{
			m: make(map[uint64]*resultData),
		}
	}
	return ml
}

func (mw multList) Register(id uint64) WaitResult {
	w := mw[id%uint64(len(mw))]
	w.l.Lock()
	defer w.l.Unlock()
	rd := w.m[id]
	if rd == nil {
		rd = newResultData()
		w.m[id] = rd
	} else {
		log.Panicf("dup id %x", id)
	}
	return rd
}

func (mw multList) Trigger(id uint64, x interface{}) {
	w := mw[id%uint64(len(mw))]
	w.l.Lock()
	rd := w.m[id]
	delete(w.m, id)
	if rd != nil {
		rd.value = x
		close(rd.done)
	}
	w.l.Unlock()
}

func (mw multList) IsRegistered(id uint64) bool {
	w := mw[id%uint64(len(mw))]
	w.l.RLock()
	defer w.l.RUnlock()
	_, ok := w.m[id]
	return ok
}

type waitWithResponse struct {
	wr *resultData
}

func NewWithResponse(ch <-chan interface{}) Wait {
	return &waitWithResponse{wr: newResultData()}
}

func (w *waitWithResponse) Register(id uint64) WaitResult {
	return w.wr
}
func (w *waitWithResponse) Trigger(id uint64, x interface{}) {}
func (w *waitWithResponse) IsRegistered(id uint64) bool {
	panic("waitWithResponse.IsRegistered() shouldn't be called")
}
