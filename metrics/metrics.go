package metrics

type MetricsList struct {
	ReceivedMessages     metric
	SentMessages         metric
	SentMessageError     metric
	ExpiredMessages      metric
	Retransmissions      metric
	SuccessfulHandshakes metric
	Sessions             metric
	Observers            metric
	ObservingTime        metric
}

type metric int64

func (m *metric) Inc() {
	*m++
}

func (m *metric) Dec() {
	*m--
}

func (m *metric) Count() int64 {
	return int64(*m)
}

func (m *metric) Set(c int64) {
	cm := metric(c)
	m = &cm
}
