package scheduler

import (
	"fmt"
	"time"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/logger"
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
			(&task).do()
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
			(&task).do()
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
func ScheduleEveryday(tag string, startHour float64, todo func()) {
	_, timeLeft := startingDate(startHour)
	SchedulePeriodic(tag, 24*time.Hour, time.Duration(timeLeft), todo)
}

// ScheduleWeekdays runs a task everyday at a given hour but only on weekdays.
func ScheduleWeekdays(tag string, startHour float64, todo func()) {
	_, timeLeft := startingDate(startHour)
	SchedulePeriodic(tag, 24*time.Hour, time.Duration(timeLeft), func() {
		weekday := commons.Now().Weekday()
		if weekday == time.Saturday || weekday == time.Sunday {
			return
		}
		todo()
	})
}

func startingDate(startHour float64) (time.Time, int64) {
	now := commons.Now()
	var refDate time.Time
	y, m, d := now.Date()
	if float64(now.Hour()) >= startHour {
		tmrw := now.Add(time.Hour * 24)
		y, m, d = tmrw.Date()
	}
	h := int(startHour)
	remainder := startHour - float64(h)
	i := int(60.0 * remainder)
	s := int(3600.0 * remainder - 60.0 * float64(i))
	refDate = time.Date(y, m, d, h, i, s, 0, commons.AsiaSeoul)
	return refDate, refDate.UnixNano() - now.UnixNano()
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
		logger.Info("[Scheduler] Appended task %s: after %f minutes", tag, float64(after)/float64(time.Minute))
		<-task.timer.C
		(&task).do()
		if !reuseTag {
			taskMap.DeleteValue(tag)
		}
	}()
}
