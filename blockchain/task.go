package blockchain

import (
	"errors"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/types"
)

type Task struct {
	sync.Mutex
	cond     *sync.Cond
	start    int64
	end      int64
	isruning bool
	ticker   *time.Timer
	timeout  time.Duration
	cb       func()
	donelist map[int64]struct{}
}

func newTask(timeout time.Duration) *Task {
	t := &Task{}
	t.timeout = timeout
	t.ticker = time.NewTimer(t.timeout)
	t.cond = sync.NewCond(t)
	go t.tick()
	return t
}

func (t *Task) tick() {
	for {
		t.cond.L.Lock()
		for !t.isruning {
			t.cond.Wait()
		}
		t.cond.L.Unlock()
		_, ok := <-t.ticker.C
		if !ok {
			chainlog.Error("task is done", "timer is stop", t.start)
			continue
		}
		t.Lock()
		if err := t.stop(); err == nil {
			chainlog.Error("task is done", "timer is stop", t.start)
		}
		t.Unlock()
	}
}

func (t *Task) InProgress() bool {
	t.Lock()
	defer t.Unlock()
	return t.isruning
}

func (t *Task) Start(start, end int64, cb func()) error {
	t.Lock()
	defer t.Unlock()
	if t.isruning {
		return errors.New("task is runing")
	}
	if start > end {
		return types.ErrStartBigThanEnd
	}
	chainlog.Error("task start:", "start", start, "end", end)
	t.isruning = true
	t.ticker.Reset(t.timeout)
	t.start = start
	t.end = end
	t.cb = cb
	t.donelist = make(map[int64]struct{})
	t.cond.Signal()
	return nil
}

func (t *Task) Done(height int64) {
	t.Lock()
	defer t.Unlock()
	if !t.isruning {
		return
	}
	if height >= t.start && height <= t.end {
		chainlog.Error("done", "height", height)
		t.done(height)
		t.ticker.Reset(t.timeout)
	}
}

func (t *Task) stop() error {
	if !t.isruning {
		return errors.New("not runing")
	}
	t.isruning = false
	if t.cb != nil {
		go t.cb()
	}
	t.ticker.Stop()
	return nil
}

func (t *Task) done(height int64) {
	if height == t.start {
		t.start = t.start + 1
		for i := t.start; i <= t.end; i++ {
			_, ok := t.donelist[i]
			if !ok {
				break
			}
			delete(t.donelist, i)
			t.start = i + 1
			//任务完成
		}
		if t.start > t.end {
			chainlog.Error("----task is done----")
			t.stop()
		}
	}
	t.donelist[height] = struct{}{}
}
