package coalago

type MetricsList struct {
	ReceivedMessages     iMetric
	SentMessages         iMetric
	SentMessageError     iMetric
	ExpiredMessages      iMetric
	Retransmissions      iMetric
	SuccessfulHandshakes iMetric
	SessionsRate         iMetric
	Sessions             iMetric
	ProxiedMessages      iMetric
	Observers            iMetric
	ObservingTime        iMetric
}

func NewMetricList(c *Coala) *MetricsList {
	m := &MetricsList{
		ReceivedMessages:     new(metric),
		SentMessages:         new(metric),
		SentMessageError:     new(metric),
		ExpiredMessages:      new(metric),
		Retransmissions:      new(metric),
		SuccessfulHandshakes: new(metric),
		SessionsRate:         new(metric),
		Sessions: sessionMetric{
			c: c,
		},
		ProxiedMessages: new(metric),
		Observers:       new(metric),
		ObservingTime:   new(metric),
	}
	return m
}

type iMetric interface {
	Inc()
	Dec()
	Count() int64
	Set(int64)
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

// Implementation iMetric interface for get session state
type sessionMetric struct {
	c *Coala
}

func (s sessionMetric) Inc() {
}

func (s sessionMetric) Dec() {
}

func (s sessionMetric) Count() int64 {
	return int64(s.c.Sessions.ItemCount())
}

func (s sessionMetric) Set(c int64) {

}
