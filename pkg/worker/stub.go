package worker

type poolStub struct{}

func NewPoolStub() Pool {
	return poolStub{}
}

func (s poolStub) Do(job Job) {
	job()
}

func (s poolStub) Wait() {}
