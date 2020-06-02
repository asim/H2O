package stats

const (
	defaultMean    = 10000 // ms
	defaultUpper95 = 30000 // ms
)

type endpoint struct {
	name    string
	mean    int32
	upper95 int32
}

func (e *endpoint) GetName() string {
	return e.name
}

func (e *endpoint) GetMean() int32 {
	return e.mean
}

func (e *endpoint) GetUpper95() int32 {
	return e.upper95
}
