package push

import (
	"github.com/helloworldpark/tickle-stock-watcher/structs"
)

type msgTask = func()

type Manager struct {
	tasksTelegram chan msgTask
}

func NewManager() *Manager {
	m := Manager{tasksTelegram: make(chan msgTask)}
	runPusher := func(tasks chan msgTask) {
		for k := range tasks {
			k()
		}
	}
	go runPusher(m.tasksTelegram)
	return &m
}

func (m *Manager) PushMessage(msg string, user structs.User) {
	m.tasksTelegram <- func() {
		SendMessageTelegram(user.UserID, msg)
	}
}
