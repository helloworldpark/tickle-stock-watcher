package scheduler

import (
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
)

var aa = time.Now()

type stoppable interface {
	getTag() string
	stop()
	do()
}

type taskTerminating struct {
	tag   string
	todo  func()
	timer *time.Timer
}

type taskPeriodic struct {
	tag    string
	todo   func()
	ticker *time.Ticker
}

type taskPeriodicFinite struct {
	taskPeriodic
	counter int64
}

var taskMap = commons.NewConcurrentMap()

func (task *taskTerminating) getTag() string {
	return task.tag
}

func (task *taskTerminating) stop() {
	task.timer.Stop()
	taskMap.DeleteValue(task.tag)
}

func (task *taskTerminating) do() {
	task.todo()
}

func (task *taskPeriodic) getTag() string {
	return task.tag
}

func (task *taskPeriodic) stop() {
	task.ticker.Stop()
	taskMap.DeleteValue(task.tag)
}

func (task *taskPeriodic) do() {
	task.todo()
}

func (task *taskPeriodicFinite) do() {
	if task.counter > 0 {
		task.taskPeriodic.do()
		task.counter--
	}
	if task.counter <= 0 {
		task.stop()
	}
}

func Cancel(tag string) {
	task, ok := taskMap.GetValue(tag)
	if !ok {
		return
	}
	task.(stoppable).stop()
}

func Schedule(tag string, after time.Duration, todo func()) {
	appendSingleTask(tag, after, todo, false)
}

func SchedulePeriodic(tag string, period, after time.Duration, todo func()) {
	todoAfter := func() {
		task := taskPeriodic{
			tag:    tag,
			todo:   todo,
			ticker: time.NewTicker(period),
		}
		taskMap.SetValue(tag, &task)
		go func() {
			for range task.ticker.C {
				task.todo()
			}
		}()
	}
	if after > 0 {
		appendSingleTask(tag, after, todoAfter, true)
	} else {
		todoAfter()
	}
}

func SchedulePeriodicFinite(tag string, period, after time.Duration, n int64, todo func()) {
	todoAfter := func() {
		task := taskPeriodicFinite{
			taskPeriodic: taskPeriodic{
				tag:    tag,
				todo:   todo,
				ticker: time.NewTicker(period),
			},
			counter: n,
		}
		taskMap.SetValue(tag, &task)
		go func() {
			for range task.ticker.C {
				task.todo()
			}
		}()
	}
	if after > 0 {
		appendSingleTask(tag, after, todoAfter, true)
	} else {
		todoAfter()
	}
}

func appendSingleTask(tag string, after time.Duration, todo func(), reuseTag bool) {
	task := taskTerminating{
		tag:   tag,
		todo:  todo,
		timer: time.NewTimer(after),
	}
	taskMap.SetValue(task.getTag(), task)
	go func() {
		<-task.timer.C
		task.do()
		if !reuseTag {
			taskMap.DeleteValue(tag)
		}
	}()
}
