package ping

import "fmt"

func (r *Result) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"host":        r.Host,
		"ip":          r.IP,
		"sent":        r.Sent,
		"received":    r.Received,
		"packet_loss": r.PacketLoss,
		"min_rtt":     r.Min.String(),
		"max_rtt":     r.Max.String(),
		"avg_rtt":     r.Avg.String(),
		"jitter":      r.Jitter.String(),
	}
}

func (r *Result) Headers() []string {
	return []string{"Host", "IP", "Sent", "Received", "PacketLoss%", "MinRTT", "MaxRTT", "AvgRTT", "Jitter"}
}

func (r *Result) Row() []string {
	return []string{
		r.Host,
		r.IP,
		fmt.Sprintf("%d", r.Sent),
		fmt.Sprintf("%d", r.Received),
		fmt.Sprintf("%.1f", r.PacketLoss),
		r.Min.String(),
		r.Max.String(),
		r.Avg.String(),
		r.Jitter.String(),
	}
}
