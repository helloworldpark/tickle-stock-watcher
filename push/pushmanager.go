package push

import (
	"strings"

	"github.com/helloworldpark/tickle-stock-watcher/commons"
)

type msgTask = func()

// Manager manager for push
type Manager struct {
	tasksTelegram chan msgTask
}

const msgMaxLength = 4096

// NewManager creates an initialized Push Manager
func NewManager() *Manager {
	m := Manager{tasksTelegram: make(chan msgTask)}
	runPusher := func(tasks chan msgTask) {
		for k := range tasks {
			k()
		}
	}
	commons.InvokeGoroutine("push_Manager_NewManager", func() {
		runPusher(m.tasksTelegram)
	})
	return &m
}

// PushMessage pushes message to Telegram Bot
func (m *Manager) PushMessage(msg string, userid int64) {
	if len(msg) == 0 {
		return
	}

	m.tasksTelegram <- func() {
		if len(msg) < msgMaxLength {
			SendMessageTelegram(userid, msg)
			return
		}

		splits := strings.Split(msg, "\n")
		if len(splits) == 1 {
			last := 0
			for last < len(msg) {
				SendMessageTelegram(userid, msg[last:commons.MinInt(last+msgMaxLength, len(msg))])
				last += msgMaxLength
			}
		} else {
			var tmpMsg string
			for _, line := range splits {
				if len(tmpMsg)+len(line) <= msgMaxLength {
					tmpMsg += (line + "\n")
				} else {
					SendMessageTelegram(userid, tmpMsg)
					if len(line) > 0 && line[0] == ' ' {
						tmpMsg = ("*" + line[1:] + "\n")
					} else {
						tmpMsg = (line + "\n")
					}
				}
			}
			if len(tmpMsg) > 0 {
				SendMessageTelegram(userid, tmpMsg)
			}
		}
	}
}
