package scheduler

import (
	"fmt"
	"testing"
	"time"
)

func TestScheduler(t *testing.T) {
	a := "999999"[0] - "0"[0]
	b := "삼성전자"[0] - "0"[0]
	fmt.Printf("a: %v b: %v c: %v\n", a, b, string(byte(47)))
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
	ScheduleWeekdays("task5", 11.0666666666666666666667, func() {
		fmt.Println("Hello! Task5")
	})

	timer30 := time.NewTimer(25 * time.Second)
	<-timer30.C
	fmt.Println("Test finished")
}

func TestWeekdays(t *testing.T) {
	now := time.Now()
	ttime := 11 + float64(now.Minute()+1)/60.0
	ScheduleWeekdays("test", ttime, func() {
		fmt.Println("HAHA")
	})
	timer30 := time.NewTimer(60 * time.Second)
	<-timer30.C
	fmt.Println("TestWeekdays")
}
