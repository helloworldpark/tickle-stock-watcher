package push

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

func (m *Manager) PushMessage(msg string, userid int64) {
	m.tasksTelegram <- func() {
		SendMessageTelegram(userid, msg)
	}
}
