package audit

import (
	"net"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type AuditEvent struct {
	Timestamp int64    `json:"ts"`
	Metrics   []string `json:"metrics"`
	IPAddress string   `json:"ip_address"`
}

type Observer interface {
	Update(event AuditEvent) error
	GetID() string
}

type Publisher struct {
	observers map[string]Observer
	logger    *zap.Logger
}

func NewPublisher(logger *zap.Logger) *Publisher {
	return &Publisher{
		observers: make(map[string]Observer),
		logger:    logger,
	}
}

func (p *Publisher) Subscribe(observer Observer) {
	p.observers[observer.GetID()] = observer
	p.logger.Info("audit observer subscribed", zap.String("id", observer.GetID()))
}

func (p *Publisher) Unsubscribe(observer Observer) {
	delete(p.observers, observer.GetID())
	p.logger.Info("audit observer unsubscribed", zap.String("id", observer.GetID()))
}

func (p *Publisher) Notify(event AuditEvent) {
	for id, observer := range p.observers {
		go func(id string, obs Observer, evt AuditEvent) {
			if err := obs.Update(evt); err != nil {
				p.logger.Error("failed to notify audit observer",
					zap.String("observer_id", id),
					zap.Error(err))
			}
		}(id, observer, event)
	}
}

func CreateAuditEvent(r *http.Request, metrics []string) AuditEvent {
	ipAddress := getClientIP(r)
	return AuditEvent{
		Timestamp: time.Now().Unix(),
		Metrics:   metrics,
		IPAddress: ipAddress,
	}
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := parseXForwardedFor(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func parseXForwardedFor(header string) []string {
	var ips []string
	for _, ip := range strings.Split(header, ",") {
		ip = strings.TrimSpace(ip)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	return ips
}
