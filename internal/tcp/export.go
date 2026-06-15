package tcp

import "fmt"

func (r *Result) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"host":    r.Host,
		"port":    r.Port,
		"state":   string(r.State),
		"latency": r.Latency.String(),
		"min":     r.Min.String(),
		"max":     r.Max.String(),
		"avg":     r.Avg.String(),
	}
}

func (r *Result) Headers() []string {
	return []string{"Host", "Port", "State", "Latency", "Min", "Max", "Avg"}
}

func (r *Result) Row() []string {
	return []string{
		r.Host,
		fmt.Sprintf("%d", r.Port),
		string(r.State),
		r.Latency.String(),
		r.Min.String(),
		r.Max.String(),
		r.Avg.String(),
	}
}
