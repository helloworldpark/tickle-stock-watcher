package scheduler

import (
	"fmt"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	task1 := func() {
		fmt.Println("Task Terminating")
	}
	Schedule("task1", 2*time.Second, task1)
	n := 5
	task2 := func() {
		fmt.Printf("Task Repeating: %d left\n", n)
		n--
	}
	SchedulePeriodicFinite("task2", 1*time.Second, 3*time.Second, int64(n), task2)
	task3 := func() {
		fmt.Println("Task Forever")
	}
	SchedulePeriodic("task3", 2*time.Second, 0, task3)
	task4 := func() {
		fmt.Println("Task Forever finish")
		Cancel("task3")
	}
	Schedule("task4", 11*time.Second, task4)
	Schedule("task4", 0, func() {
		fmt.Println("Task Forever finish cancel")
	})

	timer30 := time.NewTimer(20 * time.Second)
	<-timer30.C
	fmt.Println("Test finished")
}
