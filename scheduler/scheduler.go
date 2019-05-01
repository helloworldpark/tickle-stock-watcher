package scheduler

import (
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
)

type stoppable interface {
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

func (task *taskTerminating) stop() {
	task.timer.Stop()
	taskMap.DeleteValue(task.tag)
}

func (task *taskTerminating) do() {
	task.todo()
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

// Cancel task schedulled by tag
func Cancel(tag string) {
	task, ok := taskMap.GetValue(tag)
	if ok {
		task.(stoppable).stop()
	}
}

// Schedule single task. Duplicated tag will overwrite the task to do.
func Schedule(tag string, after time.Duration, todo func()) {
	appendSingleTask(tag, after, todo, false)
}

// SchedulePeriodic task. Can set period, and when to start.
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
				(&task).do()
			}
		}()
	}
	if after > 0 {
		appendSingleTask(tag, after, todoAfter, true)
	} else {
		Cancel(tag)
		todoAfter()
	}
}

// SchedulePeriodicFinite task. This is for periodic but terminating tasks.
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
				(&task).do()
			}
		}()
	}
	if after > 0 {
		appendSingleTask(tag, after, todoAfter, true)
	} else {
		Cancel(tag)
		todoAfter()
	}
}

// ScheduleEveryday runs a task everyday at a given hour.
func ScheduleEveryday(tag string, startHour int, todo func()) {
	_, timeLeft := startingDate(startHour)
	SchedulePeriodic(tag, 24*time.Hour, time.Duration(timeLeft)*time.Second, todo)
}

// ScheduleWeekdays runs a task everyday at a given hour but only on weekdays.
func ScheduleWeekdays(tag string, startHour int, todo func()) {
	_, timeLeft := startingDate(startHour)
	SchedulePeriodic(tag, 24*time.Hour, time.Duration(timeLeft)*time.Second, func() {
		now := commons.Now()
		if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
			return
		}
		todo()
	})
}

func startingDate(startHour int) (time.Time, int64) {
	now := commons.Now()
	var refDate time.Time
	y, m, d := now.Date()
	if now.Hour() >= startHour {
		tmrw := now.Add(time.Hour * 24)
		y, m, d = tmrw.Date()
	}
	refDate = time.Date(y, m, d, startHour, 0, 0, 0, time.UTC)
	refDate = refDate.In(commons.AsiaSeoul)
	return refDate, refDate.Unix() - now.Unix()
}

func appendSingleTask(tag string, after time.Duration, todo func(), reuseTag bool) {
	task := taskTerminating{
		tag:   tag,
		todo:  todo,
		timer: time.NewTimer(after),
	}
	Cancel(tag)
	taskMap.SetValue(tag, &task)
	go func() {
		<-task.timer.C
		(&task).do()
		if !reuseTag {
			taskMap.DeleteValue(tag)
		}
	}()
}
