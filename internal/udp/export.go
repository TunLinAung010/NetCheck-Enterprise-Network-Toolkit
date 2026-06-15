package udp

import "fmt"

func (r *Result) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"host":     r.Host,
		"port":     r.Port,
		"state":    string(r.State),
		"protocol": string(r.Protocol),
		"latency":  r.Latency.String(),
	}
}

func (r *Result) Headers() []string {
	return []string{"Host", "Port", "State", "Protocol", "Latency"}
}

func (r *Result) Row() []string {
	return []string{
		r.Host,
		fmt.Sprintf("%d", r.Port),
		string(r.State),
		string(r.Protocol),
		r.Latency.String(),
	}
}
