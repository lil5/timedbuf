/*
 *   Copyright 2016 Charith Ellawala
 *
 *   Licensed under the Apache License, Version 2.0 (the "License");
 *   you may not use this file except in compliance with the License.
 *   You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *   Unless required by applicable law or agreed to in writing, software
 *   distributed under the License is distributed on an "AS IS" BASIS,
 *   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *   See the License for the specific language governing permissions and
 *   limitations under the License.
 */
package timedbuf

import (
	"sync"
	"time"
)

// TimedBuf implements a buffer that gathers items until either the buffer size or a specified time limit is reached
type TimedBuf struct {
	mu          sync.Mutex
	maxDelay    time.Duration
	lastFlushTS time.Time
	buffer      chan any
	ticker      *time.Ticker
	flushFn     func([]any)
}

func New(size int, maxDelay time.Duration, flushFn func([]any)) *TimedBuf {
	buffer := make(chan any, size)
	ticker := time.NewTicker(maxDelay)
	tb := &TimedBuf{buffer: buffer, ticker: ticker, flushFn: flushFn, lastFlushTS: time.Now(), maxDelay: maxDelay}
	tb.startLoop()
	return tb
}

func (tb *TimedBuf) startLoop() {
	go func() {
		for _ = range tb.ticker.C {
			tb.mu.Lock()
			if time.Since(tb.lastFlushTS) > tb.maxDelay {
				tb.doFlush()
			}
			tb.mu.Unlock()
		}
	}()
}

func (tb *TimedBuf) doFlush() {
	bufLen := len(tb.buffer)
	if bufLen > 0 {
		tmp := make([]any, bufLen)
		for i := 0; i < bufLen; i++ {
			tmp[i] = <-tb.buffer
		}
		tb.flushFn(tmp)
		tb.lastFlushTS = time.Now()
	}
}

func (tb *TimedBuf) Put(items ...any) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	for _, i := range items {
		select {
		case tb.buffer <- i:
		default:
			tb.doFlush()
			tb.buffer <- i
		}
	}
}

func (tb *TimedBuf) Close() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.doFlush()
	close(tb.buffer)
	tb.ticker.Stop()
}
