package push

import (
	"github.com/helloworldpark/tickle-stock-watcher/commons"
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

type msgTask = func()

type Manager struct {
	tasksTelegram chan msgTask
	tasksLine     chan msgTask
	tasksKakao    chan msgTask
}

func NewManager() *Manager {
	m := Manager{
		tasksTelegram: make(chan msgTask),
		tasksLine:     make(chan msgTask),
		tasksKakao:    make(chan msgTask),
	}
	runPusher := func(tasks chan msgTask) {
		for k := range tasks {
			k()
		}
	}
	go runPusher(m.tasksTelegram)
	go runPusher(m.tasksLine)
	go runPusher(m.tasksKakao)
	return &m
}

func (m *Manager) PushMessage(msg string, user structs.User) {
	if user.TokenTelegram != "" {
		m.tasksTelegram <- func() {
			SendMessageTelegram(commons.GetInt64(user.TokenTelegram), msg)
		}
	}
}
