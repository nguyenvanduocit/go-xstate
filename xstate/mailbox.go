package xstate

// ---- mailbox (Mailbox.ts) ----

type mailboxItem struct {
	value Event
	next  *mailboxItem
}

type mailbox struct {
	process func(Event)
	active  bool
	current *mailboxItem
	last    *mailboxItem
}

func newMailbox(process func(Event)) *mailbox { return &mailbox{process: process} }

func (m *mailbox) start() {
	m.active = true
	m.flush()
}

func (m *mailbox) clear() {
	if m.current != nil {
		m.current.next = nil
		m.last = m.current
	}
}

func (m *mailbox) enqueue(ev Event) {
	item := &mailboxItem{value: ev}
	if m.current != nil {
		m.last.next = item
		m.last = item
		return
	}
	m.current = item
	m.last = item
	if m.active {
		m.flush()
	}
}

func (m *mailbox) flush() {
	for m.current != nil {
		consumed := m.current
		m.process(consumed.value)
		m.current = consumed.next
	}
	m.last = nil
}
