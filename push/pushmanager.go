package push

type msgTask = func()

// Manager manager for push
type Manager struct {
	tasksTelegram chan msgTask
}

// NewManager creates an initialized Push Manager
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

// PushMessage pushes message to Telegram Bot
func (m *Manager) PushMessage(msg string, userid int64) {
	m.tasksTelegram <- func() {
		SendMessageTelegram(userid, msg)
	}
}
