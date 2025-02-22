package task

type Task func()

func (t Task) Exec() {
	t()
}
